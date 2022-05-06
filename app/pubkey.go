package app

import (
	"context"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/md5"
	"crypto/rsa"
	"fmt"
	"io"

	"github.com/kayrus/putty"
	"github.com/lxn/walk"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	. "github.com/lxn/walk/declarative"
)

type PuttyKey struct {
	key            *agent.Key
	bitlen         int
	pubfingerprint string
	checked        bool
}

func NewPuttyKey(_key *agent.Key) (*PuttyKey, error) {
	puttyKey := &putty.Key{Algo: _key.Type(), PublicKey: _key.Blob}
	pubkey, _ := puttyKey.ParseRawPublicKey()
	var lpubkeylen int = -1
	switch key := pubkey.(type) {
	case *rsa.PublicKey:
		lpubkeylen = key.N.BitLen()

	case *ecdsa.PublicKey:
		lpubkeylen = key.Params().BitSize

	case *dsa.PublicKey:
		lpubkeylen = key.P.BitLen()
	}

	fp := md5.Sum(_key.Blob)
	lsfp := ""

	for i, b := range fp {
		lsfp += fmt.Sprintf("%02x", b)
		if i < len(fp)-1 {
			lsfp += ":"
		}
	}

	return &PuttyKey{_key, lpubkeylen, lsfp, false}, nil
}

type PuttyKeysModel struct {
	walk.TableModelBase
	ag    agent.Agent
	items []*PuttyKey
}

func NewPuttyKeysModel(_ag agent.Agent) *PuttyKeysModel {
	keysmodel := &PuttyKeysModel{ag: _ag, items: []*PuttyKey{}}

	keys, _ := _ag.List()
	for _, lkey := range keys {
		lagentkey, err := NewPuttyKey(lkey)
		if err != nil {
			continue
		}

		keysmodel.items = append(keysmodel.items, lagentkey)
	}

	return keysmodel
}

func (m *PuttyKeysModel) RowCount() int {
	return len(m.items)
}

func (m *PuttyKeysModel) Value(row, col int) interface{} {
	item := m.items[row]

	switch col {
	case 0:
		return item.key.Type()

	case 1:
		return item.bitlen

	case 2:
		return item.pubfingerprint

	case 3:
		return item.key.Comment
	}

	panic("unexpected col")
}

func (m *PuttyKeysModel) Checked(row int) bool {
	return m.items[row].checked
}

// Called by the TableView when the user toggled the check box of a given row.
func (m *PuttyKeysModel) SetChecked(row int, checked bool) error {
	m.items[row].checked = checked

	return nil
}

func getKey(keyfile string) (*agent.Key, *agent.AddedKey, error) {
	puttyKey, err := putty.NewFromFile(keyfile)
	if err != nil {
		return nil, nil, err
	}

	if puttyKey.Encryption != "none" {
		return nil, nil, fmt.Errorf("encrypted keys are unsupported")
	}

	privkey, err := puttyKey.ParseRawPrivateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	sshkey := &agent.Key{Format: puttyKey.Algo, Comment: puttyKey.Comment, Blob: puttyKey.PublicKey}
	return sshkey, &agent.AddedKey{PrivateKey: privkey, Comment: puttyKey.Comment}, nil
}

func (m *PuttyKeysModel) AddItem(keyfile string) error {
	sshkey, privkey, err := getKey(keyfile)
	if err != nil {
		return err
	}

	item, err := NewPuttyKey(sshkey)
	if err != nil {
		return err
	}

	m.items = append(m.items, item)
	m.ag.Add(*privkey)
	m.PublishRowsReset()

	return nil
}

func (m *PuttyKeysModel) RemoveItem(itemIndex int) error {
	if itemIndex < 0 {
		return nil
	}

	sshpubKey, err := ssh.ParsePublicKey(m.items[itemIndex].key.Blob)
	if err != nil {
		return err
	}

	err = m.ag.Remove(sshpubKey)
	if err != nil {
		return err
	}

	m.items = append(m.items[:itemIndex], m.items[itemIndex+1:]...)
	m.PublishRowsRemoved(itemIndex, itemIndex)

	return nil
}

type PubKeyView struct {
	ag     agent.Agent
	lldldg *walk.Dialog
	tv     *walk.TableView
}

func (s *PubKeyView) Run(ctx context.Context, handler func(conn io.ReadWriteCloser)) error {
	s.ag = ctx.Value("agent").(agent.Agent)
	return nil
}

func (*PubKeyView) AppId() AppId {
	return APP_PUBKEY
}

func (s *PubKeyView) Menu(ni *walk.NotifyIcon) {
	laction := walk.NewAction()
	laction.SetDefault(true)

	ni.ContextMenu().Actions().Add(laction)
	laction.SetText("View Keys")
	laction.Triggered().Attach(func() {
		s.onClick()
	})

	laddaction := walk.NewAction()

	ni.ContextMenu().Actions().Add(laddaction)
	laddaction.SetText("Add Key")
	laddaction.Triggered().Attach(func() {
		s.onClickAdd()
	})

	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
}

func (s *PubKeyView) onClick() {
	if s.lldldg != nil {
		s.lldldg.SetFocus()
	}

	ico, _ := walk.NewIconFromResourceId(2)
	keysmodel := NewPuttyKeysModel(s.ag)
	Dialog{
		Title:      "PAgent Key List",
		MinSize:    Size{600, 400},
		AssignTo:   &s.lldldg,
		Persistent: true,
		Icon:       ico,
		Layout:     VBox{},
		Children: []Widget{
			TableView{
				AssignTo:         &s.tv,
				HeaderHidden:     true,
				AlternatingRowBG: false,
				CheckBoxes:       false,
				ColumnsOrderable: false,
				MultiSelection:   false,
				Columns: []TableViewColumn{
					{Title: "Key Type"},
					{Title: "PubKey Length", Width: 50},
					{Title: "PubKey FingerPrint"},
					{Title: "Key Description", Width: 150},
				},
				Model: keysmodel,
				OnBoundsChanged: func() {
					b := s.tv.Bounds()
					c := s.tv.Columns()

					lwidth := b.Width - c.At(0).Width() - c.At(1).Width() - c.At(3).Width()
					c.At(2).SetWidth(lwidth)
				},
			},
			Composite{
				Layout: Grid{Columns: 2, Alignment: AlignHCenterVCenter},
				Children: []Widget{
					PushButton{
						Text: "Add Key",
						OnClicked: func() {
							dlg := new(walk.FileDialog)

							dlg.FilePath = "d:\\src\\walk\\examples\\tableview"
							dlg.Filter = "Putty ppk Files *.ppk"
							dlg.Title = "Select an key file"

							if ok, err := dlg.ShowOpen(s.lldldg); err != nil {
								return
							} else if !ok {
								return
							}

							err := keysmodel.AddItem(dlg.FilePath)
							if err != nil {
								walk.MsgBox(s.lldldg, "Can't add key", err.Error(), walk.MsgBoxIconError)
							}
						},
					},
					PushButton{
						Text: "Remove Key",
						OnClicked: func() {
							err := keysmodel.RemoveItem(s.tv.CurrentIndex())
							if err != nil {
								walk.MsgBox(s.lldldg, "Can't remove key", err.Error(), walk.MsgBoxIconError)
							}
						},
					},
				},
			},
		},
	}.Create(nil)

	s.lldldg.Show()
	s.lldldg.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		s.lldldg = nil
	})
}

func (s *PubKeyView) onClickAdd() {
	dlg := new(walk.FileDialog)
	dlg.FilePath = "d:\\src\\walk\\examples\\tableview"
	dlg.Filter = "Putty ppk Files *.ppk"
	dlg.Title = "Select an key file"

	if s.lldldg != nil {
		if ok, err := dlg.ShowOpen(s.lldldg); err != nil {
			return
		} else if !ok {
			return
		}

		err := s.tv.Model().(*PuttyKeysModel).AddItem(dlg.FilePath)
		if err != nil {
			walk.MsgBox(s.lldldg, "Can't add key", err.Error(), walk.MsgBoxIconError)
		}
	} else {
		if ok, err := dlg.ShowOpen(nil); err != nil {
			return
		} else if !ok {
			return
		}

		_, privkey, err := getKey(dlg.FilePath)
		if err != nil {
			walk.MsgBox(s.lldldg, "Can't add key", err.Error(), walk.MsgBoxIconError)
		}

		s.ag.Add(*privkey)
	}
}
