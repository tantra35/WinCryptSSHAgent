@echo off

set GOROOT=c:\Tools\go-1.17.x
set GOPATH=c:\Lib\Golang
set PATH=%GOROOT%\bin;%PATH%

go get github.com/lxn/walk
go get github.com/kayrus/putty
go get github.com/jessevdk/go-flags
