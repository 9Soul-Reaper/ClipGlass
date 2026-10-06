package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ---------- 类型 ----------

type Kind int

const (
	KText Kind = iota
	KLink
	KCode
	KContact
	KPath
	KNumber
	KImage
	KColor
	KCommand
	KData
	KTable
	KFile
	KDate
	KNetwork
	KID
	KAddress
	KSecret
	kindCount
)

var kindKey = [...]string{"kt.text", "kt.link", "kt.code", "kt.contact", "kt.path", "kt.number", "kt.image", "",
	"kt.command", "kt.data", "kt.table", "kt.file", "kt.date", "kt.network", "kt.id", "kt.address", "kt.secret"}
var kindChipKey = [...]string{"chip.text", "chip.link", "chip.code", "chip.contact", "chip.path", "chip.number", "chip.image", "chip.color",
	"chip.command", "chip.data", "chip.table", "chip.file", "chip.date", "chip.network", "chip.id", "chip.address", "chip.secret"}
var kindColor = [...]Color{{140, 152, 182}, {76, 141, 255}, {167, 112, 255}, {52, 199, 140}, {255, 166, 60}, {255, 99, 132}, {70, 190, 230}, {150, 150, 150},
	{120, 200, 120}, {240, 200, 80}, {90, 190, 170}, {255, 140, 90}, {150, 130, 240}, {70, 160, 255}, {190, 150, 110}, {235, 120, 180}, {240, 80, 80}}

// 筛选码:0 全部,1 置顶,100+类型,1000+自定义分区
const (
	FilterAll       = 0
	FilterPinned    = 1
	FilterKindBase  = 100
	FilterGroupBase = 1000
)

type Group struct {
	ID    int    `json:"id"`
	Name  string `json:"n"`
	Color int    `json:"c"`
}

type Item struct {
	ID     int64  `json:"id"`
	Text   string `json:"t"`
	Kind   Kind   `json:"k"`
	Time   int64  `json:"ts"`
	Pinned bool   `json:"p,omitempty"`
	Source string `json:"s,omitempty"`
	Exe    string `json:"x,omitempty"`
	Uses   int    `json:"u,omitempty"`
	Img    string `json:"i,omitempty"` // 图片哈希(文件名)
	W      int    `json:"w,omitempty"`
	H      int    `json:"h,omitempty"`
	Group  int    `json:"g,omitempty"`

	Mem    bool   `json:"-"` // 仅内存保存(敏感内容),不落盘、限时清除
	Reason string `json:"-"` // 敏感类型的 i18n key

	lower     string
	lines     int
	runes     int
	thumb     *Canvas
	thumbFail bool
	color     Color
}

func (it *Item) prepare() {
	if it.Kind == KSecret {
		it.lower = "sensitive secret 敏感 " + strings.ToLower(it.Source)
		it.lines, it.runes = 1, len([]rune(it.Text))
		return
	}
	if it.Kind == KImage {
		it.lower = strings.ToLower("image 图片 圖片 画像 이미지 " + fmt.Sprintf("%dx%d ", it.W, it.H) + it.Source)
		return
	}
	it.lower = strings.ToLower(it.Text + " " + it.Source)
	it.lines = strings.Count(it.Text, "\n") + 1
	it.runes = len([]rune(it.Text))
	if it.Kind == KColor {
		it.color, _ = parseColor(it.Text)
	}
}

// ---------- 内容分类 ----------

var (
	reURL      = regexp.MustCompile(`(?i)^(https?://|ftp://|www\.)\S+$`)
	reURLInner = regexp.MustCompile(`(?i)\bhttps?://\S+`)
	reEmail    = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
	rePhone    = regexp.MustCompile(`^\+?\d[\d\-\s()]{5,18}\d$`)
	reWinPath  = regexp.MustCompile(`^([A-Za-z]:\\|\\\\)[^\r\n]{2,}$`)
	reUnixPath = regexp.MustCompile(`^(~|\.{1,2})?/[\w.\-]+(/[\w.\- ]*)*$`)
	reNumber   = regexp.MustCompile(`^[¥￥$€]?\s?\d[\d,，.\s:\-/]*%?$`)
	reHexColor = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	reRGB      = regexp.MustCompile(`(?i)^rgba?\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(,\s*[\d.]+\s*)?\)$`)
	reCodeLead = regexp.MustCompile(`(?m)^\s*(func|function|def|class|import|from|package|using|public|private|protected|const|let|var|return|if|else|for|while|switch|case|try|catch|#include|<\?php|<html|<div|SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP)\b`)
	reHSL      = regexp.MustCompile(`(?i)^hsla?\(\s*(\d{1,3})(deg)?\s*[, ]\s*(\d{1,3})%\s*[, ]\s*(\d{1,3})%\s*([,/]\s*[\d.]+%?\s*)?\)$`)
	reLoose    = regexp.MustCompile(`(?i)^(localhost|([a-z0-9][a-z0-9\-]*\.)+[a-z]{2,})(:\d{1,5})?(/\S*)$`)
	reLocal    = regexp.MustCompile(`(?i)^localhost:\d{1,5}$`)
	reFileName = regexp.MustCompile(`(?i)^[^\s\\/:*?"<>|]{2,}\.(txt|md|go|py|js|ts|tsx|jsx|json|pdf|docx?|xlsx?|pptx?|zip|rar|7z|tar|gz|png|jpe?g|gif|webp|svg|bmp|mp[34]|mkv|avi|mov|exe|msi|csv|log|html?|css|java|kt|c|cpp|h|cs|rs|sh|bat|ps1|ya?ml|toml|ini|sql|apk|iso|dmg|dll|so)$`)
	reNum2     = regexp.MustCompile(`^[-+]?\d+(\.\d+)?([eE][-+]?\d+)?%$|^[-+]?\d+(\.\d+)?[eE][-+]?\d+$`)
	reVersion  = regexp.MustCompile(`^[vV]?\d+(\.\d+){1,3}([-+][0-9A-Za-z.\-]+)?$`)
	reEmails   = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}([,;，；\s]+[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,})+$`)
	reCodeLine = regexp.MustCompile(`(?i)^(import\s+[\w.{}*, ]+|from\s+\S+\s+import\s+.+|(const|let|var|def|func|function|class|package|using|return|print|echo)\s+\S.*|select\s+.+|insert\s+into\s.+|update\s+\S+\s+set\s.+|delete\s+from\s.+|[A-Za-z_][\w.]*\(.*\);?|[\w.\[\]]+\s*(=|\+=|-=|:=)\s*.+;)$`)
	reCommand  = regexp.MustCompile(`^(git|npm|npx|yarn|pnpm|pip|pip3|python|python3|node|docker|kubectl|curl|wget|sudo|cd|ls|mkdir|rm|cp|mv|cat|go|cargo|dotnet|make|ssh|scp|apt|brew)\s+\S`)
)

func parseColor(s string) (Color, bool) {
	s = strings.TrimSpace(s)
	if reHexColor.MatchString(s) {
		h := s[1:]
		if len(h) == 3 {
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		}
		v, _ := strconv.ParseUint(h[:6], 16, 32)
		return Color{uint8(v >> 16), uint8(v >> 8), uint8(v)}, true
	}
	if m := reRGB.FindStringSubmatch(s); m != nil {
		r, _ := strconv.Atoi(m[1])
		g, _ := strconv.Atoi(m[2])
		b, _ := strconv.Atoi(m[3])
		if r < 256 && g < 256 && b < 256 {
			return Color{uint8(r), uint8(g), uint8(b)}, true
		}
	}
	if mm := reHSL.FindStringSubmatch(s); mm != nil {
		h, _ := strconv.Atoi(mm[1])
		sa, _ := strconv.Atoi(mm[3])
		l, _ := strconv.Atoi(mm[4])
		if h <= 360 && sa <= 100 && l <= 100 {
			return hslToColor(float64(h), float64(sa)/100, float64(l)/100), true
		}
	}
	return Color{}, false
}

func hslToColor(h, s, l float64) Color {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	mm := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return Color{clampByte((r + mm) * 255), clampByte((g + mm) * 255), clampByte((b + mm) * 255)}
}

func digitCount(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

func cjkRatio(s string) float64 {
	total, cjk := 0, 0
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		total++
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			cjk++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(cjk) / float64(total)
}

func classify(text string) Kind {
	t := strings.TrimSpace(text)
	if t == "" {
		return KText
	}
	oneLine := !strings.Contains(t, "\n")
	if oneLine {
		if _, ok := parseColor(t); ok {
			return KColor
		}
		switch {
		case reURL.MatchString(t):
			return KLink
		case reEmail.MatchString(t), reEmails.MatchString(t):
			return KContact
		case reLocal.MatchString(t), reLoose.MatchString(t) && looseHostOK(t):
			return KLink
		case reFileName.MatchString(t):
			return KPath
		case reWinPath.MatchString(t) || reUnixPath.MatchString(t):
			return KPath
		}
		if k, ok := classifyMore(t); ok && (k == KDate || k == KNetwork || k == KID) {
			return k
		}
		dc := digitCount(t)
		if reNum2.MatchString(t) || (reVersion.MatchString(t) && dc >= 2) {
			return KNumber
		}
		if rePhone.MatchString(t) && dc >= 7 && dc <= 13 {
			return KContact
		}
		if reNumber.MatchString(t) && dc >= 4 {
			return KNumber
		}
		if len(t) < 600 && reURLInner.MatchString(t) {
			if rest := strings.TrimSpace(reURLInner.ReplaceAllString(t, "")); len([]rune(rest)) <= 6 {
				return KLink
			}
		}
		if k, ok := classifyMore(t); ok {
			return k
		}
		if cjkRatio(t) < 0.3 && len(t) < 300 && reCodeLine.MatchString(t) && !reCommand.MatchString(t) {
			return KCode
		}
		if reCommand.MatchString(t) || strings.HasPrefix(t, "$ ") || strings.HasPrefix(t, "PS C:\\") {
			return KCommand
		}
	}
	if k, ok := classifyBlock(t); ok {
		return k
	}
	if cjkRatio(t) < 0.3 && looksLikeCode(t) {
		return KCode
	}
	return KText
}

// looseHostOK:无协议的 host/path 只有在后缀是常见域名后缀(或 localhost)时才算链接。
func looseHostOK(t string) bool {
	host := strings.ToLower(strings.SplitN(strings.SplitN(t, "/", 2)[0], ":", 2)[0])
	if host == "localhost" {
		return true
	}
	return knownTLD[host[strings.LastIndex(host, ".")+1:]]
}

func looksLikeCode(t string) bool {
	score := 0
	if reCodeLead.MatchString(t) {
		score += 2
	}
	for _, s := range []string{"{", "}", ";", "=>", "->", "</", "/>", "();", "):", "==", "!=", "&&", "||", "::", "<=", ">="} {
		if strings.Contains(t, s) {
			score++
		}
	}
	lines := strings.Split(t, "\n")
	if len(lines) > 1 {
		ind := 0
		for _, l := range lines {
			if strings.HasPrefix(l, "  ") || strings.HasPrefix(l, "\t") {
				ind++
			}
		}
		if ind*3 >= len(lines) {
			score += 2
		}
	}
	up := strings.ToUpper(t)
	if strings.Contains(up, "SELECT ") && strings.Contains(up, " FROM ") {
		score += 2
	}
	return score >= 3
}

func luhn(d string) bool {
	sum, alt := 0, false
	for i := len(d) - 1; i >= 0; i-- {
		n := int(d[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

// 常见密码管理器的进程名片段
var pwdManagers = []string{"keepass", "1password", "bitwarden", "lastpass", "dashlane", "enpass", "roboform", "nordpass"}

// ---------- 设置 ----------

const (
	ModAlt   = 1
	ModCtrl  = 2
	ModShift = 4
	ModWin   = 8
)

type HotkeySpec struct {
	Mods uint32 `json:"m"`
	VK   uint32 `json:"k"`
}

func (h HotkeySpec) IsSet() bool { return h.VK != 0 }

func (h HotkeySpec) String() string {
	if !h.IsSet() {
		return T("hk.none")
	}
	var p []string
	if h.Mods&ModCtrl != 0 {
		p = append(p, "Ctrl")
	}
	if h.Mods&ModAlt != 0 {
		p = append(p, "Alt")
	}
	if h.Mods&ModShift != 0 {
		p = append(p, "Shift")
	}
	if h.Mods&ModWin != 0 {
		p = append(p, "Win")
	}
	return strings.Join(append(p, vkName(h.VK)), "+")
}

func vkName(vk uint32) string {
	switch {
	case vk >= 'A' && vk <= 'Z', vk >= '0' && vk <= '9':
		return string(rune(vk))
	case vk >= 0x70 && vk <= 0x87:
		return fmt.Sprintf("F%d", vk-0x6F)
	case vk >= 0x60 && vk <= 0x69:
		return fmt.Sprintf("Num%d", vk-0x60)
	}
	names := map[uint32]string{0x20: "Space", 0x09: "Tab", 0x0D: "Enter", 0x2D: "Insert", 0x2E: "Delete", 0x24: "Home", 0x23: "End",
		0x21: "PageUp", 0x22: "PageDown", 0x25: "←", 0x26: "↑", 0x27: "→", 0x28: "↓", 0xC0: "`", 0xBA: ";", 0xBB: "=", 0xBC: ",",
		0xBD: "-", 0xBE: ".", 0xBF: "/", 0xDB: "[", 0xDC: "\\", 0xDD: "]", 0xDE: "'", 0x13: "Pause", 0x2C: "PrtSc"}
	if n, ok := names[vk]; ok {
		return n
	}
	return fmt.Sprintf("0x%X", vk)
}

type Settings struct {
	Lang          string     `json:"lang"` // auto / zh-CN / en / zh-TW / ja / ko
	Glass         bool       `json:"glass"`
	Blur          int        `json:"blur"`   // 0 低 1 中 2 高
	Theme         int        `json:"theme"`  // 0 自动 1 浅色 2 深色
	Accent        int        `json:"accent"` // 色板序号
	UIScale       int        `json:"uiScale"`
	Compact       bool       `json:"compact"`
	Anim          bool       `json:"anim"`
	AutoPaste     bool       `json:"autoPaste"`
	AutoStart     bool       `json:"autoStart"`
	Sensitive     int        `json:"sensitive"` // 0 跳过 1 仅内存保留(遮罩) 2 不过滤
	IgnorePwd     bool       `json:"ignorePwd"`
	RecordImages  bool       `json:"recordImages"`
	Retention     int        `json:"retention"` // 天,0 永久
	MaxItems      int        `json:"maxItems"`
	Hotkey        HotkeySpec `json:"hotkey"`
	PauseKey      HotkeySpec `json:"pauseKey"`
	PosMode       int        `json:"posMode"` // 0 鼠标附近 1 屏幕中央 2 记住位置
	PosX          int        `json:"posX"`
	PosY          int        `json:"posY"`
	PosSet        bool       `json:"posSet"`
	Paused        bool       `json:"paused"`
	IgnoreSources []string   `json:"ignoreSources"`
	ClickMode     int        `json:"clickMode"` // 0 点击=复制(窗口保持) 1 点击=粘贴并关闭
	Encrypt       bool       `json:"encrypt"`
	AutoPasteNote int        `json:"autoPasteNote"` // 0 未粘贴过 1 待提示 2 已提示
	Opacity       int        `json:"opacity"`       // 玻璃浓度 0 通透 1 均衡 2 清晰
	KeepOpen      bool       `json:"keepOpen"`
	Ball          bool       `json:"ball"`
	BallX         int        `json:"ballX"`
	BallY         int        `json:"ballY"`
	BallSet       bool       `json:"ballSet"`
}

func defaultSettings() *Settings {
	return &Settings{
		Lang: "auto", Glass: true, Blur: 1, Theme: 0, Accent: 0, UIScale: 1, Anim: true, AutoPaste: true,
		Sensitive: 0, Opacity: 1, IgnorePwd: true, RecordImages: true, Retention: 30, MaxItems: 500,
		Hotkey: HotkeySpec{Mods: ModCtrl | ModShift, VK: 'V'},
	}
}

var (
	retentionVals = []int{7, 30, 90, 0}
	maxItemVals   = []int{200, 500, 1000}
	scaleVals     = []float64{0.9, 1.0, 1.15, 1.3}
	scaleLbls     = []string{"90%", "100%", "115%", "130%"}
)

func indexOf(vals []int, v int) int {
	for i, x := range vals {
		if x == v {
			return i
		}
	}
	return 0
}

// ---------- 存储 ----------

type Store struct {
	Items     []*Item // 最新在前
	Groups    []Group
	Dir       string
	Encrypt   bool // 静态加密(DPAPI)
	Recovered bool // 启动时从备份恢复过
	purgeBak  bool
	nextID    int64
	dirty     bool
}

const secretTTL = 10 * 60 // 敏感内容在内存中保留的秒数

func dataDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	d := filepath.Join(base, "ClipGlass")
	_ = os.MkdirAll(d, 0o755)
	return d
}

func LoadSettings(dir string) *Settings {
	s := defaultSettings()
	if b, err := os.ReadFile(filepath.Join(dir, "settings.json")); err == nil {
		_ = json.Unmarshal(b, s)
	}
	s.Blur = clampi(s.Blur, 0, 2)
	s.Theme = clampi(s.Theme, 0, 2)
	s.Accent = clampi(s.Accent, 0, len(accents)-1)
	s.UIScale = clampi(s.UIScale, 0, len(scaleVals)-1)
	s.PosMode = clampi(s.PosMode, 0, 2)
	s.Sensitive = clampi(s.Sensitive, 0, 2)
	s.Opacity = clampi(s.Opacity, 0, 2)
	s.ClickMode = clampi(s.ClickMode, 0, 1)
	if s.MaxItems <= 0 {
		s.MaxItems = 500
	}
	return s
}

func (s *Settings) Save(dir string) {
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		atomicWrite(filepath.Join(dir, "settings.json"), b)
	}
}

func atomicWrite(path string, b []byte) {
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, path)
	}
}

func LoadStore(dir string) *Store {
	st := &Store{Dir: dir, nextID: time.Now().UnixNano()}
	st.loadHistory()
	for _, it := range st.Items {
		it.prepare()
	}
	if b, err := os.ReadFile(filepath.Join(dir, "groups.json")); err == nil {
		_ = json.Unmarshal(b, &st.Groups)
	}
	return st
}

func (s *Store) saveGroups() {
	if s.Dir == "" {
		return
	}
	if b, err := json.Marshal(s.Groups); err == nil {
		writeWithBackup(filepath.Join(s.Dir, "groups.json"), b)
	}
}

func (s *Store) GroupByID(id int) *Group {
	for i := range s.Groups {
		if s.Groups[i].ID == id {
			return &s.Groups[i]
		}
	}
	return nil
}

func (s *Store) AddGroup(name string) *Group {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	id := 1
	for _, g := range s.Groups {
		if g.ID >= id {
			id = g.ID + 1
		}
	}
	s.Groups = append(s.Groups, Group{ID: id, Name: name, Color: (len(s.Groups) + 1) % len(accents)})
	s.saveGroups()
	return &s.Groups[len(s.Groups)-1]
}

func (s *Store) RenameGroup(id int, name string) {
	if g := s.GroupByID(id); g != nil && strings.TrimSpace(name) != "" {
		g.Name = strings.TrimSpace(name)
		s.saveGroups()
	}
}

func (s *Store) RecolorGroup(id int) {
	if g := s.GroupByID(id); g != nil {
		g.Color = (g.Color + 1) % len(accents)
		s.saveGroups()
	}
}

func (s *Store) DeleteGroup(id int) {
	for i, g := range s.Groups {
		if g.ID == id {
			s.Groups = append(s.Groups[:i], s.Groups[i+1:]...)
			break
		}
	}
	for _, it := range s.Items {
		if it.Group == id {
			it.Group = 0
			s.dirty = true
		}
	}
	s.saveGroups()
}

func (s *Store) SetItemGroup(itemID int64, gid int) {
	for _, it := range s.Items {
		if it.ID == itemID {
			it.Group = gid
			s.dirty = true
			return
		}
	}
}

func (s *Store) CountKind(k Kind) int {
	n := 0
	for _, it := range s.Items {
		if it.Kind == k {
			n++
		}
	}
	return n
}

func (s *Store) Save() {
	if !s.dirty || s.Dir == "" {
		return
	}
	b, err := s.marshalHistory()
	if err != nil {
		return
	}
	path := filepath.Join(s.Dir, "history.json")
	if writeWithBackup(path, s.seal(b)) {
		s.dirty = false
		if s.purgeBak { // 删除 / 识别为敏感后,不再保留含旧明文的备份
			_ = os.Remove(path + ".bak")
			s.purgeBak = false
		}
	}
}

// Bump 把条目移到最前。
func (s *Store) Bump(it *Item) {
	for i, x := range s.Items {
		if x == it {
			copy(s.Items[1:i+1], s.Items[:i])
			s.Items[0] = it
			it.Time = time.Now().Unix()
			it.Uses++
			s.dirty = true
			return
		}
	}
}

func (s *Store) Add(text, source string) *Item { return s.AddFrom(text, source, "", "") }

// AddFrom 新增文本条目。reason 非空表示敏感内容(仅内存保留)。
func (s *Store) AddFrom(text, source, exe, reason string) *Item {
	for i, it := range s.Items {
		if it.Kind != KImage && it.Text == text {
			// 去重命中时也要按本次的敏感判定更新:转为敏感则改为仅内存并清掉磁盘明文
			if reason != "" && !it.Mem {
				it.Kind, it.Mem, it.Reason = KSecret, true, reason
				s.purgeBak = true
			} else if reason == "" && it.Mem {
				it.Mem, it.Reason = false, ""
				it.Kind = classify(text)
			}
			it.Time = time.Now().Unix()
			it.Uses++
			if source != "" {
				it.Source = source
				it.Exe = exe
			}
			it.prepare()
			copy(s.Items[1:i+1], s.Items[:i])
			s.Items[0] = it
			s.dirty = true
			return it
		}
	}
	s.nextID++
	it := &Item{ID: s.nextID, Text: text, Kind: classify(text), Time: time.Now().Unix(), Source: source, Exe: exe}
	if reason != "" {
		it.Kind, it.Mem, it.Reason = KSecret, true, reason
	}
	it.prepare()
	s.Items = append([]*Item{it}, s.Items...)
	s.dirty = true
	return it
}

func (s *Store) imgPath(hash string) string   { return filepath.Join(s.Dir, "images", hash+".png") }
func (s *Store) thumbPath(hash string) string { return filepath.Join(s.Dir, "images", hash+"_t.png") }

// AddImage 保存图片(PNG)与缩略图,同内容去重并置顶。
func (s *Store) AddImage(img *image.NRGBA, source, exe string) *Item {
	sum := sha1.Sum(img.Pix)
	hash := hex.EncodeToString(sum[:8])
	for i, it := range s.Items {
		if it.Kind == KImage && it.Img == hash {
			it.Time = time.Now().Unix()
			copy(s.Items[1:i+1], s.Items[:i])
			s.Items[0] = it
			s.dirty = true
			return it
		}
	}
	_ = os.MkdirAll(filepath.Join(s.Dir, "images"), 0o755)
	data := encodePNG(img)
	if data == nil || s.writeBlob(s.imgPath(hash), data) != nil {
		return nil
	}
	_ = s.writeBlob(s.thumbPath(hash), encodePNG(coverThumb(img, 96)))
	s.nextID++
	it := &Item{ID: s.nextID, Kind: KImage, Time: time.Now().Unix(), Source: source, Exe: exe, Img: hash, W: img.Bounds().Dx(), H: img.Bounds().Dy()}
	it.prepare()
	s.Items = append([]*Item{it}, s.Items...)
	s.dirty = true
	return it
}

// Thumb 懒加载缩略图;失败返回 nil。
func (s *Store) Thumb(it *Item) *Canvas {
	if it.thumb != nil || it.thumbFail || it.Kind != KImage {
		return it.thumb
	}
	b, err := s.readBlob(s.thumbPath(it.Img))
	if err == nil {
		if im, err := png.Decode(bytes.NewReader(b)); err == nil {
			n := image.NewNRGBA(im.Bounds())
			for y := 0; y < im.Bounds().Dy(); y++ {
				for x := 0; x < im.Bounds().Dx(); x++ {
					r, g, bl, a := im.At(x, y).RGBA()
					i := y*n.Stride + x*4
					n.Pix[i], n.Pix[i+1], n.Pix[i+2], n.Pix[i+3] = byte(r>>8), byte(g>>8), byte(bl>>8), byte(a>>8)
				}
			}
			it.thumb = canvasFromNRGBA(n)
			return it.thumb
		}
	}
	it.thumbFail = true
	return nil
}

// LoadImage 读取原图。
func (s *Store) LoadImage(it *Item) *image.NRGBA {
	b, err := s.readBlob(s.imgPath(it.Img))
	if err != nil {
		return nil
	}
	im, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	n := image.NewNRGBA(im.Bounds())
	for y := 0; y < im.Bounds().Dy(); y++ {
		for x := 0; x < im.Bounds().Dx(); x++ {
			r, g, bl, _ := im.At(im.Bounds().Min.X+x, im.Bounds().Min.Y+y).RGBA()
			i := y*n.Stride + x*4
			n.Pix[i], n.Pix[i+1], n.Pix[i+2], n.Pix[i+3] = byte(r>>8), byte(g>>8), byte(bl>>8), 255
		}
	}
	return n
}

func (s *Store) removeFiles(it *Item) {
	if it.Kind == KImage && it.Img != "" {
		_ = os.Remove(s.imgPath(it.Img))
		_ = os.Remove(s.thumbPath(it.Img))
	}
}

func (s *Store) Delete(id int64) {
	for i, it := range s.Items {
		if it.ID == id {
			s.removeFiles(it)
			s.Items = append(s.Items[:i], s.Items[i+1:]...)
			s.dirty, s.purgeBak = true, true
			return
		}
	}
}

func (s *Store) TogglePin(id int64) {
	for _, it := range s.Items {
		if it.ID == id {
			it.Pinned = !it.Pinned
			s.dirty = true
			return
		}
	}
}

func (s *Store) ClearUnpinned() {
	keep := s.Items[:0:0]
	for _, it := range s.Items {
		if it.Pinned {
			keep = append(keep, it)
		} else {
			s.removeFiles(it)
		}
	}
	s.Items = keep
	s.dirty, s.purgeBak = true, true
}

func (s *Store) Prune(set *Settings) {
	now := time.Now().Unix()
	keep := make([]*Item, 0, len(s.Items))
	count := 0
	for _, it := range s.Items {
		if it.Mem && now-it.Time > secretTTL {
			continue
		}
		if !it.Pinned {
			if set.Retention > 0 && now-it.Time > int64(set.Retention)*86400 {
				s.removeFiles(it)
				continue
			}
			count++
			if count > set.MaxItems {
				s.removeFiles(it)
				continue
			}
		}
		keep = append(keep, it)
	}
	if len(keep) != len(s.Items) {
		s.Items = keep
		s.dirty, s.purgeBak = true, true
	}
}

// View 按筛选码与关键词返回条目(置顶在前)。
func (s *Store) View(filter int, query string) []*Item {
	toks := strings.Fields(strings.ToLower(query))
	var pinned, rest []*Item
	for _, it := range s.Items {
		switch {
		case filter == FilterPinned && !it.Pinned:
			continue
		case filter >= FilterGroupBase && it.Group != filter-FilterGroupBase:
			continue
		case filter >= FilterKindBase && filter < FilterGroupBase && int(it.Kind) != filter-FilterKindBase:
			continue
		}
		gname := ""
		if it.Group != 0 {
			if g := s.GroupByID(it.Group); g != nil {
				gname = strings.ToLower(g.Name)
			}
		}
		ok := true
		for _, t := range toks {
			if !strings.Contains(it.lower, t) && !strings.Contains(gname, t) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if it.Pinned {
			pinned = append(pinned, it)
		} else {
			rest = append(rest, it)
		}
	}
	return append(pinned, rest...)
}

// ExportText 把文本历史导出为可读文本。
func (s *Store) ExportText() string {
	var b strings.Builder
	for _, it := range s.Items {
		if it.Kind == KImage || it.Mem {
			continue
		}
		fmt.Fprintf(&b, "[%s] %s\r\n%s\r\n\r\n", time.Unix(it.Time, 0).Format("2006-01-02 15:04:05"), it.Source, strings.ReplaceAll(it.Text, "\n", "\r\n"))
	}
	return b.String()
}

// ---------- 展示辅助 ----------

func previewText(s string) string {
	if len(s) > 400 {
		s = s[:400]
	}
	s = strings.NewReplacer("\r", "", "\n", " ", "\t", " ").Replace(s)
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

func relTime(ts int64, now time.Time) string {
	t := time.Unix(ts, 0)
	d := now.Sub(t)
	switch {
	case d < 10*time.Second:
		return T("time.now")
	case d < time.Minute:
		return Tf("time.s", int(d.Seconds()))
	case d < time.Hour:
		return Tf("time.m", int(d.Minutes()))
	}
	y1, m1, d1 := now.Date()
	y2, m2, d2 := t.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return t.Format("15:04")
	}
	if now.AddDate(0, 0, -1).Format("2006-01-02") == t.Format("2006-01-02") {
		return Tf("time.y", t.Format("15:04"))
	}
	return Tf("time.md", int(m2), d2)
}

func (it *Item) Meta(now time.Time) string {
	parts := []string{}
	if it.Source != "" {
		parts = append(parts, it.Source)
	}
	parts = append(parts, relTime(it.Time, now))
	switch {
	case it.Kind == KSecret:
		parts = append(parts, T(it.Reason))
	case it.Kind == KImage:
		parts = append(parts, fmt.Sprintf("%d×%d", it.W, it.H))
	case it.Kind == KFile:
		parts = append(parts, Tf("meta.files", it.lines))
	case it.lines > 1:
		parts = append(parts, Tf("meta.lines", it.lines))
	case it.runes > 60:
		parts = append(parts, Tf("meta.chars", it.runes))
	}
	return strings.Join(parts, " · ")
}

var friendlyNames = map[string]string{
	"chrome": "Chrome", "msedge": "Edge", "firefox": "Firefox", "code": "VS Code",
	"qq": "QQ", "tim": "TIM", "dingtalk": "钉钉", "notepad++": "Notepad++",
	"winword": "Word", "excel": "Excel", "powerpnt": "PowerPoint", "outlook": "Outlook",
	"devenv": "Visual Studio", "idea64": "IntelliJ", "pycharm64": "PyCharm", "powershell": "PowerShell",
	"wps": "WPS", "et": "WPS 表格", "wpp": "WPS 演示", "feishu": "飞书", "lark": "Lark",
	"telegram": "Telegram", "slack": "Slack", "obsidian": "Obsidian", "typora": "Typora",
}

var localizedNames = map[string]string{
	"weixin": "src.wechat", "wechat": "src.wechat", "explorer": "src.explorer", "notepad": "src.notepad",
	"windowsterminal": "src.terminal", "cmd": "src.cmd",
}

// exeKey 返回小写、不含 .exe 的进程名。
func exeKey(exe string) string {
	return strings.TrimSuffix(strings.ToLower(exe), ".exe")
}

func friendlySource(exe string) string {
	k := exeKey(exe)
	if k == "" {
		return ""
	}
	if key, ok := localizedNames[k]; ok {
		return T(key)
	}
	if n, ok := friendlyNames[k]; ok {
		return n
	}
	rs := []rune(k)
	return strings.ToUpper(string(rs[:1])) + string(rs[1:])
}

// maskSecret 生成敏感内容的遮罩预览:只露出开头少量字符。
func maskSecret(s string) string {
	rs := []rune(strings.TrimSpace(s))
	keep := 3
	if len(rs) < 8 {
		keep = 1
	}
	return string(rs[:minInt(keep, len(rs))]) + "••••••••••"
}
