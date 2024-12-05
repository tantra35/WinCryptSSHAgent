@echo off

set GOROOT=%MYTOOLSPATH%\go-1.21.x
set GOPATH=%MYLIBSPATH%\Golang
set PATH=%GOROOT%\bin;%PATH%

go mod tidy -v
