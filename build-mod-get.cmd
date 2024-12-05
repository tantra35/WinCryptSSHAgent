@echo off

set GOROOT=%MYTOOLSPATH%\go-1.21.x
set GOPATH=%MYLIBSPATH%\Golang
set PATH=%GOROOT%\bin;%PATH%

go get github.com/lxn/walk
go get -u github.com/kayrus/putty
go get -u github.com/jessevdk/go-flags
go get -u golang.org/x/crypto