//go:build !windows

package main

import "fmt"

// runGUI 非 Windows 平台只给提示(GUI 版仅支持 Windows)。
func runGUI(files []string) {
	fmt.Println("VideoToAudio - 视频转音频")
	fmt.Println("请在 Windows 上运行 VideoToAudio.exe。")
}
