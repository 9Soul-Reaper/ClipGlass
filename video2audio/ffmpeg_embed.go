//go:build embedffmpeg

package main

import _ "embed"

// 构建时用 -tags embedffmpeg 内嵌 ffmpeg.exe(由 ci/build-ffmpeg.sh 生成)。
//
//go:embed ffmpeg.exe
var embeddedFFmpeg []byte
