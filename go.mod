module github.com/buptczq/WinCryptSSHAgent

go 1.19

require (
	github.com/Microsoft/go-winio v0.4.16
	github.com/bi-zone/wmi v1.1.4
	github.com/fullsailor/pkcs7 v0.0.0-20190404230743-d7302db945fa
	github.com/jessevdk/go-flags v1.5.0
	github.com/kayrus/putty v1.0.4
	github.com/linuxkit/virtsock v0.0.0-20180830132707-8e79449dea07
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794
	golang.org/x/crypto v0.11.0
	golang.org/x/sys v0.10.0
)

require (
	github.com/bi-zone/go-ole v1.2.5 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/scjalliance/comshim v0.0.0-20190308082608-cf06d2532c4e // indirect
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
)

replace github.com/Microsoft/go-winio v0.4.16 => github.com/buptczq/go-winio v0.4.16-1
