@echo off

set GOROOT=c:\Tools\go-1.19.x
set GOPATH=c:\Lib\Golang
set PATH=%GOROOT%\bin;%PATH%

go mod tidy -v
