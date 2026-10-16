package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Fmt 输出格式。
type Fmt string

const (
	FmtMP3  Fmt = "mp3"
	FmtM4A  Fmt = "m4a"
	FmtWAV  Fmt = "wav"
	FmtFLAC Fmt = "flac"
	FmtCopy Fmt = "copy" // 原样提取音频流,不转码
)

// FmtDef 界面上的一个输出格式选项。
type FmtDef struct {
	Key      Fmt
	Label    string
	Ext      string // FmtCopy 时为空,由源音频编码决定
	Bitrates []string
}

// FmtList 下拉框顺序即界面顺序。
var FmtList = []FmtDef{
	{FmtMP3, "MP3 · 兼容性最好", "mp3", []string{"128 kbps", "192 kbps", "256 kbps", "320 kbps"}},
	{FmtM4A, "M4A · AAC 编码,体积更小", "m4a", []string{"128 kbps", "192 kbps", "256 kbps"}},
	{FmtWAV, "WAV · 无损未压缩", "wav", nil},
	{FmtFLAC, "FLAC · 无损压缩", "flac", nil},
	{FmtCopy, "原样提取 · 不转码,无损", "", nil},
}

func fmtDef(k Fmt) FmtDef {
	for _, f := range FmtList {
		if f.Key == k {
			return f
		}
	}
	return FmtList[0]
}

// OutMode 音频文件的保存位置。
type OutMode int

const (
	OutSame   OutMode = iota // 与源文件同目录
	OutSub                   // 源文件目录下的 audio 子目录
	OutCustom                // 自定义目录
)

// Job 一次转换任务的公共配置。
type Job struct {
	Format    Fmt
	Bitrate   string
	Mode      OutMode
	CustomDir string
}

// bitrateVal 把界面上的 "192 kbps" 转成 ffmpeg 的 "192k"。
func bitrateVal(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return "192k"
	}
	if n, err := strconv.Atoi(f[0]); err == nil && n > 0 && n <= 999 {
		return fmt.Sprintf("%dk", n)
	}
	return "192k"
}

// copyExt 原样提取时,按音频编码选择输出扩展名。
func copyExt(codec string) string {
	switch codec {
	case "aac", "aac_latm", "alac":
		return "m4a"
	case "mp3", "mp2":
		return "mp3"
	case "mp1":
		return "mp2"
	case "opus":
		return "opus"
	case "vorbis":
		return "ogg"
	case "flac":
		return "flac"
	case "ac3":
		return "ac3"
	case "eac3":
		return "eac3"
	case "wmav1", "wmav2", "wmapro", "wmalossless":
		return "wma"
	}
	if strings.HasPrefix(codec, "pcm_") {
		return "wav"
	}
	return "mka"
}

// outDirFor 计算某输入文件的输出目录。
func outDirFor(input string, job Job) string {
	switch job.Mode {
	case OutSub:
		return filepath.Join(filepath.Dir(input), "audio")
	case OutCustom:
		if job.CustomDir != "" {
			return job.CustomDir
		}
	}
	return filepath.Dir(input)
}

// outPath 计算输出文件路径;若与输入相同则加 _audio 后缀。
func outPath(input string, job Job, codec string) string {
	ext := fmtDef(job.Format).Ext
	if job.Format == FmtCopy {
		ext = copyExt(codec)
	}
	name := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	p := filepath.Join(outDirFor(input, job), name+"."+ext)
	if sameFilePath(p, input) {
		p = filepath.Join(outDirFor(input, job), name+"_audio."+ext)
	}
	return p
}

func sameFilePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// BuildArgs 构造 ffmpeg 命令行参数。
func BuildArgs(input, output string, job Job, codec string) []string {
	a := []string{"-hide_banner", "-nostdin", "-y", "-i", input, "-vn", "-map", "0:a:0", "-map_metadata", "0"}
	switch job.Format {
	case FmtMP3:
		a = append(a, "-c:a", "libmp3lame", "-b:a", bitrateVal(job.Bitrate))
	case FmtM4A:
		a = append(a, "-c:a", "aac", "-b:a", bitrateVal(job.Bitrate))
	case FmtWAV:
		a = append(a, "-c:a", "pcm_s16le")
	case FmtFLAC:
		a = append(a, "-c:a", "flac")
	case FmtCopy:
		a = append(a, "-c:a", "copy")
		if codec == "aac" || codec == "aac_latm" {
			a = append(a, "-bsf:a", "aac_adtstoasc")
		}
	}
	a = append(a, "-progress", "pipe:1", "-nostats", output)
	return a
}

// Probe 文件探测结果。
type Probe struct {
	Duration   time.Duration
	AudioCodec string
}

var (
	reDuration = regexp.MustCompile(`Duration: (\d+):(\d+):(\d+(?:\.\d+)?)`)
	reAudio    = regexp.MustCompile(`Stream #\S+.*?: Audio: ([A-Za-z0-9_]+)`)
	reFFPrefix = regexp.MustCompile(`\[[^\]]*? @ 0x[0-9a-fA-F]+\]\s*`)
)

// ProbeFile 用 ffmpeg 读取时长与音频编码(顺带探测文件合法性)。
func ProbeFile(ffmpeg, input string) (Probe, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-i", input)
	out, _ := cmd.CombinedOutput() // 无输出文件,退出码非 0,忽略
	text := string(out)

	var p Probe
	if m := reDuration.FindStringSubmatch(text); m != nil {
		h, _ := strconv.Atoi(m[1])
		mi, _ := strconv.Atoi(m[2])
		s, _ := strconv.ParseFloat(m[3], 64)
		p.Duration = time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute +
			time.Duration(s*float64(time.Second))
	}
	if m := reAudio.FindStringSubmatch(text); m != nil {
		p.AudioCodec = strings.ToLower(m[1])
	}
	if p.AudioCodec == "" {
		if p.Duration == 0 {
			return p, errors.New("无法读取该文件")
		}
		return p, errors.New("文件中没有音频流")
	}
	return p, nil
}

// splitProgressLine 解析 -progress 输出的 key=value 行。
func splitProgressLine(line string) (string, string) {
	k, v, ok := strings.Cut(line, "=")
	if !ok {
		return "", ""
	}
	return strings.TrimSpace(k), strings.TrimSpace(v)
}

// parseOutTime 解析 00:00:05.000000 形式的时间。
func parseOutTime(v string) (time.Duration, bool) {
	parts := strings.SplitN(v, ":", 3)
	if len(parts) != 3 {
		return 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	s, err3 := strconv.ParseFloat(parts[2], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, false
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
		time.Duration(s*float64(time.Second)), true
}

// tailBuffer 只保留最后 max 字节,用于截取 ffmpeg 报错。
type tailBuffer struct {
	buf bytes.Buffer
	max int
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	if t.max <= 0 {
		t.max = 2048
	}
	t.buf.Write(p)
	if t.buf.Len() > t.max {
		b := t.buf.Bytes()
		t.buf.Reset()
		t.buf.Write(b[len(b)-t.max:])
	}
	return len(p), nil
}

func (t *tailBuffer) String() string { return t.buf.String() }

// summarizeErr 从 ffmpeg 输出里挑一句最有用的错误。
func summarizeErr(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := reFFPrefix.ReplaceAllString(strings.TrimSpace(lines[i]), "")
		if l == "" {
			continue
		}
		low := strings.ToLower(l)
		for _, k := range []string{"error", "invalid", "no such", "unable", "does not contain", "failed", "denied", "not found", "unsupported"} {
			if strings.Contains(low, k) {
				return l
			}
		}
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return reFFPrefix.ReplaceAllString(l, "")
		}
	}
	return "转换失败"
}

// Ev 转换过程中的事件。
type EvType int

const (
	EvFileStart EvType = iota
	EvProgress
	EvFileDone
	EvFileFail
	EvQueueDone
)

type Ev struct {
	Type    EvType
	Index   int
	Percent float64
	OutPath string
	Err     string
	OK      int
	Fail    int
}

// Runner 顺序执行一批转换任务。
type Runner struct {
	FFmpeg string
	EvCh   chan Ev
	cancel context.CancelFunc
}

func NewRunner(ffmpeg string) *Runner {
	return &Runner{FFmpeg: ffmpeg, EvCh: make(chan Ev, 256)}
}

// Start 在后台开始转换,进度通过 EvCh 发送。
func (r *Runner) Start(files []string, job Job) {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	go func() {
		ok, fail := 0, 0
		for i, f := range files {
			if ctx.Err() != nil {
				r.EvCh <- Ev{Type: EvFileFail, Index: i, Err: "已取消"}
				fail++
				continue
			}
			r.EvCh <- Ev{Type: EvFileStart, Index: i}
			out, err := r.convertOne(ctx, i, f, job)
			if err != nil {
				msg := err.Error()
				if ctx.Err() != nil {
					msg = "已取消"
				}
				r.EvCh <- Ev{Type: EvFileFail, Index: i, Err: msg}
				fail++
				continue
			}
			r.EvCh <- Ev{Type: EvFileDone, Index: i, OutPath: out}
			ok++
		}
		r.EvCh <- Ev{Type: EvQueueDone, OK: ok, Fail: fail}
	}()
}

func (r *Runner) Cancel() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Runner) convertOne(ctx context.Context, idx int, input string, job Job) (string, error) {
	pr, err := ProbeFile(r.FFmpeg, input)
	if err != nil {
		return "", err
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	out := outPath(input, job, pr.AudioCodec)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", fmt.Errorf("无法创建输出目录: %v", err)
	}
	args := BuildArgs(input, out, job, pr.AudioCodec)
	cmd := exec.CommandContext(ctx, r.FFmpeg, args...)
	stderr := &tailBuffer{max: 2048}
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("无法启动 ffmpeg: %v", err)
	}
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		k, v := splitProgressLine(sc.Text())
		if k != "out_time" {
			continue
		}
		d, ok := parseOutTime(v)
		if !ok || pr.Duration <= 0 {
			continue
		}
		p := float64(d) / float64(pr.Duration)
		if p > 1 {
			p = 1
		}
		r.EvCh <- Ev{Type: EvProgress, Index: idx, Percent: p}
	}
	_ = sc.Err()
	err = cmd.Wait()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", errors.New(summarizeErr(stderr.String()))
	}
	return out, nil
}
