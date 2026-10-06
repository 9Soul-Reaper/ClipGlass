#!/bin/sh
# 在任何系统上交叉编译 Windows 版(仅需 Go 1.24+,无第三方依赖)
set -e
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H=windowsgui -s -w" -o ClipGlass.exe .
sha256sum ClipGlass.exe | tee ClipGlass.exe.sha256
