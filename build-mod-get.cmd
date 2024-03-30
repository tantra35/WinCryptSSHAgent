@echo off

set GOROOT=c:\Tools\go-1.19.x
set GOPATH=c:\Lib\Golang
set PATH=%GOROOT%\bin;%PATH%

go get github.com/lxn/walk
go get -u github.com/kayrus/putty
go get -u github.com/jessevdk/go-flags
go get -u golang.org/x/crypto