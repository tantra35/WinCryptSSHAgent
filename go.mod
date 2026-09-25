module github.com/buptczq/WinCryptSSHAgent

go 1.23.0

replace github.com/Microsoft/go-winio => github.com/buptczq/go-winio v0.4.16-1

replace github.com/lxn/walk => github.com/tantra35/walk v0.0.0-20240330122720-676edb3df880

require (
	github.com/Microsoft/go-winio v0.4.16
	github.com/bi-zone/wmi v1.1.4
	github.com/fullsailor/pkcs7 v0.0.0-20190404230743-d7302db945fa
	github.com/kayrus/putty v1.0.5
	github.com/linuxkit/virtsock v0.0.0-20180830132707-8e79449dea07
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	golang.org/x/crypto v0.38.0
	golang.org/x/sys v0.33.0
)

require (
	github.com/bi-zone/go-ole v1.2.5 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/scjalliance/comshim v0.0.0-20190308082608-cf06d2532c4e // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/text v0.28.0 // indirect
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
)
