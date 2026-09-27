@echo off

set GOROOT=%MYTOOLSPATH%\go-1.23.x
set GOPATH=%MYLIBSPATH%\Golang
set PATH=%GOROOT%\bin;%PATH%

rem go get github.com/lxn/walk
rem go get -u github.com/kayrus/putty
rem go get -u github.com/jessevdk/go-flags
go get -u golang.org/x/crypto
go get -u github.com/Microsoft/go-winio
go get -u github.com/bi-zone/wmi
