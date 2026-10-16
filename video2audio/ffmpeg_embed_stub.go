//go:build !embedffmpeg

package main

// 不带 embedffmpeg 构建标签时不内嵌 ffmpeg(用于开发编译与单元测试)。
var embeddedFFmpeg []byte
