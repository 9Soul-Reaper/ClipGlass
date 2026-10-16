#!/bin/sh
# 在装有 Go 的机器上构建 Windows 版:
#   1) 可选:把 ffmpeg.exe 放到本目录,即可内嵌为单文件(推荐)
#   2) 不放也可以编译,运行时会用同目录或 PATH 里的 ffmpeg
# 用法: sh build.sh          -> dist/VideoToAudio.exe (不含内嵌 ffmpeg)
#       sh build-embed.sh    -> dist/VideoToAudio.exe (含内嵌 ffmpeg)
set -e
mkdir -p dist
CGO_ENABLED=0 go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
	go build -trimpath -ldflags "-H windowsgui -s -w" -o dist/VideoToAudio.exe .
echo "完成: dist/VideoToAudio.exe"
