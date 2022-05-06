@echo off

set GOROOT=c:\Tools\go-1.17.x
set GOPATH=c:\Lib\Golang
set PATH=%GOROOT%\bin;%PATH%

call build-embed-rsrc.cmd

go build -ldflags="-H windowsgui"
rem go build
