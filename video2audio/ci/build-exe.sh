#!/bin/bash
# 构建 Windows 单文件 VideoToAudio.exe(内嵌 ffmpeg.exe)。
set -euxo pipefail
cd "$(dirname "$0")/.."   # video2audio/

mkdir -p dist

if [ ! -f .build/ffmpeg.exe ]; then
	echo "缺少 .build/ffmpeg.exe,请先运行 ci/build-ffmpeg.sh" >&2
	exit 1
fi
cp -f .build/ffmpeg.exe ffmpeg.exe

echo "==> go test"
CGO_ENABLED=0 go test ./...

echo "==> go vet"
CGO_ENABLED=0 go vet ./...

echo "==> 编译 VideoToAudio.exe"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
	go build -tags embedffmpeg -trimpath -ldflags "-H windowsgui -s -w" \
	-o dist/VideoToAudio.exe .

sha256sum dist/VideoToAudio.exe | tee dist/VideoToAudio.exe.sha256
ls -la dist/
