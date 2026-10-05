package main

import (
	"math"
	"strings"
	"time"
)

// ---------- 设计尺寸(DIP) ----------

const (
	designW      = 480.0
	designH      = 640.0
	designMargin = 24.0
	panelRadius  = 22.0
	cardGap      = 8.0
	listTop      = 148.0
	footerH      = 36.0
)

type TextStyle struct {
	Size int // 像素
	Bold bool
}

const (
	AlignLeft = iota
	AlignCenter
	AlignRight
)

// TextRenderer 由平台层实现(Windows 为 GDI)。
type TextRenderer interface {
	Draw(s string, r Rect, st TextStyle, col Color, align, lines int)
	Measure(s string, st TextStyle) int
	SetClip(r Rect)
}

// ---------- 交互动作 ----------

const (
	ActNone = iota
	ActDrag
	ActClose
	ActSettings
	ActBack
	ActPause
	ActClearQuery
	ActChip
	ActItem
	ActCopy
	ActPin
	ActDel
	ActSet
	ActSetVal
	ActPopBack
	ActPopPanel
	ActPopItem
	ActKeep
)

// 对条目的操作
const (
	OpPaste = iota
	OpCopy
	OpPlain
	OpUpper
	OpLower
)

type Hit struct {
	R        Rect
	Act, Arg int
}

// 设置项 ID
const (
	SetLang = iota
	SetHotkey
	SetPauseKey
	SetPos
	SetAutoPaste
	SetAutoStart
	SetGlass
	SetBlur
	SetTheme
	SetAccent
	SetScale
	SetCompact
	SetAnim
	SetSensitive
	SetIgnorePwd
	SetIgnoreList
	SetImages
	SetRetention
	SetMaxItems
	SetOpenFolder
	SetExport
	SetClearAll
	SetOpacity
	SetKeepOpen
	SetBall
)

// 主题色板:{深色模式, 浅色模式}
var accents = [][2]Color{
	{{86, 150, 255}, {0, 108, 235}},
	{{164, 124, 255}, {112, 64, 232}},
	{{52, 205, 145}, {10, 148, 100}},
	{{255, 164, 60}, {214, 110, 0}},
	{{255, 112, 170}, {214, 44, 112}},
	{{255, 98, 98}, {214, 52, 52}},
}

// PopItem 是弹出菜单的一项。
type PopItem struct {
	ID             int
	Label          string
	Checked, Sep   bool
	Danger, HasDot bool
	Dot            Color
}

type Popup struct {
	Items  []PopItem
	AX, AY int // 锚点(窗口像素坐标)
	Sel    int
	OnPick func(id int)
}

type UI struct {
	Pop *Popup

	S          float64
	Dark       bool
	Store      *Store
	Set        *Settings
	Query      []rune
	Filter     int
	Sel        int
	View       []*Item
	Scroll     float64
	ScrollTo   float64
	SettingsOn bool
	CaretOn    bool
	Recording  int // 正在录制的设置项,-1 为无

	ChipScroll float64 // 分类标签行横向滚动(DIP)
	chipMax    float64
	chipEnsure bool
	Naming     int // 0 否;-1 新建分区;>0 重命名该分区
	Name       []rune
	AssignNew  int64 // 新建分区后自动移入该条目

	Hover    Hit
	MX, MY   int
	toast    string
	toastEnd time.Time
	confirm  time.Time
	Dirty    bool

	enterStart time.Time
	entering   bool

	icon     *Canvas
	iconSize int

	hits    []Hit
	hitClip Rect

	// 回调(由宿主设置)
	OnAction      func(it *Item, op int)
	OnHide        func()
	OnSetting     func(id int)
	OnPauseToggle func()
	OnContext     func(it *Item)
	OnHotkey      func(id int, spec HotkeySpec) bool
	OnRecording   func(on bool)
	OnChipMenu    func(filter int) // 右键点击分区标签
	OnLangMenu    func()
}

func NewUI(s float64, st *Store, set *Settings) *UI {
	u := &UI{S: s, Store: st, Set: set, CaretOn: true, MX: -1, MY: -1, Recording: -1}
	u.Rebuild()
	return u
}

func (u *UI) px(v float64) int      { return int(math.Round(v * u.S)) }
func (u *UI) fpx(v float64) float64 { return v * u.S }

func (u *UI) WinSize() (int, int) {
	return u.px(designW + designMargin*2), u.px(designH + designMargin*2)
}

func (u *UI) InnerRect() Rect {
	m := u.px(designMargin)
	return Rect{m, m, u.px(designW), u.px(designH)}
}

func (u *UI) cardH() float64 {
	if u.Set.Compact {
		return 54
	}
	return 74
}

func (u *UI) accent() Color {
	a := accents[clampi(u.Set.Accent, 0, len(accents)-1)]
	if u.Dark {
		return a[0]
	}
	return a[1]
}

// OpenPopup 在鼠标位置弹出贴合主题的玻璃菜单。
func (u *UI) OpenPopup(items []PopItem, onPick func(id int)) {
	p := &Popup{Items: items, AX: u.MX, AY: u.MY, Sel: -1, OnPick: onPick}
	for i, it := range items {
		if it.Checked && !it.Sep {
			p.Sel = i
		}
	}
	u.Pop = p
	u.Hover = Hit{}
	u.Dirty = true
}

func (u *UI) ClosePopup() {
	if u.Pop != nil {
		u.Pop = nil
		u.Dirty = true
	}
}

func (u *UI) pickPopup(i int) {
	p := u.Pop
	if p == nil || i < 0 || i >= len(p.Items) || p.Items[i].Sep {
		return
	}
	id := p.Items[i].ID
	u.Pop = nil
	u.Dirty = true
	if p.OnPick != nil {
		p.OnPick(id)
	}
}

func (u *UI) popMove(d int) {
	p := u.Pop
	n := len(p.Items)
	i := p.Sel
	for k := 0; k < n; k++ {
		i = ((i+d)%n + n) % n
		if !p.Items[i].Sep {
			p.Sel = i
			break
		}
	}
	u.Dirty = true
}

func (u *UI) Reset() {
	u.Pop = nil
	u.CancelRecording()
	u.View = nil
	u.MX, u.MY = -1, -1
	u.Query = nil
	u.Filter = 0
	u.Naming, u.Name = 0, nil
	u.ChipScroll = 0
	u.Sel = 0
	u.Scroll, u.ScrollTo = 0, 0
	u.SettingsOn = false
	u.Hover = Hit{}
	u.enterStart = time.Now()
	u.entering = u.Set.Anim
	u.Rebuild()
	u.Dirty = true
}

func (u *UI) Rebuild() {
	var keepID int64 = -1
	if u.Sel >= 0 && u.Sel < len(u.View) {
		keepID = u.View[u.Sel].ID
	}
	u.View = u.Store.View(u.Filter, string(u.Query))
	u.Sel = 0
	for i, it := range u.View {
		if it.ID == keepID {
			u.Sel = i
			break
		}
	}
	u.clampScroll()
	u.Dirty = true
}

func (u *UI) Toast(s string) { u.ToastFor(s, 1500*time.Millisecond) }

func (u *UI) ToastFor(s string, d time.Duration) {
	u.toast = s
	u.toastEnd = time.Now().Add(d)
	u.Dirty = true
}

func (u *UI) HasToast() bool { return u.toast != "" }

func (u *UI) top() float64 {
	if u.SettingsOn {
		return 64
	}
	return listTop
}

func (u *UI) viewportH() float64 { return u.fpx(designH - u.top() - footerH) }

func (u *UI) contentH() float64 {
	if u.SettingsOn {
		h := 24.0
		for _, r := range u.rows() {
			h += r.height()
		}
		return u.fpx(h)
	}
	n := float64(len(u.View))
	if n == 0 {
		return 0
	}
	return u.fpx(n*(u.cardH()+cardGap) - cardGap)
}

func (u *UI) maxScroll() float64 { return math.Max(0, u.contentH()-u.viewportH()) }

func (u *UI) clampScroll() {
	u.ScrollTo = math.Max(0, math.Min(u.ScrollTo, u.maxScroll()))
	u.Scroll = math.Max(0, math.Min(u.Scroll, u.maxScroll()))
}

func (u *UI) ensureVisible() {
	if u.SettingsOn || len(u.View) == 0 {
		return
	}
	top := u.fpx(float64(u.Sel) * (u.cardH() + cardGap))
	bot := top + u.fpx(u.cardH())
	vh := u.viewportH()
	if top < u.ScrollTo {
		u.ScrollTo = top - u.fpx(4)
	} else if bot > u.ScrollTo+vh {
		u.ScrollTo = bot - vh + u.fpx(4)
	}
	u.clampScroll()
}

// Tick 推进动画,返回是否仍需继续刷新。
func (u *UI) Tick() bool {
	active := false
	if d := u.ScrollTo - u.Scroll; math.Abs(d) > 0.5 {
		u.Scroll += d * 0.28
		active = true
		u.Dirty = true
	} else if u.Scroll != u.ScrollTo {
		u.Scroll = u.ScrollTo
		u.Dirty = true
	}
	if u.entering {
		active = true
		u.Dirty = true
	}
	if u.toast != "" {
		if time.Now().After(u.toastEnd) {
			u.toast = ""
			u.Dirty = true
		} else {
			active = true
		}
	}
	return active
}

// Animating 供宿主判断是否需要启动动画定时器。
func (u *UI) Animating() bool {
	return u.entering || u.toast != "" || u.Scroll != u.ScrollTo
}

// ---------- 输入 ----------

func (u *UI) HitAt(x, y int) Hit {
	for i := len(u.hits) - 1; i >= 0; i-- {
		if u.hits[i].R.Contains(x, y) {
			return u.hits[i]
		}
	}
	return Hit{}
}

func isItemAct(a int) bool { return a == ActItem || a == ActCopy || a == ActPin || a == ActDel }

func (u *UI) MouseMove(x, y int) {
	u.MX, u.MY = x, y
	h := u.HitAt(x, y)
	if h != u.Hover {
		u.Hover = h
		if isItemAct(h.Act) {
			u.Sel = h.Arg
		}
		if h.Act == ActPopItem && u.Pop != nil {
			u.Pop.Sel = h.Arg
		}
		u.Dirty = true
	}
}

func (u *UI) MouseLeave() {
	u.MX, u.MY = -1, -1
	if u.Hover != (Hit{}) {
		u.Hover = Hit{}
		u.Dirty = true
	}
}

func (u *UI) Wheel(delta int) {
	if u.Pop != nil {
		return
	}
	in := u.InnerRect()
	if !u.SettingsOn && u.MY >= in.Y+u.px(106) && u.MY < in.Y+u.px(142) {
		u.ChipScroll -= float64(delta) / 120 * 90
		u.Dirty = true
		return
	}
	step := u.cardH() + cardGap
	if u.SettingsOn {
		step = 62
	}
	u.ScrollTo -= float64(delta) / 120 * u.fpx(step) * 1.05
	u.clampScroll()
	u.Dirty = true
}

func (u *UI) Click(h Hit, right bool) {
	u.Dirty = true
	if u.Pop != nil {
		switch {
		case h.Act == ActPopItem && !right:
			u.pickPopup(h.Arg)
		case h.Act == ActPopPanel && !right:
		default:
			u.ClosePopup()
		}
		return
	}
	if u.Recording >= 0 && h.Act != ActNone {
		u.CancelRecording()
	}
	switch h.Act {
	case ActKeep:
		u.Set.KeepOpen = !u.Set.KeepOpen
		if u.Set.KeepOpen {
			u.Toast(T("t.pinon"))
		} else {
			u.Toast(T("t.pinoff"))
		}
		u.fireSetting(SetKeepOpen)
	case ActClose:
		u.OnHide()
	case ActSettings:
		u.SettingsOn = true
		u.Scroll, u.ScrollTo = 0, 0
	case ActBack:
		u.CancelRecording()
		u.SettingsOn = false
		u.Scroll, u.ScrollTo = 0, 0
		u.Rebuild()
	case ActPause:
		u.OnPauseToggle()
	case ActClearQuery:
		u.Query = nil
		u.Rebuild()
	case ActChip:
		if h.Arg == -1 {
			u.BeginNaming(-1)
			return
		}
		if right {
			if h.Arg >= FilterGroupBase && u.OnChipMenu != nil {
				u.OnChipMenu(h.Arg)
			}
			return
		}
		u.CancelNaming()
		u.SetFilter(h.Arg)
	case ActItem:
		if h.Arg >= 0 && h.Arg < len(u.View) {
			u.Sel = h.Arg
			if right {
				u.OnContext(u.View[h.Arg])
			} else {
				u.OnAction(u.View[h.Arg], OpPaste)
			}
		}
	case ActCopy:
		if h.Arg >= 0 && h.Arg < len(u.View) {
			u.OnAction(u.View[h.Arg], OpCopy)
		}
	case ActPin:
		u.togglePin(h.Arg)
	case ActDel:
		u.deleteAt(h.Arg)
	case ActSet:
		u.applySetting(h.Arg)
	case ActSetVal:
		if h.Arg>>8 == SetAccent {
			u.Set.Accent = h.Arg & 0xff
			u.fireSetting(SetAccent)
		}
	}
}

func (u *UI) SetFilter(f int) {
	u.Filter = f
	u.chipEnsure = true
	u.Scroll, u.ScrollTo = 0, 0
	u.Sel = 0
	u.Rebuild()
}

// ---------- 分区命名 ----------

func (u *UI) BeginNaming(id int) {
	u.Naming = id
	u.Name = nil
	if id > 0 {
		if g := u.Store.GroupByID(id); g != nil {
			u.Name = []rune(g.Name)
		}
	}
	u.Query = nil
	u.Rebuild()
	u.Dirty = true
}

func (u *UI) CancelNaming() {
	if u.Naming != 0 {
		u.Naming, u.Name, u.AssignNew = 0, nil, 0
		u.Dirty = true
	}
}

func (u *UI) confirmNaming() {
	name := strings.TrimSpace(string(u.Name))
	id := u.Naming
	u.Naming, u.Name = 0, nil
	u.Dirty = true
	if name == "" {
		return
	}
	if id == -1 {
		if g := u.Store.AddGroup(name); g != nil {
			u.Toast(Tf("t.gcreated", g.Name))
			if u.AssignNew != 0 {
				u.Store.SetItemGroup(u.AssignNew, g.ID)
				u.AssignNew = 0
			}
			u.SetFilter(FilterGroupBase + g.ID)
		}
	} else {
		u.Store.RenameGroup(id, name)
	}
}

// AssignGroup 把条目移动到分区(0 为无分区)。
func (u *UI) AssignGroup(it *Item, gid int) {
	u.Store.SetItemGroup(it.ID, gid)
	u.Rebuild()
}

type chip struct {
	f     int
	label string
	col   Color
	dot   bool
}

func (u *UI) chips() []chip {
	cs := []chip{{f: FilterAll, label: T("chip.all")}, {f: FilterPinned, label: T("chip.pinned")}}
	for _, g := range u.Store.Groups {
		a := accents[clampi(g.Color, 0, len(accents)-1)]
		col := a[1]
		if u.Dark {
			col = a[0]
		}
		cs = append(cs, chip{FilterGroupBase + g.ID, g.Name, col, true})
	}
	var cnt [kindCount]int
	for _, it := range u.Store.Items {
		cnt[it.Kind]++
	}
	for k := Kind(0); k < kindCount; k++ {
		if cnt[k] > 0 || u.Filter == FilterKindBase+int(k) {
			cs = append(cs, chip{f: FilterKindBase + int(k), label: T(kindChipKey[k])})
		}
	}
	return cs
}

func (u *UI) togglePin(i int) {
	if i < 0 || i >= len(u.View) {
		return
	}
	u.Store.TogglePin(u.View[i].ID)
	u.Rebuild()
}

func (u *UI) deleteAt(i int) {
	if i < 0 || i >= len(u.View) {
		return
	}
	u.Store.Delete(u.View[i].ID)
	sel := u.Sel
	u.Rebuild()
	u.Sel = clampi(sel, 0, maxi(len(u.View)-1, 0))
	u.Hover = Hit{}
}

func (u *UI) fireSetting(id int) {
	if u.OnSetting != nil {
		u.OnSetting(id)
	}
	u.Dirty = true
}

func (u *UI) applySetting(id int) {
	s := u.Set
	switch id {
	case SetLang:
		if u.OnLangMenu != nil {
			u.OnLangMenu()
		}
		return
	case SetHotkey, SetPauseKey:
		u.StartRecording(id)
		return
	case SetPos:
		s.PosMode = (s.PosMode + 1) % 3
	case SetAutoPaste:
		s.AutoPaste = !s.AutoPaste
	case SetAutoStart:
		s.AutoStart = !s.AutoStart
	case SetGlass:
		s.Glass = !s.Glass
	case SetBlur:
		s.Blur = (s.Blur + 1) % 3
	case SetTheme:
		s.Theme = (s.Theme + 1) % 3
	case SetAccent:
		s.Accent = (s.Accent + 1) % len(accents)
	case SetScale:
		s.UIScale = (s.UIScale + 1) % len(scaleVals)
	case SetCompact:
		s.Compact = !s.Compact
		u.Scroll, u.ScrollTo = 0, 0
	case SetAnim:
		s.Anim = !s.Anim
	case SetOpacity:
		s.Opacity = (s.Opacity + 1) % 3
	case SetKeepOpen:
		s.KeepOpen = !s.KeepOpen
	case SetBall:
		s.Ball = !s.Ball
	case SetSensitive:
		s.Sensitive = (s.Sensitive + 1) % 3
	case SetIgnorePwd:
		s.IgnorePwd = !s.IgnorePwd
	case SetIgnoreList:
		s.IgnoreSources = nil
		u.Toast(T("t.ignreset"))
	case SetImages:
		s.RecordImages = !s.RecordImages
	case SetRetention:
		s.Retention = retentionVals[(indexOf(retentionVals, s.Retention)+1)%len(retentionVals)]
	case SetMaxItems:
		s.MaxItems = maxItemVals[(indexOf(maxItemVals, s.MaxItems)+1)%len(maxItemVals)]
	case SetClearAll:
		if time.Now().Before(u.confirm) {
			u.confirm = time.Time{}
			u.Store.ClearUnpinned()
			u.Rebuild()
			u.Toast(T("t.cleared"))
		} else {
			u.confirm = time.Now().Add(3 * time.Second)
			u.Toast(T("t.clear2"))
		}
		return
	case SetOpenFolder, SetExport:
	}
	u.fireSetting(id)
}

// ---------- 快捷键录制 ----------

func (u *UI) StartRecording(id int) {
	u.Recording = id
	if u.OnRecording != nil {
		u.OnRecording(true)
	}
	u.ToastFor(T("t.hkrec"), 10*time.Second)
}

func (u *UI) CancelRecording() {
	if u.Recording >= 0 {
		u.Recording = -1
		if u.OnRecording != nil {
			u.OnRecording(false)
		}
		u.toast = ""
		u.Dirty = true
	}
}

func isModifierVK(vk int) bool {
	switch vk {
	case 0x10, 0x11, 0x12, 0x5B, 0x5C, 0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5:
		return true
	}
	return false
}

// RecordKey 由宿主在录制状态下调用。mods 使用 Mod* 位。
func (u *UI) RecordKey(vk int, mods uint32) {
	u.Dirty = true
	if isModifierVK(vk) {
		return
	}
	id := u.Recording
	if vk == 0x1B && mods == 0 {
		u.CancelRecording()
		return
	}
	var spec HotkeySpec
	switch {
	case (vk == 0x2E || vk == 0x08) && mods == 0 && id == SetPauseKey:
		u.finishRecording()
		u.OnHotkey(id, HotkeySpec{})
		u.Toast(T("t.hkcleared"))
		return
	case mods == 0:
		u.ToastFor(T("t.hkneed"), 2*time.Second)
		return
	default:
		spec = HotkeySpec{Mods: mods, VK: uint32(vk)}
	}
	u.finishRecording()
	if u.OnHotkey(id, spec) {
		u.Toast(Tf("t.hkset", spec.String()))
	} else {
		u.ToastFor(T("t.hkfail"), 2500*time.Millisecond)
	}
}

func (u *UI) finishRecording() {
	u.Recording = -1
	u.toast = ""
	if u.OnRecording != nil {
		u.OnRecording(false)
	}
}

// ---------- 键盘 ----------

func (u *UI) Char(r rune) {
	if u.SettingsOn || u.Recording >= 0 || r < 32 || r == 127 {
		return
	}
	if u.Naming != 0 {
		if len(u.Name) < 24 {
			u.Name = append(u.Name, r)
		}
		u.Dirty = true
		return
	}
	u.Query = append(u.Query, r)
	u.queryChanged()
}

func (u *UI) Backspace() {
	if u.Naming != 0 {
		if len(u.Name) > 0 {
			u.Name = u.Name[:len(u.Name)-1]
		}
		u.Dirty = true
		return
	}
	if len(u.Query) > 0 {
		u.Query = u.Query[:len(u.Query)-1]
		u.queryChanged()
	}
}

func (u *UI) queryChanged() {
	u.Sel = 0
	u.Scroll, u.ScrollTo = 0, 0
	u.View = u.Store.View(u.Filter, string(u.Query))
	u.Dirty = true
}

func (u *UI) moveSel(d int) {
	if len(u.View) == 0 {
		return
	}
	u.Sel = clampi(u.Sel+d, 0, len(u.View)-1)
	u.ensureVisible()
	u.Dirty = true
}

// Key 处理按键,返回是否已处理。
func (u *UI) Key(vk int, ctrl, shift, alt bool) bool {
	u.Dirty = true
	if u.Recording >= 0 {
		return true
	}
	if u.Pop != nil {
		switch vk {
		case 0x1B:
			u.ClosePopup()
		case 0x26:
			u.popMove(-1)
		case 0x28:
			u.popMove(1)
		case 0x0D:
			u.pickPopup(u.Pop.Sel)
		}
		return true
	}
	if u.Naming != 0 {
		switch vk {
		case 0x1B:
			u.CancelNaming()
			return true
		case 0x0D:
			u.confirmNaming()
			return true
		case 0x08:
			u.Backspace()
			return true
		}
		return false
	}
	switch vk {
	case 0x1B: // Esc
		switch {
		case u.SettingsOn:
			u.Click(Hit{Act: ActBack}, false)
		case len(u.Query) > 0:
			u.Query = nil
			u.queryChanged()
		default:
			u.OnHide()
		}
		return true
	case 0x08: // Backspace
		if ctrl {
			u.Query = nil
			u.queryChanged()
		} else {
			u.Backspace()
		}
		return true
	case 0x26:
		u.moveSel(-1)
		return true
	case 0x28:
		u.moveSel(1)
		return true
	case 0x21:
		u.moveSel(-5)
		return true
	case 0x22:
		u.moveSel(5)
		return true
	case 0x24:
		u.moveSel(-1 << 20)
		return true
	case 0x23:
		u.moveSel(1 << 20)
		return true
	case 0x0D: // Enter:Ctrl=仅复制 Shift=单行纯文本
		if !u.SettingsOn && u.Sel < len(u.View) {
			op := OpPaste
			if ctrl {
				op = OpCopy
			} else if shift {
				op = OpPlain
			}
			u.OnAction(u.View[u.Sel], op)
		}
		return true
	case 0x2E: // Delete
		if !u.SettingsOn {
			u.deleteAt(u.Sel)
		}
		return true
	case 0x09: // Tab
		if u.SettingsOn {
			return true
		}
		cs := u.chips()
		cur := 0
		for i, c := range cs {
			if c.f == u.Filter {
				cur = i
			}
		}
		d := 1
		if shift {
			d = len(cs) - 1
		}
		u.SetFilter(cs[(cur+d)%len(cs)].f)
		return true
	case 0x50: // P
		if ctrl && !u.SettingsOn {
			u.togglePin(u.Sel)
			return true
		}
	case 0xBC: // Ctrl+,
		if ctrl {
			u.SettingsOn = !u.SettingsOn
			u.Scroll, u.ScrollTo = 0, 0
			return true
		}
	}
	if ctrl && vk >= 0x30 && vk <= 0x39 && !u.SettingsOn && u.Sel < len(u.View) {
		n := vk - 0x30
		gid := 0
		if n >= 1 {
			if n > len(u.Store.Groups) {
				return true
			}
			gid = u.Store.Groups[n-1].ID
		}
		u.AssignGroup(u.View[u.Sel], gid)
		return true
	}
	if alt && vk >= 0x31 && vk <= 0x39 && !u.SettingsOn {
		i := vk - 0x31
		if i < len(u.View) {
			u.OnAction(u.View[i], OpPaste)
		}
		return true
	}
	u.Dirty = false
	return false
}

// CaretPos 返回搜索框光标的窗口内像素坐标(用于输入法候选框定位)。
func (u *UI) CaretPos(t TextRenderer) (int, int) {
	in := u.InnerRect()
	w := t.Measure(string(u.Query), TextStyle{Size: u.px(14)})
	return in.X + u.px(16+40) + w, in.Y + u.px(62+30)
}

// ---------- 配色 ----------

type Pal struct {
	Dark                          bool
	Text, Sub, Accent, Warn, Gold Color
	CardA, CardHoverA             float64
	CardCol                       Color
	ChipCol                       Color
	ChipA                         float64
	BorderCol                     Color
	BorderA                       float64
	Gloss                         float64
}

func (u *UI) pal() Pal {
	if u.Dark {
		return Pal{Dark: true, Text: Color{238, 241, 248}, Sub: Color{152, 162, 186}, Accent: u.accent(),
			Warn: Color{255, 170, 70}, Gold: Color{255, 200, 70},
			CardA: 0.07, CardHoverA: 0.13, CardCol: Color{255, 255, 255},
			ChipCol: Color{255, 255, 255}, ChipA: 0.09, BorderCol: Color{255, 255, 255}, BorderA: 0.10, Gloss: 0.10}
	}
	return Pal{Text: Color{22, 26, 38}, Sub: Color{92, 102, 124}, Accent: u.accent(),
		Warn: Color{214, 120, 10}, Gold: Color{232, 160, 0},
		CardA: 0.52, CardHoverA: 0.78, CardCol: Color{255, 255, 255},
		ChipCol: Color{0, 0, 0}, ChipA: 0.06, BorderCol: Color{0, 0, 0}, BorderA: 0.07, Gloss: 0.7}
}

// BuildBase 构建毛玻璃底图。snapshot 是窗口后方桌面的截图(内部区域大小),可为 nil。
func (u *UI) BuildBase(snapshot *Canvas) (*Canvas, []byte) {
	W, H := u.WinSize()
	in := u.InnerRect()
	var inner *Canvas
	if snapshot == nil || snapshot.W != in.W || snapshot.H != in.H {
		snapshot = fallbackBackdrop(in.W, in.H, u.Dark)
	}
	tint := Color{244, 247, 253}
	if u.Dark {
		tint = Color{14, 17, 26}
	}
	if u.Set.Glass {
		radii := []float64{10, 20, 34}
		inner = frost(snapshot, u.px(radii[u.Set.Blur]))
		ta := []float64{0.46, 0.56, 0.64}[u.Set.Blur] + []float64{-0.10, 0.10, 0.24}[clampi(u.Set.Opacity, 0, 2)]
		if !u.Dark {
			ta -= 0.04
		}
		ta = math.Min(ta, 0.95)
		tintAndSaturate(inner, tint, ta, 1.35, 2)
	} else {
		inner = NewCanvas(in.W, in.H)
		for i := 0; i < len(inner.Pix); i += 4 {
			inner.Pix[i], inner.Pix[i+1], inner.Pix[i+2], inner.Pix[i+3] = tint.B, tint.G, tint.R, 255
		}
		if u.Dark {
			tintAndSaturate(inner, Color{30, 34, 48}, 0.0, 1, 0)
		}
	}
	// 顶部高光:模拟玻璃反光
	a0 := 0.16
	if !u.Dark {
		a0 = 0.40
	}
	inner.VGradient(Rect{0, 0, in.W, in.H * 2 / 5}, Color{255, 255, 255}, a0, 0)
	// 强调色光晕,让玻璃更有层次
	if u.Set.Glass {
		glow := accents[clampi(u.Set.Accent, 0, len(accents)-1)][0]
		ga := 0.10
		if !u.Dark {
			ga = 0.08
		}
		for r := 0; r < 6; r++ {
			inner.FillCircle(float64(in.W)*0.9, float64(in.H)*0.06, u.fpx(160)-float64(r)*u.fpx(20), glow, ga/3)
		}
	}
	base := NewCanvas(W, H)
	for y := 0; y < in.H; y++ {
		copy(base.Pix[((y+in.Y)*W+in.X)*4:((y+in.Y)*W+in.X+in.W)*4], inner.Pix[y*in.W*4:(y+1)*in.W*4])
	}
	finalizeFrame(base, in.X, in.W, in.H, u.fpx(panelRadius), u.Dark, u.fpx(designMargin-4))
	mask := make([]byte, W*H)
	for i := range mask {
		mask[i] = base.Pix[i*4+3]
	}
	return base, mask
}

// ---------- 绘制 ----------

func (u *UI) addHit(r Rect, act, arg int) {
	r = r.Intersect(u.hitClip)
	if r.Empty() {
		return
	}
	u.hits = append(u.hits, Hit{R: r, Act: act, Arg: arg})
}

func (u *UI) setClip(c *Canvas, t TextRenderer, r Rect) {
	c.SetClip(r)
	t.SetClip(c.Clip)
	u.hitClip = c.Clip
}

func (u *UI) fitStyle(t TextRenderer, s string, maxW int, size float64, bold bool) TextStyle {
	for sz := size; sz > 8; sz -= 0.5 {
		st := TextStyle{Size: int(math.Round(sz * u.S)), Bold: bold}
		if t.Measure(s, st) <= maxW {
			return st
		}
	}
	return TextStyle{Size: int(math.Round(8 * u.S)), Bold: bold}
}

func (u *UI) iconCanvas(size int) *Canvas {
	if u.icon == nil || u.iconSize != size {
		u.icon, u.iconSize = drawAppIcon(size), size
	}
	return u.icon
}

func (u *UI) Draw(c *Canvas, t TextRenderer) {
	u.hits = u.hits[:0]
	u.entering = false
	in := u.InnerRect()
	p := u.pal()
	S := u.S
	now := time.Now()

	R := func(x, y, w, h float64) Rect {
		return Rect{in.X + int(math.Round(x*S)), in.Y + int(math.Round(y*S)), int(math.Round(w * S)), int(math.Round(h * S))}
	}
	fx := func(v float64) float64 { return float64(in.X) + v*S }
	fy := func(v float64) float64 { return float64(in.Y) + v*S }
	ts := func(size float64, bold bool) TextStyle { return TextStyle{Size: int(math.Round(size * S)), Bold: bold} }

	u.setClip(c, t, in)

	// ---- 头部 ----
	u.addHit(R(0, 0, designW, 56), ActDrag, 0)
	ic := u.iconCanvas(u.px(30))
	c.Blit(ic, in.X+u.px(18), in.Y+u.px(13))
	title := T("title")
	if u.SettingsOn {
		title = T("settings")
	}
	t.Draw(title, R(58, 9, 200, 26), ts(17, true), p.Text, AlignLeft, 1)
	if !u.SettingsOn {
		t.Draw(Tf("count", len(u.Store.Items)), R(58, 33, 200, 16), ts(11, false), p.Sub, AlignLeft, 1)
	}

	pillW := func(label string) float64 { return float64(t.Measure(label, ts(12, false)))/S + 26 }
	pill := func(x, w float64, label string, col Color, act int) {
		r := R(x, 14, w, 28)
		a := p.ChipA
		if u.Hover.Act == act {
			a *= 2
		}
		c.FillRRect(float64(r.X), float64(r.Y), float64(r.W), float64(r.H), float64(r.H)/2, p.ChipCol, a)
		t.Draw(label, r, ts(12, false), col, AlignCenter, 1)
		u.addHit(r, act, 0)
	}
	closeX := designW - 16 - 28
	cr := R(closeX, 14, 28, 28)
	ca := p.ChipA
	if u.Hover.Act == ActClose {
		ca *= 2.2
	}
	c.FillCircle(float64(cr.X)+float64(cr.W)/2, float64(cr.Y)+float64(cr.H)/2, float64(cr.W)/2, p.ChipCol, ca)
	u.drawX(c, float64(cr.X)+float64(cr.W)/2, float64(cr.Y)+float64(cr.H)/2, u.fpx(4.5), p.Sub)
	u.addHit(cr, ActClose, 0)

	if u.SettingsOn {
		lbl := T("back")
		w := pillW(lbl)
		pill(closeX-8-w, w, lbl, p.Accent, ActBack)
	} else {
		sl := T("settings")
		sw := pillW(sl)
		pill(closeX-8-sw, sw, sl, p.Sub, ActSettings)
		pl, pc := T("pause"), p.Sub
		if u.Set.Paused {
			pl, pc = T("resume"), p.Warn
		}
		pw := pillW(pl)
		pill(closeX-8-sw-8-pw, pw, pl, pc, ActPause)
		// 钉住按钮
		kr := R(closeX-8-sw-8-pw-8-28, 14, 28, 28)
		kcx, kcy := float64(kr.X)+float64(kr.W)/2, float64(kr.Y)+float64(kr.H)/2
		if u.Set.KeepOpen {
			c.FillCircle(kcx, kcy, float64(kr.W)/2, p.Accent, 0.95)
		} else {
			ka := p.ChipA
			if u.Hover.Act == ActKeep {
				ka *= 2.2
			}
			c.FillCircle(kcx, kcy, float64(kr.W)/2, p.ChipCol, ka)
		}
		kc := p.Sub
		if u.Set.KeepOpen {
			kc = Color{255, 255, 255}
		}
		// 图钉:圆头 + 针
		u.drawPin(c, kcx, kcy, kc, 0.95)
		u.addHit(kr, ActKeep, 0)
	}

	if !u.SettingsOn {
		// ---- 搜索框 ----
		sr := R(16, 62, designW-32, 40)
		rx, ry, rw, rh := float64(sr.X), float64(sr.Y), float64(sr.W), float64(sr.H)
		c.FillRRect(rx, ry, rw, rh, rh/2, p.ChipCol, p.ChipA*1.1)
		if len(u.Query) > 0 || u.Naming != 0 {
			c.StrokeRRect(rx-u.fpx(1.5), ry-u.fpx(1.5), rw+u.fpx(3), rh+u.fpx(3), rh/2+u.fpx(1.5), u.fpx(3), p.Accent, 0.16)
			c.StrokeRRect(rx, ry, rw, rh, rh/2, u.fpx(1.2), p.Accent, 0.7)
		} else {
			c.StrokeRRect(rx, ry, rw, rh, rh/2, u.fpx(1), p.BorderCol, p.BorderA)
		}
		mx, my := fx(16+17), fy(62+20)
		c.StrokeCircle(mx-u.fpx(1.5), my-u.fpx(1.5), u.fpx(5), u.fpx(1.6), p.Sub, 0.95)
		c.Line(mx+u.fpx(2), my+u.fpx(2), mx+u.fpx(6.5), my+u.fpx(6.5), u.fpx(1.7), p.Sub, 0.95)
		q := string(u.Query)
		ph := T("search")
		if u.Naming != 0 {
			q, ph = string(u.Name), T("g.name.ph")
		}
		tr := R(16+42, 62, designW-32-42-40, 40)
		if q == "" {
			t.Draw(ph, tr, u.fitStyle(t, ph, tr.W, 14, false), p.Sub, AlignLeft, 1)
		} else {
			t.Draw(q, tr, ts(14, false), p.Text, AlignLeft, 1)
		}
		if u.CaretOn {
			cx := float64(tr.X) + float64(minInt(t.Measure(q, ts(14, false)), tr.W)) + u.fpx(1)
			c.Line(cx, fy(62+11), cx, fy(62+29), u.fpx(1.4), p.Accent, 0.95)
		}
		if q != "" && u.Naming == 0 {
			xr := R(designW-16-34, 62+3, 34, 34)
			if u.Hover.Act == ActClearQuery {
				c.FillCircle(float64(xr.X)+float64(xr.W)/2, float64(xr.Y)+float64(xr.H)/2, u.fpx(11), p.ChipCol, p.ChipA*1.6)
			}
			u.drawX(c, float64(xr.X)+float64(xr.W)/2, float64(xr.Y)+float64(xr.H)/2, u.fpx(3.6), p.Sub)
			u.addHit(xr, ActClearQuery, 0)
		}

		// ---- 分类标签(可横向滚动)----
		cst := ts(12, false)
		cs := u.chips()
		const padEach, gapC = 11.0, 5.0
		areaW := designW - 32 - 34
		total, activeX, activeW := 0.0, 0.0, 0.0
		ws := make([]float64, len(cs))
		for i, ch := range cs {
			w := float64(t.Measure(ch.label, cst))/S + padEach*2
			if ch.dot {
				w += 12
			}
			ws[i] = w
			if ch.f == u.Filter {
				activeX, activeW = total, w
			}
			total += w + gapC
		}
		total -= gapC
		u.chipMax = math.Max(0, total-areaW)
		if u.chipEnsure {
			u.chipEnsure = false
			if activeX < u.ChipScroll {
				u.ChipScroll = activeX - 8
			} else if activeX+activeW > u.ChipScroll+areaW {
				u.ChipScroll = activeX + activeW - areaW + 8
			}
		}
		u.ChipScroll = math.Max(0, math.Min(u.ChipScroll, u.chipMax))
		u.setClip(c, t, R(16, 106, areaW, 36))
		x := 16.0 - u.ChipScroll
		for i, ch := range cs {
			w := ws[i]
			r := R(x, 110, w, 28)
			active := u.Filter == ch.f
			hov := u.Hover.Act == ActChip && u.Hover.Arg == ch.f
			if active {
				c.FillRRect(float64(r.X), float64(r.Y), float64(r.W), float64(r.H), float64(r.H)/2, p.Accent, 0.95)
			} else {
				a := p.ChipA
				if hov {
					a *= 2
				}
				c.FillRRect(float64(r.X), float64(r.Y), float64(r.W), float64(r.H), float64(r.H)/2, p.ChipCol, a)
			}
			col := p.Sub
			if active {
				col = Color{255, 255, 255}
			} else if hov {
				col = p.Text
			}
			lr := r
			if ch.dot {
				dc := ch.col
				if active {
					dc = Color{255, 255, 255}
				}
				c.FillCircle(float64(r.X)+u.fpx(padEach+3), float64(r.Y)+float64(r.H)/2, u.fpx(3.5), dc, 1)
				lr = Rect{r.X + u.px(12), r.Y, r.W - u.px(12), r.H}
			}
			t.Draw(ch.label, lr, ts(12, active), col, AlignCenter, 1)
			u.addHit(r, ActChip, ch.f)
			x += w + gapC
		}
		// 固定在右侧的“+”新建分区
		u.setClip(c, t, in)
		pr := R(designW-16-28, 110, 28, 28)
		pa := p.ChipA
		if u.Hover.Act == ActChip && u.Hover.Arg == -1 {
			pa *= 2.4
		}
		if u.Naming == -1 {
			c.FillCircle(float64(pr.X+pr.W/2), float64(pr.Y+pr.H/2), float64(pr.W)/2, p.Accent, 0.95)
		} else {
			c.FillCircle(float64(pr.X+pr.W/2), float64(pr.Y+pr.H/2), float64(pr.W)/2, p.ChipCol, pa)
		}
		pc := p.Sub
		if u.Naming == -1 {
			pc = Color{255, 255, 255}
		}
		pcx, pcy := float64(pr.X)+float64(pr.W)/2, float64(pr.Y)+float64(pr.H)/2
		c.Line(pcx-u.fpx(5), pcy, pcx+u.fpx(5), pcy, u.fpx(1.6), pc, 0.95)
		c.Line(pcx, pcy-u.fpx(5), pcx, pcy+u.fpx(5), u.fpx(1.6), pc, 0.95)
		u.addHit(pr, ActChip, -1)
	}

	// ---- 列表 / 设置 ----
	vp := R(0, u.top()-4, designW, designH-u.top()-footerH+4)
	u.setClip(c, t, vp)
	if u.SettingsOn {
		u.drawSettings(c, t, p, in, ts)
	} else {
		u.drawList(c, t, p, in, now, R, fx, ts)
	}
	if mx := u.maxScroll(); mx > 0 {
		vh := u.viewportH()
		th := math.Max(u.fpx(28), vh*vh/(vh+mx))
		ty := float64(in.Y) + u.fpx(u.top()) + (vh-th)*(u.Scroll/mx)
		c.FillRRect(float64(in.X)+u.fpx(designW-7), ty, u.fpx(3.5), th, u.fpx(1.75), p.ChipCol, 0.28)
	}

	// ---- 底栏 ----
	u.setClip(c, t, in)
	hint := T("hint.list")
	if u.SettingsOn {
		hint = T("hint.set")
	}
	hr := R(14, designH-footerH+2, designW-28, footerH-6)
	t.Draw(hint, hr, u.fitStyle(t, hint, hr.W, 11, false), p.Sub, AlignCenter, 1)

	u.drawPopup(c, t, p, in, R)

	// ---- Toast ----
	if u.toast != "" {
		st := u.fitStyle(t, u.toast, u.px(designW-80), 13, false)
		w := float64(t.Measure(u.toast, st))/S + 36
		r := R((designW-w)/2, designH-footerH-48, w, 32)
		c.FillRRect(float64(r.X)+1, float64(r.Y)+u.fpx(2), float64(r.W), float64(r.H), float64(r.H)/2, Color{0, 0, 0}, 0.18)
		c.FillRRect(float64(r.X), float64(r.Y), float64(r.W), float64(r.H), float64(r.H)/2, Color{28, 32, 44}, 0.94)
		t.Draw(u.toast, r, st, Color{245, 247, 252}, AlignCenter, 1)
	}
}

func (u *UI) drawPopup(c *Canvas, t TextRenderer, p Pal, in Rect, R func(x, y, w, h float64) Rect) {
	pp := u.Pop
	if pp == nil {
		return
	}
	S := u.S
	u.setClip(c, t, in)
	scrim := 0.20
	if u.Dark {
		scrim = 0.34
	}
	c.FillRRect(float64(in.X), float64(in.Y), float64(in.W), float64(in.H), u.fpx(panelRadius), Color{0, 0, 0}, scrim)
	u.addHit(in, ActPopBack, 0)

	st := TextStyle{Size: int(math.Round(13 * S))}
	w := 170.0
	for _, it := range pp.Items {
		if it.Sep {
			continue
		}
		lw := float64(t.Measure(it.Label, st))/S + 64
		if lw > w {
			w = lw
		}
	}
	w = math.Min(w, designW-24)
	rowH, sepH := 34.0, 11.0
	h := 12.0
	n := 0
	for _, it := range pp.Items {
		if it.Sep {
			h += sepH
		} else {
			h += rowH
			n++
		}
	}
	if max := designH - 24; h > max {
		rowH = math.Floor((max - 12 - sepH*float64(len(pp.Items)-n)) / float64(n))
		h = max
	}
	ax, ay := float64(pp.AX-in.X)/S, float64(pp.AY-in.Y)/S
	x := math.Max(12, math.Min(ax-10, designW-12-w))
	y := math.Max(12, math.Min(ay-6, designH-12-h))
	pr := R(x, y, w, h)
	fr := func(v float64) float64 { return v * S }
	c.FillRRect(float64(pr.X), float64(pr.Y)+fr(5), float64(pr.W), float64(pr.H), fr(16), Color{0, 0, 0}, 0.30)
	fill, fa := Color{252, 253, 255}, 0.98
	if u.Dark {
		fill, fa = Color{30, 34, 50}, 0.97
	}
	c.FillRRect(float64(pr.X), float64(pr.Y), float64(pr.W), float64(pr.H), fr(16), fill, fa)
	c.VGradient(Rect{pr.X, pr.Y, pr.W, pr.H / 3}, Color{255, 255, 255}, p.Gloss*0.25, 0)
	c.StrokeRRect(float64(pr.X), float64(pr.Y), float64(pr.W), float64(pr.H), fr(16), fr(1), p.BorderCol, p.BorderA*1.8)
	u.addHit(pr, ActPopPanel, 0)

	yy := y + 6
	for i, it := range pp.Items {
		if it.Sep {
			c.Line(float64(pr.X)+fr(14), float64(R(0, yy+sepH/2, 1, 1).Y), float64(pr.X+pr.W)-fr(14), float64(R(0, yy+sepH/2, 1, 1).Y), fr(1), p.BorderCol, p.BorderA*2)
			yy += sepH
			continue
		}
		rr := R(x+6, yy, w-12, rowH)
		col := p.Text
		if it.Danger {
			col = Color{232, 70, 70}
		}
		if i == pp.Sel {
			if it.Danger {
				c.FillRRect(float64(rr.X), float64(rr.Y), float64(rr.W), float64(rr.H), fr(10), Color{232, 70, 70}, 0.14)
			} else {
				c.FillRRect(float64(rr.X), float64(rr.Y), float64(rr.W), float64(rr.H), fr(10), p.Accent, 0.92)
				col = Color{255, 255, 255}
			}
		}
		tx := rr.X + u.px(12)
		if it.HasDot {
			dc := it.Dot
			if i == pp.Sel && !it.Danger {
				dc = Color{255, 255, 255}
			}
			c.FillCircle(float64(tx)+fr(4), float64(rr.Y)+float64(rr.H)/2, fr(4.5), dc, 1)
			tx += u.px(16)
		}
		t.Draw(it.Label, Rect{tx, rr.Y, rr.X + rr.W - u.px(34) - tx, rr.H}, st, col, AlignLeft, 1)
		if it.Checked {
			cc := p.Accent
			if i == pp.Sel {
				cc = Color{255, 255, 255}
			}
			kx, ky := float64(rr.X+rr.W)-fr(20), float64(rr.Y)+float64(rr.H)/2
			c.Line(kx-fr(4.5), ky, kx-fr(1.5), ky+fr(3.5), fr(1.8), cc, 1)
			c.Line(kx-fr(1.5), ky+fr(3.5), kx+fr(5), ky-fr(3.5), fr(1.8), cc, 1)
		}
		u.addHit(rr, ActPopItem, i)
		yy += rowH
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// drawPin 绘制倾斜的图钉图标(置顶 / 钉住共用)。
func (u *UI) drawPin(c *Canvas, cx, cy float64, col Color, a float64) {
	s := u.S
	ux, uy := 0.57, -0.82 // 钉头方向
	px, py := 0.82, 0.57
	bx, by := cx-ux*1.5*s, cy-uy*1.5*s
	tx, ty := cx-ux*7.5*s, cy-uy*7.5*s
	c.Line(tx, ty, bx, by, 1.3*s, col, a)
	c.Line(bx-px*4.8*s, by-py*4.8*s, bx+px*4.8*s, by+py*4.8*s, 1.9*s, col, a)
	c.Line(bx, by, bx+ux*4.6*s, by+uy*4.6*s, 4.4*s, col, a)
	c.Line(bx+ux*4.6*s, by+uy*4.6*s, bx+ux*7.4*s, by+uy*7.4*s, 6.2*s, col, a)
}

func (u *UI) drawX(c *Canvas, cx, cy, r float64, col Color) {
	c.Line(cx-r, cy-r, cx+r, cy+r, u.fpx(1.6), col, 0.95)
	c.Line(cx-r, cy+r, cx+r, cy-r, u.fpx(1.6), col, 0.95)
}

func (u *UI) drawCopyIcon(c *Canvas, cx, cy float64, col Color) {
	s := u.S
	c.StrokeRRect(cx-5.5*s, cy-2.5*s, 8*s, 9*s, 2*s, 1.3*s, col, 0.9)
	c.StrokeRRect(cx-2.5*s, cy-5.5*s, 8*s, 9*s, 2*s, 1.3*s, col, 0.9)
}

func (u *UI) drawList(c *Canvas, t TextRenderer, p Pal, in Rect, now time.Time,
	R func(x, y, w, h float64) Rect, fx func(float64) float64, ts func(float64, bool) TextStyle) {

	if len(u.View) == 0 {
		msg, sub := T("empty.t"), T("empty.s")
		if len(u.Query) > 0 || u.Filter != 0 {
			msg, sub = T("nores.t"), T("nores.s")
		}
		t.Draw(msg, R(0, listTop+90, designW, 28), ts(16, true), p.Text, AlignCenter, 1)
		sr := R(20, listTop+122, designW-40, 22)
		t.Draw(sub, sr, u.fitStyle(t, sub, sr.W, 12, false), p.Sub, AlignCenter, 1)
		return
	}
	compact := u.Set.Compact
	chDIP := u.cardH()
	step := u.fpx(chDIP + cardGap)
	top0 := float64(in.Y) + u.fpx(listTop)
	vh := u.viewportH()
	first := maxi(0, int(math.Floor(u.Scroll/step))-1)
	since := time.Since(u.enterStart).Seconds()

	for i := first; i < len(u.View); i++ {
		cy := top0 + float64(i)*step - u.Scroll
		if cy > top0+vh+u.fpx(20) {
			break
		}
		if cy+u.fpx(chDIP) < top0-u.fpx(20) {
			continue
		}
		// 入场动画:前 12 张卡片依次上滑
		if u.Set.Anim && i < 12 && since < 1.2 {
			e := clamp01((since - float64(i)*0.03) / 0.28)
			e = 1 - (1-e)*(1-e)*(1-e)
			if e < 1 {
				u.entering = true
			}
			cy += (1 - e) * u.fpx(18)
		}
		it := u.View[i]
		cx := fx(16)
		cw := u.fpx(designW - 32)
		ch := u.fpx(chDIP)
		card := Rect{int(cx), int(cy), int(cw), int(ch)}
		hovering := isItemAct(u.Hover.Act) && u.Hover.Arg == i
		selected := i == u.Sel

		a := p.CardA
		if hovering || selected {
			a = p.CardHoverA
		}
		rad := u.fpx(15)
		c.FillRRect(cx, cy, cw, ch, rad, p.CardCol, a)
		c.Line(cx+rad, cy+u.fpx(1.2), cx+cw-rad, cy+u.fpx(1.2), u.fpx(1), Color{255, 255, 255}, p.Gloss*0.5)
		if selected {
			c.FillRRect(cx, cy, cw, ch, rad, p.Accent, 0.12)
			c.StrokeRRect(cx, cy, cw, ch, rad, u.fpx(1.3), p.Accent, 0.7)
			c.FillRRect(cx+u.fpx(4), cy+ch*0.24, u.fpx(3), ch*0.52, u.fpx(1.5), p.Accent, 1)
		} else {
			c.StrokeRRect(cx, cy, cw, ch, rad, u.fpx(1), p.BorderCol, p.BorderA)
		}
		u.addHit(card, ActItem, i)

		// 类型图标 / 缩略图 / 颜色块
		tsz := 38.0
		tx0, ty0 := 12.0, 12.0
		if compact {
			tsz, tx0, ty0 = 32, 11, 11
		}
		tile := Rect{int(cx + u.fpx(tx0)), int(cy + u.fpx(ty0)), u.px(tsz), u.px(tsz)}
		kc := kindColor[it.Kind]
		switch it.Kind {
		case KImage:
			if th := u.Store.Thumb(it); th != nil {
				c.DrawCover(th, float64(tile.X), float64(tile.Y), float64(tile.W), u.fpx(11))
				c.StrokeRRect(float64(tile.X), float64(tile.Y), float64(tile.W), float64(tile.H), u.fpx(11), u.fpx(1), p.BorderCol, p.BorderA*1.5)
			} else {
				c.FillRRect(float64(tile.X), float64(tile.Y), float64(tile.W), float64(tile.H), u.fpx(11), kc, 0.22)
				t.Draw(T(kindKey[it.Kind]), tile, ts(14, true), kc, AlignCenter, 1)
			}
		case KColor:
			c.FillRRect(float64(tile.X), float64(tile.Y), float64(tile.W), float64(tile.H), u.fpx(11), it.color, 1)
			c.StrokeRRect(float64(tile.X), float64(tile.Y), float64(tile.W), float64(tile.H), u.fpx(11), u.fpx(1), p.BorderCol, p.BorderA*2)
		default:
			c.FillRRect(float64(tile.X), float64(tile.Y), float64(tile.W), float64(tile.H), u.fpx(11), kc, 0.20)
			c.VGradient(Rect{tile.X, tile.Y + u.px(2), tile.W, tile.H / 2}, Color{255, 255, 255}, 0.10, 0)
			t.Draw(T(kindKey[it.Kind]), tile, ts(15, true), kc, AlignCenter, 1)
		}

		// 文本区
		tx := int(cx + u.fpx(tx0+tsz+12))
		rightPad := u.px(14)
		if hovering && compact {
			rightPad = u.px(14 + 3*24 + 2*5 + 6)
		}
		tw := card.X + card.W - rightPad - tx
		var main string
		switch it.Kind {
		case KImage:
			main = T("img.label") + "  " + itoa(it.W) + "×" + itoa(it.H)
		case KSecret:
			main = maskSecret(it.Text)
		default:
			main = previewText(it.Text)
		}
		meta := it.Meta(now)
		metaX := tx
		if g := u.Store.GroupByID(it.Group); g != nil && it.Group != 0 {
			a := accents[clampi(g.Color, 0, len(accents)-1)]
			gc := a[1]
			if u.Dark {
				gc = a[0]
			}
			metaY := cy + u.fpx(50+8)
			if compact {
				metaY = cy + u.fpx(29+8)
			}
			c.FillCircle(float64(tx)+u.fpx(4), metaY, u.fpx(3.5), gc, 1)
			metaX += u.px(12)
			tw -= u.px(12)
			meta = g.Name + " · " + meta
		}
		if compact {
			t.Draw(main, Rect{tx, int(cy + u.fpx(6)), tw, u.px(22)}, ts(13, false), p.Text, AlignLeft, 1)
			t.Draw(meta, Rect{metaX, int(cy + u.fpx(29)), tw, u.px(16)}, ts(10.5, false), p.Sub, AlignLeft, 1)
		} else {
			lines := 2
			if it.Kind == KImage {
				lines = 1
			}
			t.Draw(main, Rect{tx, int(cy + u.fpx(10)), tw, u.px(35)}, ts(13, false), p.Text, AlignLeft, lines)
			mw := tw - u.px(8)
			if hovering {
				mw = tw - u.px(3*24+2*5+10)
			}
			t.Draw(meta, Rect{metaX, int(cy + u.fpx(50)), mw, u.px(16)}, ts(11, false), p.Sub, AlignLeft, 1)
		}

		if it.Pinned && !hovering {
			u.drawPin(c, float64(card.X+card.W)-u.fpx(22), cy+u.fpx(17), p.Gold, 1)
		}
		if !hovering && i < 9 && !compact {
			t.Draw("Alt+"+string(rune('1'+i)), Rect{card.X + card.W - u.px(64), int(cy + u.fpx(50)), u.px(52), u.px(16)}, ts(10, false), p.Sub, AlignRight, 1)
		}

		// 悬停操作:复制 / 置顶 / 删除
		if hovering {
			bs := u.fpx(24)
			by := cy + ch - bs - u.fpx(6)
			if compact {
				by = cy + (ch-bs)/2
			}
			delX := cx + cw - u.fpx(12) - bs
			pinX := delX - u.fpx(5) - bs
			cpyX := pinX - u.fpx(5) - bs
			for k, bx := range []float64{cpyX, pinX, delX} {
				act := []int{ActCopy, ActPin, ActDel}[k]
				bhov := u.Hover.Act == act && u.Hover.Arg == i
				ba := p.ChipA * 1.6
				if bhov {
					ba = p.ChipA * 3.4
				}
				c.FillCircle(bx+bs/2, by+bs/2, bs/2, p.ChipCol, ba)
				br := Rect{int(bx), int(by), int(bs), int(bs)}
				switch k {
				case 0:
					cc := p.Sub
					if bhov {
						cc = p.Accent
					}
					u.drawCopyIcon(c, bx+bs/2, by+bs/2, cc)
				case 1:
					gc, ga := p.Sub, 0.9
					if it.Pinned {
						gc, ga = p.Gold, 1
					} else if bhov {
						gc = p.Accent
					}
					u.drawPin(c, bx+bs/2, by+bs/2, gc, ga)
				default:
					xc := p.Sub
					if bhov {
						xc = Color{255, 90, 90}
					}
					u.drawX(c, bx+bs/2, by+bs/2, u.fpx(3.8), xc)
				}
				u.addHit(br, act, i)
			}
		}
	}
}

// ---------- 设置页 ----------

const (
	rSection = iota
	rSwitch
	rPill
	rSwatch
	rHotkey
	rButton
)

type srow struct {
	kind, id    int
	label, desc string
	val         string
	on, danger  bool
}

func (r srow) height() float64 {
	if r.kind == rSection {
		return 36
	}
	return 62
}

func (u *UI) rows() []srow {
	s := u.Set
	langDisp := T("lang.auto") + " · " + langNames[curLang]
	for i, c := range langCodes {
		if c == s.Lang {
			langDisp = langNames[i]
		}
	}
	pos := []string{T("pos.cursor"), T("pos.center"), T("pos.last")}[s.PosMode]
	lvl := []string{T("lvl.low"), T("lvl.mid"), T("lvl.high")}[s.Blur]
	th := []string{T("th.auto"), T("th.light"), T("th.dark")}[s.Theme]
	clearDesc := T("r.clear.d")
	if time.Now().Before(u.confirm) {
		clearDesc = T("r.clear.c")
	}
	hk := func(h HotkeySpec, id int) string {
		if u.Recording == id {
			return T("hk.rec")
		}
		return h.String()
	}
	return []srow{
		{kind: rSection, label: T("s.general")},
		{rPill, SetLang, T("r.lang"), T("r.lang.d"), langDisp, false, false},
		{rHotkey, SetHotkey, T("r.hotkey"), T("r.hotkey.d"), hk(s.Hotkey, SetHotkey), false, false},
		{rHotkey, SetPauseKey, T("r.pausekey"), T("r.pausekey.d"), hk(s.PauseKey, SetPauseKey), false, false},
		{rPill, SetPos, T("r.pos"), T("r.pos.d"), pos, false, false},
		{rSwitch, SetAutoPaste, T("r.autopaste"), T("r.autopaste.d"), "", s.AutoPaste, false},
		{rSwitch, SetAutoStart, T("r.autostart"), T("r.autostart.d"), "", s.AutoStart, false},
		{rSwitch, SetKeepOpen, T("r.keepopen"), T("r.keepopen.d"), "", s.KeepOpen, false},
		{rSwitch, SetBall, T("r.ball"), T("r.ball.d"), "", s.Ball, false},

		{kind: rSection, label: T("s.look")},
		{rSwitch, SetGlass, T("r.glass"), T("r.glass.d"), "", s.Glass, false},
		{rPill, SetBlur, T("r.blur"), T("r.blur.d"), lvl, false, false},
		{rPill, SetOpacity, T("r.opacity"), T("r.opacity.d"), T("op." + itoa(clampi(s.Opacity, 0, 2))), false, false},
		{rPill, SetTheme, T("r.theme"), T("r.theme.d"), th, false, false},
		{rSwatch, SetAccent, T("r.accent"), T("r.accent.d"), "", false, false},
		{rPill, SetScale, T("r.scale"), T("r.scale.d"), scaleLbls[s.UIScale], false, false},
		{rSwitch, SetCompact, T("r.compact"), T("r.compact.d"), "", s.Compact, false},
		{rSwitch, SetAnim, T("r.anim"), T("r.anim.d"), "", s.Anim, false},

		{kind: rSection, label: T("s.privacy")},
		{rPill, SetSensitive, T("r.sensitive"), T("r.sensitive.d"), T([]string{"sens.skip", "sens.keep", "sens.off"}[clampi(s.Sensitive, 0, 2)]), false, false},
		{rSwitch, SetIgnorePwd, T("r.pwd"), T("r.pwd.d"), "", s.IgnorePwd, false},
		{rButton, SetIgnoreList, T("r.ignore") + " (" + itoa(len(s.IgnoreSources)) + ")", T("r.ignore.d"), T("b.reset"), false, false},
		{rSwitch, SetImages, T("r.images"), T("r.images.d"), "", s.RecordImages, false},
		{rPill, SetRetention, T("r.keep"), T("r.keep.d"), T("keep." + itoa(s.Retention)), false, false},
		{rPill, SetMaxItems, T("r.max"), T("r.max.d"), Tf("n.items", s.MaxItems), false, false},
		{rButton, SetOpenFolder, T("r.folder"), T("r.folder.d"), T("b.open"), false, false},
		{rButton, SetExport, T("r.export"), T("r.export.d"), T("b.export"), false, false},
		{rButton, SetClearAll, T("r.clear"), clearDesc, T("b.clear"), false, true},
	}
}

func (u *UI) drawSettings(c *Canvas, t TextRenderer, p Pal, in Rect, ts func(float64, bool) TextStyle) {
	rows := u.rows()
	y := float64(in.Y) + u.fpx(u.top()) - u.Scroll
	vTop := float64(in.Y) + u.fpx(u.top()) - u.fpx(10)
	vBot := vTop + u.viewportH() + u.fpx(20)
	cx, cw := float64(in.X)+u.fpx(16), u.fpx(designW-32)

	for _, r := range rows {
		hgt := u.fpx(r.height())
		if y > vBot {
			break
		}
		if y+hgt < vTop {
			y += hgt
			continue
		}
		if r.kind == rSection {
			t.Draw(r.label, Rect{int(cx + u.fpx(6)), int(y + u.fpx(8)), u.px(300), u.px(22)}, ts(12, true), p.Accent, AlignLeft, 1)
			y += hgt
			continue
		}
		ch := u.fpx(54)
		hov := u.Hover.Act == ActSet && u.Hover.Arg == r.id
		a := p.CardA
		if hov {
			a = p.CardHoverA
		}
		c.FillRRect(cx, y, cw, ch, u.fpx(15), p.CardCol, a)
		c.Line(cx+u.fpx(15), y+u.fpx(1.2), cx+cw-u.fpx(15), y+u.fpx(1.2), u.fpx(1), Color{255, 255, 255}, p.Gloss*0.5)
		c.StrokeRRect(cx, y, cw, ch, u.fpx(15), u.fpx(1), p.BorderCol, p.BorderA)
		col := p.Text
		if r.danger {
			col = Color{232, 70, 70}
		}
		right := cx + cw - u.fpx(16)
		textW := int(cw - u.fpx(32) - u.fpx(130))
		t.Draw(r.label, Rect{int(cx + u.fpx(16)), int(y + u.fpx(8)), textW, u.px(22)}, ts(14, false), col, AlignLeft, 1)
		dr := Rect{int(cx + u.fpx(16)), int(y + u.fpx(30)), textW + u.px(40), u.px(18)}
		t.Draw(r.desc, dr, u.fitStyle(t, r.desc, dr.W, 11, false), p.Sub, AlignLeft, 1)

		rowRect := Rect{int(cx), int(y), int(cw), int(ch)}
		u.addHit(rowRect, ActSet, r.id)

		switch r.kind {
		case rSwitch:
			sw, sh := u.fpx(44), u.fpx(24)
			sx, sy := right-sw, y+(ch-sh)/2
			if r.on {
				c.FillRRect(sx, sy, sw, sh, sh/2, p.Accent, 0.95)
			} else {
				c.FillRRect(sx, sy, sw, sh, sh/2, p.ChipCol, p.ChipA*2.2)
			}
			kx := sx + u.fpx(3)
			if r.on {
				kx = sx + sw - sh + u.fpx(3)
			}
			kr := (sh - u.fpx(6)) / 2
			c.FillCircle(kx+kr, sy+sh/2+0.5, kr+0.5, Color{0, 0, 0}, 0.12)
			c.FillCircle(kx+kr, sy+sh/2, kr, Color{255, 255, 255}, 1)
		case rSwatch:
			d := u.fpx(22)
			gap := u.fpx(9)
			x := right - float64(len(accents))*d - float64(len(accents)-1)*gap
			for i := range accents {
				ac := accents[i][0]
				if !u.Dark {
					ac = accents[i][1]
				}
				c.FillCircle(x+d/2, y+ch/2, d/2, ac, 1)
				if u.Set.Accent == i {
					c.StrokeCircle(x+d/2, y+ch/2, d/2+u.fpx(3), u.fpx(1.8), p.Text, 0.9)
				}
				u.addHit(Rect{int(x - gap/2), int(y), int(d + gap), int(ch)}, ActSetVal, SetAccent<<8|i)
				x += d + gap
			}
		default: // 药丸:选项 / 热键 / 按钮
			label := r.val
			w := float64(t.Measure(label, ts(12, true))) + u.fpx(28)
			pr := Rect{int(right - w), int(y + (ch-u.fpx(28))/2), int(w), u.px(28)}
			fill, tc, ca := p.ChipCol, p.Accent, p.ChipA*1.4
			if hov {
				ca *= 1.8
			}
			if r.kind == rHotkey && u.Recording == r.id {
				fill, tc, ca = p.Accent, Color{255, 255, 255}, 0.95
			}
			if r.danger {
				fill, tc = Color{232, 70, 70}, Color{232, 70, 70}
				ca = 0.12
				if hov {
					ca = 0.22
				}
			}
			c.FillRRect(float64(pr.X), float64(pr.Y), float64(pr.W), float64(pr.H), float64(pr.H)/2, fill, ca)
			t.Draw(label, pr, ts(12, true), tc, AlignCenter, 1)
		}
		y += hgt
	}
	t.Draw(T("about"), Rect{int(float64(in.X) + u.fpx(10)), int(y + u.fpx(2)), u.px(designW - 20), u.px(24)}, ts(11, false), p.Sub, AlignCenter, 1)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

var _ = strings.TrimSpace
