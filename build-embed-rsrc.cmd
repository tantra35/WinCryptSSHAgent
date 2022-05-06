@echo off

set GOROOT=c:\Tools\go-1.17.x
set GOPATH=c:\Lib\Golang
set PATH=%GOROOT%\bin;%GOPATH%\bin;%PATH%

rsrc -manifest %CD%\WinCryptSSHAgent.exe.manifest -ico %CD%\assets\icon.ico -o rsrc.syso
