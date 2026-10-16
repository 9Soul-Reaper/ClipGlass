package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBitrateVal(t *testing.T) {
	cases := map[string]string{
		"128 kbps": "128k",
		"192 kbps": "192k",
		"320 kbps": "320k",
		"":         "192k",
		"abc":      "192k",
	}
	for in, want := range cases {
		if got := bitrateVal(in); got != want {
			t.Errorf("bitrateVal(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCopyExt(t *testing.T) {
	cases := map[string]string{
		"aac":     "m4a",
		"alac":    "m4a",
		"mp3":     "mp3",
		"opus":    "opus",
		"vorbis":  "ogg",
		"flac":    "flac",
		"ac3":     "ac3",
		"wmav2":   "wma",
		"pcm_s16le": "wav",
		"dca":     "mka",
		"cook":    "mka",
	}
	for in, want := range cases {
		if got := copyExt(in); got != want {
			t.Errorf("copyExt(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseOutTime(t *testing.T) {
	d, ok := parseOutTime("00:00:05.000000")
	if !ok || d != 5*time.Second {
		t.Errorf("parseOutTime = %v %v", d, ok)
	}
	d, ok = parseOutTime("01:02:03.500000")
	if !ok || d != time.Hour+2*time.Minute+3500*time.Millisecond {
		t.Errorf("parseOutTime = %v %v", d, ok)
	}
	if _, ok := parseOutTime("bad"); ok {
		t.Error("parseOutTime should fail on garbage")
	}
}

func TestSplitProgressLine(t *testing.T) {
	k, v := splitProgressLine("out_time=00:00:05.000000")
	if k != "out_time" || v != "00:00:05.000000" {
		t.Errorf("got %q %q", k, v)
	}
	if k, v := splitProgressLine("nop"); k != "" || v != "" {
		t.Errorf("got %q %q", k, v)
	}
}

func TestBuildArgs(t *testing.T) {
	args := BuildArgs(`C:\a\in.mp4`, `C:\a\out.mp3`, Job{Format: FmtMP3, Bitrate: "192 kbps"}, "aac")
	s := strings.Join(args, " ")
	for _, want := range []string{"-i C:\\a\\in.mp4 -vn", "-c:a libmp3lame", "-b:a 192k", "-progress pipe:1", "C:\\a\\out.mp3"} {
		if !strings.Contains(s, want) {
			t.Errorf("mp3 args missing %q in %q", want, s)
		}
	}

	args = BuildArgs("in.mkv", "out.m4a", Job{Format: FmtCopy}, "aac")
	s = strings.Join(args, " ")
	if !strings.Contains(s, "-c:a copy") || !strings.Contains(s, "aac_adtstoasc") {
		t.Errorf("copy aac args wrong: %q", s)
	}

	args = BuildArgs("in.mkv", "out.mka", Job{Format: FmtCopy}, "dca")
	s = strings.Join(args, " ")
	if !strings.Contains(s, "-c:a copy") || strings.Contains(s, "aac_adtstoasc") {
		t.Errorf("copy dca args wrong: %q", s)
	}

	args = BuildArgs("in.mkv", "out.wav", Job{Format: FmtWAV}, "aac")
	if !strings.Contains(strings.Join(args, " "), "-c:a pcm_s16le") {
		t.Error("wav args wrong")
	}
	args = BuildArgs("in.mkv", "out.flac", Job{Format: FmtFLAC}, "aac")
	if !strings.Contains(strings.Join(args, " "), "-c:a flac") {
		t.Error("flac args wrong")
	}
}

func TestOutPath(t *testing.T) {
	// 用 slash 路径书写,使测试在 Windows/Linux 上都成立
	p := outPath("C:/v/movie.mp4", Job{Format: FmtMP3, Mode: OutSame}, "aac")
	if want := filepath.Join("C:", "v", "movie.mp3"); p != want {
		t.Errorf("got %q want %q", p, want)
	}
	// 输出与输入相同时加 _audio
	p = outPath("C:/v/song.mp3", Job{Format: FmtMP3, Mode: OutSame}, "mp3")
	if want := filepath.Join("C:", "v", "song_audio.mp3"); p != want {
		t.Errorf("got %q want %q", p, want)
	}
	// 子目录模式
	p = outPath("C:/v/movie.mkv", Job{Format: FmtCopy, Mode: OutSub}, "opus")
	if want := filepath.Join("C:", "v", "audio", "movie.opus"); p != want {
		t.Errorf("got %q want %q", p, want)
	}
	// 自定义目录
	p = outPath("C:/v/movie.mkv", Job{Format: FmtFLAC, Mode: OutCustom, CustomDir: "D:/out"}, "aac")
	if want := filepath.Join("D:", "out", "movie.flac"); p != want {
		t.Errorf("got %q want %q", p, want)
	}
	// copy 时按编码选扩展名
	p = outPath("C:/v/movie.mkv", Job{Format: FmtCopy, Mode: OutSame}, "vorbis")
	if want := filepath.Join("C:", "v", "movie.ogg"); p != want {
		t.Errorf("got %q want %q", p, want)
	}
}

func TestSummarizeErr(t *testing.T) {
	s := "[aac @ 0x55d0] Some note\n[in#0 @ 0x55d1] Error opening input: Invalid data found when processing input\nError opening input files: Invalid data found when processing input\n"
	got := summarizeErr(s)
	if !strings.Contains(got, "Invalid data") {
		t.Errorf("got %q", got)
	}
	// 前缀应被去掉
	if strings.Contains(got, " @ 0x") {
		t.Errorf("prefix not stripped: %q", got)
	}
}
