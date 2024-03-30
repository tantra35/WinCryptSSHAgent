@echo off
setlocal

set GOROOT=%MYTOOLSPATH%\go-1.21.x
set GOPATH=%MYLIBSPATH%\Golang
set PATH=%GOROOT%\bin;%GOPATH%\bin;%PATH%

rsrc -manifest %CD%\WinCryptSSHAgent.exe.manifest -ico %CD%\assets\icon.ico -o rsrc.syso
