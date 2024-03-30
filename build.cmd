@echo off
setlocal

set GOROOT=%MYTOOLSPATH%\go-1.21.x
set GOPATH=%MYLIBSPATH%\Golang
set PATH=%GOROOT%\bin;%PATH%

call build-embed-rsrc.cmd

go build -ldflags="-H windowsgui"
rem go build
