@echo off

set GOROOT=%MYTOOLSPATH%\go-1.23.x
set GOPATH=%MYLIBSPATH%\Golang
set PATH=%GOROOT%\bin;%PATH%

go mod tidy -v
