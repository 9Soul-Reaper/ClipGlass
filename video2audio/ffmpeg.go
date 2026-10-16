package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// locateFFmpeg 查找 ffmpeg:
//  1. 与 exe 同目录的 ffmpeg.exe(用户可自行替换)
//  2. 内嵌的 ffmpeg.exe(释放到缓存目录)
//  3. 系统 PATH
func locateFFmpeg() (string, error) {
	if exe, err := os.Executable(); err == nil {
		for _, name := range []string{"ffmpeg.exe", "ffmpeg"} {
			p := filepath.Join(filepath.Dir(exe), name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	if p, ok := extractEmbeddedFFmpeg(); ok {
		return p, nil
	}
	for _, name := range []string{"ffmpeg.exe", "ffmpeg"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("未找到 ffmpeg:请把 ffmpeg.exe 放到本程序同目录,或安装 ffmpeg 到系统 PATH")
}

// extractEmbeddedFFmpeg 把内嵌的 ffmpeg.exe 释放到缓存目录并返回路径。
// 未内嵌时(embeddedFFmpeg 为空)返回 false。
func extractEmbeddedFFmpeg() (string, bool) {
	if len(embeddedFFmpeg) == 0 {
		return "", false
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	sum := sha256.Sum256(embeddedFFmpeg)
	dir := filepath.Join(base, "VideoToAudio")
	exe := filepath.Join(dir, "ffmpeg-"+hex.EncodeToString(sum[:4])+".exe")
	// 已释放过且大小一致就直接复用
	if st, err := os.Stat(exe); err == nil && st.Size() == int64(len(embeddedFFmpeg)) {
		return exe, true
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false
	}
	tmp := exe + ".tmp"
	if err := os.WriteFile(tmp, embeddedFFmpeg, 0o755); err != nil {
		return "", false
	}
	_ = os.Remove(exe)
	if err := os.Rename(tmp, exe); err != nil {
		_ = os.Remove(tmp)
		// 并发释放时可能已存在,复用
		if st, err := os.Stat(exe); err == nil && st.Size() == int64(len(embeddedFFmpeg)) {
			return exe, true
		}
		return "", false
	}
	return exe, true
}
