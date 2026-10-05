//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// 窗口消息
const (
	wmDestroy         = 0x0002
	wmActivate        = 0x0006
	wmClose           = 0x0010
	wmEndSession      = 0x0016
	wmEraseBkgnd      = 0x0014
	wmNCHitTest       = 0x0084
	wmKeyDown         = 0x0100
	wmChar            = 0x0102
	wmSysKeyDown      = 0x0104
	wmCommand         = 0x0111
	wmTimer           = 0x0113
	wmMouseMove       = 0x0200
	wmLButtonDown     = 0x0201
	wmLButtonUp       = 0x0202
	wmLButtonDblClk   = 0x0203
	wmRButtonUp       = 0x0205
	wmMouseWheel      = 0x020A
	wmMouseLeave      = 0x02A3
	wmHotkey          = 0x0312
	wmClipboardUpdate = 0x031D
	wmExitSizeMove    = 0x0232
	wmApp             = 0x8000
	wmShowFromOther   = wmApp + 1
	wmTray            = wmApp + 2
)

const (
	timerSave  = 1
	timerAnim  = 2
	timerCaret = 3

	menuShow   = 1001
	menuPause  = 1002
	menuAuto   = 1003
	menuFolder = 1004
	menuClear  = 1005
	menuExit   = 1006

	mPaste   = 2001
	mCopy    = 2002
	mPlain   = 2003
	mUpper   = 2004
	mLower   = 2005
	mOpen    = 2006
	mReveal  = 2007
	mMail    = 2008
	mPin     = 2009
	mIgnore  = 2010
	mDelete  = 2011
	mNewGrp  = 2012
	mMove    = 2013
	mNoGrp   = 2020
	mGrpBase = 2100
	gmRename = 2201
	gmColor  = 2202
	gmDelete = 2203
	langBase = 3000

	className = "ClipGlassWnd"
	appVer    = "1.4"
	appTitle  = "ClipGlass " + appVer
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
)

// 传给系统调用的结构体统一放在包级变量里:Go 的栈可能移动,而包级变量地址固定。
var (
	gMsg      msg
	gPt       point
	gRect     winRect
	gMI       monitorInfo
	gBIH      bitmapInfoHeader
	gBlend    blendFunction
	gSizeW    sizeT
	gSrcPt    point
	gTME      trackMouseEvent
	gNID      notifyIconData
	gII       iconInfo
	gCF       compositionForm
	gCand     candidateForm
	gBits     uintptr
	gPid      uint32
	gRegKey   uintptr
	gRegU32   uint32
	gRegTyp   uint32
	gRegSz    uint32
	gNameBuf  [1040]uint16
	gNameSz   uint32
	gMaskBits []byte
)

type App struct {
	hwnd, hinst uintptr
	dir         string
	store       *Store
	set         *Settings
	ui          *UI
	memDC       uintptr
	dib         uintptr
	canvas      *Canvas
	base        *Canvas
	mask        []byte
	text        *gdiText
	snap        *Canvas
	prevFg      uintptr
	visible     bool
	alpha       byte
	fading      bool
	fadeStart   time.Time
	shownAt     time.Time
	hiddenAt    time.Time
	animOn      bool
	tracking    bool
	menuOpen    bool
	fromBall    bool
	pressed     Hit
	taskbarMsg  uint32
	hkOK        [3]bool
	icon        uintptr
	highSur     rune
	saveTicks   int
	dpiScale    float64
	sysLang     string
	firstRun    bool
}

var app = &App{}

func main() {
	runtime.LockOSThread()
	hidden := false
	for _, a := range os.Args[1:] {
		if a == "--tray" {
			hidden = true
		}
	}
	call(pSetProcessDPIAware)

	// 单实例:再次启动时唤起已有窗口
	_, _, e := pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(u16(`Local\ClipGlass_Single_v1`))))
	if e == syscall.Errno(183) {
		w := call(pFindWindowW, uintptr(unsafe.Pointer(u16(className))), 0)
		if w == 0 {
			return
		}
		if windowText(w) == appTitle {
			call(pPostMessageW, w, wmShowFromOther, 0, 0)
			return
		}
		// 后台还在运行旧版本:结束它并由新版本接管(否则会一直看到旧界面)
		gPid = 0
		call(pGetWindowThreadProcessId, w, uintptr(unsafe.Pointer(&gPid)))
		if gPid != 0 {
			if h := call(pOpenProcess, 0x100001, 0, uintptr(gPid)); h != 0 {
				call(pTerminateProcess, h, 0)
				call(pWaitForSingleObject, h, 3000)
				call(pCloseHandle, h)
			}
		}
	}

	a := app
	a.dir = dataDir()
	_, err := os.Stat(filepath.Join(a.dir, "settings.json"))
	a.firstRun = err != nil
	a.set = LoadSettings(a.dir)
	a.store = LoadStore(a.dir)
	a.store.Prune(a.set)
	if a.firstRun {
		a.set.Save(a.dir)
	}
	a.sysLang = langFromLangID(uint16(call(pGetUserDefaultUILanguage)))
	ApplyLang(a.set.Lang, a.sysLang)

	dpi := uintptr(96)
	if pGetDpiForSystem.Find() == nil {
		if d := call(pGetDpiForSystem); d != 0 {
			dpi = d
		}
	}
	a.dpiScale = float64(dpi) / 96
	a.ui = NewUI(a.dpiScale*scaleVals[clampi(a.set.UIScale, 0, len(scaleVals)-1)], a.store, a.set)
	a.ui.OnAction = a.doAction
	a.ui.OnContext = a.itemMenu
	a.ui.OnHide = a.hide
	a.ui.OnSetting = a.onSetting
	a.ui.OnPauseToggle = a.togglePause
	a.ui.OnHotkey = a.setHotkey
	a.ui.OnRecording = a.onRecording
	a.ui.OnChipMenu = a.groupMenu
	a.ui.OnLangMenu = a.langMenu

	a.hinst = call(pGetModuleHandleW, 0)
	cls := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  a.hinst,
		Cursor:    call(pLoadCursorW, 0, 32512),
		ClassName: u16(className),
	}
	call(pRegisterClassExW, uintptr(unsafe.Pointer(&cls)))
	W, H := a.ui.WinSize()
	// WS_EX_LAYERED | WS_EX_TOPMOST | WS_EX_TOOLWINDOW
	a.hwnd = call(pCreateWindowExW, 0x80000|0x8|0x80, uintptr(unsafe.Pointer(u16(className))),
		uintptr(unsafe.Pointer(u16(appTitle))), 0x80000000, /*WS_POPUP*/
		0, 0, uintptr(W), uintptr(H), 0, 0, a.hinst, 0)
	if a.hwnd == 0 {
		return
	}

	a.memDC = call(pCreateCompatibleDC, 0)
	a.initSurface()
	a.text = newGDIText(a.memDC)

	a.taskbarMsg = uint32(call(pRegisterWindowMessageW, uintptr(unsafe.Pointer(u16("TaskbarCreated")))))
	call(pAddClipboardFormatListen, a.hwnd)
	a.registerHotkeys()
	a.icon = makeIcon()
	a.trayAdd()
	if a.set.AutoStart {
		setAutoStart(true) // 刷新路径
	}
	call(pSetTimer, a.hwnd, timerSave, 5000, 0)

	a.syncBall()
	if !hidden && !a.set.Ball {
		a.show()
	}

	for {
		r := call(pGetMessageW, uintptr(unsafe.Pointer(&gMsg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		call(pTranslateMessage, uintptr(unsafe.Pointer(&gMsg)))
		call(pDispatchMessageW, uintptr(unsafe.Pointer(&gMsg)))
	}
	a.store.Save()
	a.set.Save(a.dir)
}

// initSurface 按当前界面尺寸(重新)创建离屏 32 位 DIB。
func (a *App) initSurface() {
	W, H := a.ui.WinSize()
	gBIH = bitmapInfoHeader{Size: 40, Width: int32(W), Height: -int32(H), Planes: 1, Bits: 32}
	bmp := call(pCreateDIBSection, a.memDC, uintptr(unsafe.Pointer(&gBIH)), 0, uintptr(unsafe.Pointer(&gBits)), 0, 0)
	old := call(pSelectObject, a.memDC, bmp)
	if a.dib != 0 {
		call(pDeleteObject, a.dib)
	} else {
		_ = old
	}
	a.dib = bmp
	a.canvas = &Canvas{W: W, H: H, Pix: unsafe.Slice((*byte)(unsafe.Pointer(gBits)), W*H*4), Clip: Rect{0, 0, W, H}}
}

func logErr(v interface{}) {
	f, err := os.OpenFile(filepath.Join(app.dir, "error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %v\n%s\n", time.Now().Format("2006-01-02 15:04:05"), v, debug.Stack())
}

func wndProc(hwnd, m, wp, lp uintptr) (ret uintptr) {
	defer func() {
		if r := recover(); r != nil {
			logErr(r)
			a := app
			a.menuOpen = false
			ret = call(pDefWindowProcW, hwnd, m, wp, lp)
		}
	}()
	return app.handle(hwnd, uint32(m), wp, lp)
}

func keyDown(vk uintptr) bool { return call(pGetKeyState, vk)&0x8000 != 0 }

func (a *App) handle(hwnd uintptr, m uint32, wp, lp uintptr) uintptr {
	switch m {
	case wmEraseBkgnd:
		return 1
	case wmClose:
		a.hide()
		return 0
	case wmDestroy:
		a.trayDelete()
		call(pRemoveClipboardListener, hwnd)
		call(pUnregisterHotKey, hwnd, 1)
		call(pUnregisterHotKey, hwnd, 2)
		call(pPostQuitMessage, 0)
		return 0
	case wmEndSession:
		a.store.Save()
		a.set.Save(a.dir)
		return 0
	case wmClipboardUpdate:
		a.onClipboard()
		return 0
	case wmHotkey:
		if wp == 2 {
			a.togglePause()
		} else {
			a.toggle()
		}
		return 0
	case wmShowFromOther:
		if !a.visible {
			a.show()
		}
		return 0
	case wmTray:
		switch uint32(lp) {
		case wmLButtonUp, wmLButtonDblClk:
			// 点击托盘会先让窗口失焦而自动收起,此时不要再弹出来
			if a.visible || time.Since(a.hiddenAt) > 350*time.Millisecond {
				a.toggle()
			}
		case wmRButtonUp:
			a.trayMenu()
		}
		return 0
	case wmCommand:
		a.onMenu(int(wp & 0xFFFF))
		return 0
	case wmActivate:
		if wp&0xFFFF == 0 && a.visible && !a.set.KeepOpen && time.Since(a.shownAt) > 300*time.Millisecond {
			a.hide()
		}
		return 0
	case wmExitSizeMove:
		// 记住拖动后的位置
		call(pGetWindowRect, hwnd, uintptr(unsafe.Pointer(&gRect)))
		in := a.ui.InnerRect()
		a.set.PosX, a.set.PosY, a.set.PosSet = int(gRect.Left)+in.X, int(gRect.Top)+in.Y, true
		a.set.Save(a.dir)
		a.recapture()
		return 0
	case wmNCHitTest:
		x := int(int16(lp)) - a.winLeft()
		y := int(int16(lp>>16)) - a.winTop()
		in := a.ui.InnerRect()
		if !in.Contains(x, y) {
			return 1 // 阴影区域:当作客户区,点击时收起
		}
		if a.ui.HitAt(x, y).Act == ActDrag {
			return 2 // HTCAPTION:拖动窗口
		}
		return 1
	case wmMouseMove:
		if !a.tracking {
			gTME = trackMouseEvent{Size: uint32(unsafe.Sizeof(gTME)), Flags: 2, HWnd: hwnd}
			call(pTrackMouseEvent, uintptr(unsafe.Pointer(&gTME)))
			a.tracking = true
		}
		a.ui.MouseMove(int(int16(lp)), int(int16(lp>>16)))
		a.afterInput()
		return 0
	case wmMouseLeave:
		a.tracking = false
		a.ui.MouseLeave()
		a.afterInput()
		return 0
	case wmLButtonDown:
		if !a.ui.InnerRect().Contains(int(int16(lp)), int(int16(lp>>16))) {
			a.hide()
			return 0
		}
		a.pressed = a.ui.HitAt(int(int16(lp)), int(int16(lp>>16)))
		return 0
	case wmLButtonUp:
		h := a.ui.HitAt(int(int16(lp)), int(int16(lp>>16)))
		if h.Act != ActNone && h == a.pressed {
			a.ui.Click(h, false)
		}
		a.pressed = Hit{}
		a.afterInput()
		return 0
	case wmRButtonUp:
		if a.ui.Pop != nil {
			a.ui.ClosePopup()
			a.afterInput()
			return 0
		}
		h := a.ui.HitAt(int(int16(lp)), int(int16(lp>>16)))
		switch {
		case isItemAct(h.Act):
			a.ui.Click(Hit{Act: ActItem, Arg: h.Arg}, true)
		case h.Act == ActChip:
			a.ui.Click(h, true)
		}
		a.afterInput()
		return 0
	case wmMouseWheel:
		a.ui.Wheel(int(int16(wp >> 16)))
		a.afterInput()
		return 0
	case wmKeyDown, wmSysKeyDown:
		ctrl := keyDown(0x11)
		shift := keyDown(0x10)
		alt := m == wmSysKeyDown || keyDown(0x12)
		if a.ui.Recording >= 0 {
			var mods uint32
			if alt {
				mods |= ModAlt
			}
			if ctrl {
				mods |= ModCtrl
			}
			if shift {
				mods |= ModShift
			}
			if keyDown(0x5B) || keyDown(0x5C) {
				mods |= ModWin
			}
			a.ui.RecordKey(int(wp), mods)
			a.afterInput()
			return 0
		}
		if ctrl && wp == 'V' && !a.ui.SettingsOn {
			// 粘贴文字到搜索框
			if t, ok := readClipboardText(a.hwnd); ok {
				for _, r := range strings.TrimSpace(strings.SplitN(strings.ReplaceAll(t, "\r", ""), "\n", 2)[0]) {
					a.ui.Char(r)
				}
				a.afterInput()
			}
			return 0
		}
		handled := a.ui.Key(int(wp), ctrl, shift, alt)
		a.afterInput()
		if handled {
			return 0
		}
	case wmChar:
		r := rune(wp)
		if r >= 0xD800 && r < 0xDC00 {
			a.highSur = r
			return 0
		}
		if r >= 0xDC00 && r < 0xE000 && a.highSur != 0 {
			r = 0x10000 + (a.highSur-0xD800)<<10 + (r - 0xDC00)
		}
		a.highSur = 0
		a.ui.Char(r)
		a.afterInput()
		return 0
	case wmTimer:
		a.onTimer(int(wp))
		return 0
	}
	if m == a.taskbarMsg && m != 0 {
		a.trayAdd()
		return 0
	}
	return call(pDefWindowProcW, hwnd, uintptr(m), wp, lp)
}

// recapture 窗口被拖动后,重新截取新位置后方的桌面并重建毛玻璃底图。
func (a *App) recapture() {
	if !a.visible {
		return
	}
	in := a.ui.InnerRect()
	call(pGetWindowRect, a.hwnd, uintptr(unsafe.Pointer(&gRect)))
	x, y := int(gRect.Left)+in.X, int(gRect.Top)+in.Y
	a.shownAt = time.Now() // 隐藏会触发失活,避免被当成“点击别处”而收起
	call(pShowWindow, a.hwnd, 0)
	snap := a.capture(x, y, in.W, in.H)
	call(pShowWindow, a.hwnd, 8)
	if snap != nil {
		a.snap = snap
		a.base, a.mask = a.ui.BuildBase(a.snap)
	}
	a.ui.Dirty = true
	a.render()
	a.forceForeground()
}

func windowText(w uintptr) string {
	buf := make([]uint16, 128)
	n := call(pGetWindowTextW, w, uintptr(unsafe.Pointer(&buf[0])), 128)
	return syscall.UTF16ToString(buf[:n])
}

func (a *App) winLeft() int {
	call(pGetWindowRect, a.hwnd, uintptr(unsafe.Pointer(&gRect)))
	return int(gRect.Left)
}
func (a *App) winTop() int {
	call(pGetWindowRect, a.hwnd, uintptr(unsafe.Pointer(&gRect)))
	return int(gRect.Top)
}

// ---------- 显示 / 隐藏 / 渲染 ----------

func (a *App) isDark() bool {
	switch a.set.Theme {
	case 1:
		return false
	case 2:
		return true
	}
	v, ok := regReadDWORD(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "AppsUseLightTheme")
	return ok && v == 0
}

func (a *App) capture(x, y, w, h int) *Canvas {
	screen := call(pGetDC, 0)
	defer call(pReleaseDC, 0, screen)
	dc := call(pCreateCompatibleDC, screen)
	defer call(pDeleteDC, dc)
	gBIH = bitmapInfoHeader{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, Bits: 32}
	var bits uintptr
	bmp := call(pCreateDIBSection, dc, uintptr(unsafe.Pointer(&gBIH)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == 0 {
		return nil
	}
	defer call(pDeleteObject, bmp)
	old := call(pSelectObject, dc, bmp)
	defer call(pSelectObject, dc, old)
	// SRCCOPY | CAPTUREBLT
	if call(pBitBlt, dc, 0, 0, uintptr(w), uintptr(h), screen, uintptr(x), uintptr(y), 0x00CC0020|0x40000000) == 0 {
		return nil
	}
	call(pGdiFlush)
	c := NewCanvas(w, h)
	copy(c.Pix, unsafe.Slice((*byte)(unsafe.Pointer(bits)), w*h*4))
	return c
}

func (a *App) workAreaAt(x, y int) winRect {
	mon := call(pMonitorFromPoint, uintptr(uint32(int32(x)))|uintptr(uint32(int32(y)))<<32, 2)
	gMI = monitorInfo{Size: uint32(unsafe.Sizeof(gMI))}
	call(pGetMonitorInfoW, mon, uintptr(unsafe.Pointer(&gMI)))
	return gMI.Work
}

func (a *App) show() {
	if fg := call(pGetForegroundWindow); fg != a.hwnd {
		a.prevFg = fg
	}
	call(pGetCursorPos, uintptr(unsafe.Pointer(&gPt)))
	cx, cy := int(gPt.X), int(gPt.Y)
	W, H := a.ui.WinSize()
	in := a.ui.InnerRect()
	var x, y int
	work := a.workAreaAt(cx, cy)
	br, hasBall := a.ballRect()
	switch {
	case a.fromBall && hasBall:
		work = a.workAreaAt(int(br.Left), int(br.Top))
		if int(br.Left)+ball.size/2 < int(work.Left+work.Right)/2 {
			x = int(br.Right) + 8
		} else {
			x = int(br.Left) - 8 - in.W
		}
		y = int(br.Top) - in.H/3
	case a.set.PosMode == 2 && a.set.PosSet:
		x, y = a.set.PosX, a.set.PosY
		work = a.workAreaAt(x+in.W/2, y+in.H/2)
	case a.set.PosMode >= 1:
		x = int(work.Left) + (int(work.Right-work.Left)-in.W)/2
		y = int(work.Top) + (int(work.Bottom-work.Top)-in.H)/2
	default:
		x, y = cx+14, cy+14
	}
	const gap = 8
	if x+in.W > int(work.Right)-gap {
		x = int(work.Right) - gap - in.W
	}
	if y+in.H > int(work.Bottom)-gap {
		y = int(work.Bottom) - gap - in.H
	}
	if x < int(work.Left)+gap {
		x = int(work.Left) + gap
	}
	if y < int(work.Top)+gap {
		y = int(work.Top) + gap
	}

	a.fromBall = false
	a.ui.Dark = a.isDark()
	a.ui.Reset()
	// 已显示时(如缩放变化后重新布局)先隐藏再取景,避免截到自己
	if a.visible {
		call(pShowWindow, a.hwnd, 0)
		a.visible = false
	}
	a.snap = a.capture(x, y, in.W, in.H)
	a.base, a.mask = a.ui.BuildBase(a.snap)
	a.alpha = 0
	a.fading = a.set.Anim
	if !a.fading {
		a.alpha = 255
	}
	a.fadeStart = time.Now()
	a.shownAt = time.Now()
	// 防闪烁:窗口仍隐藏时先移动到位,并用全透明渲染一帧覆盖旧位图,最后再显示(不抢焦点)
	call(pSetWindowPos, a.hwnd, ^uintptr(0), uintptr(x-in.X), uintptr(y-in.Y), uintptr(W), uintptr(H), 0x10) // HWND_TOPMOST, SWP_NOACTIVATE
	a.visible = true
	a.render()
	call(pShowWindow, a.hwnd, 8) // SW_SHOWNA
	a.forceForeground()

	if a.firstRun {
		a.firstRun = false
		a.ui.Toast(Tf("t.first", a.set.Hotkey.String()))
	} else if !a.hkOK[1] && a.set.Hotkey.IsSet() {
		a.ui.ToastFor(T("t.hkbusy"), 3*time.Second)
	}
	a.startAnim()
	call(pSetTimer, a.hwnd, timerCaret, 530, 0)
	a.render()
}

func (a *App) hide() {
	if !a.visible {
		return
	}
	a.visible = false
	a.hiddenAt = time.Now()
	call(pKillTimer, a.hwnd, timerCaret)
	call(pShowWindow, a.hwnd, 0)
	a.store.Save()
}

func (a *App) toggle() {
	if a.visible {
		a.hide()
	} else {
		a.show()
	}
}

func (a *App) forceForeground() {
	fg := call(pGetForegroundWindow)
	var fgThread uintptr
	if fg != 0 {
		fgThread = call(pGetWindowThreadProcessId, fg, 0)
	}
	cur := call(pGetCurrentThreadId)
	attached := fgThread != 0 && fgThread != cur
	if attached {
		call(pAttachThreadInput, cur, fgThread, 1)
	}
	call(pSetForegroundWindow, a.hwnd)
	call(pSetFocus, a.hwnd)
	if attached {
		call(pAttachThreadInput, cur, fgThread, 0)
	}
}

func (a *App) render() {
	if !a.visible || a.base == nil {
		return
	}
	for pass := 0; pass < 2; pass++ {
		copy(a.canvas.Pix, a.base.Pix)
		a.canvas.SetClip(Rect{0, 0, a.canvas.W, a.canvas.H})
		a.text.SetClip(a.canvas.Clip)
		a.ui.Draw(a.canvas, a.text)
		// GDI 不维护 alpha,统一恢复为遮罩
		pix := a.canvas.Pix
		for i, mv := range a.mask {
			pix[i*4+3] = mv
		}
		// 滚动/内容变化后,鼠标下的元素可能变了:重新计算悬停并重绘一次
		if a.ui.MX >= 0 {
			if h := a.ui.HitAt(a.ui.MX, a.ui.MY); h != a.ui.Hover {
				a.ui.Hover = h
				continue
			}
		}
		break
	}
	gSizeW = sizeT{int32(a.canvas.W), int32(a.canvas.H)}
	gSrcPt = point{}
	gBlend = blendFunction{Op: 0, Flags: 0, Alpha: a.alpha, Format: 1}
	call(pUpdateLayeredWindow, a.hwnd, 0, 0, uintptr(unsafe.Pointer(&gSizeW)), a.memDC,
		uintptr(unsafe.Pointer(&gSrcPt)), 0, uintptr(unsafe.Pointer(&gBlend)), 2)
	a.ui.Dirty = false
}

func (a *App) afterInput() {
	if !a.visible {
		return
	}
	if a.ui.Dirty {
		a.render()
		a.updateIME()
	}
	a.startAnim()
}

func (a *App) startAnim() {
	if a.animOn {
		return
	}
	if a.fading || a.ui.Scroll != a.ui.ScrollTo || a.ui.HasToast() || a.ui.Animating() {
		a.animOn = true
		call(pSetTimer, a.hwnd, timerAnim, 16, 0)
	}
}

func (a *App) onTimer(id int) {
	switch id {
	case timerSave:
		a.store.Save()
		a.saveTicks++
		n := len(a.store.Items)
		if a.saveTicks%12 == 0 {
			a.store.Prune(a.set)
		} else {
			a.store.Prune(a.set) // 敏感内容 10 分钟到期清除
		}
		if n != len(a.store.Items) && a.visible {
			a.ui.Rebuild()
			a.render()
		}
	case timerAnim:
		busy := a.ui.Tick()
		if a.fading {
			t := float64(time.Since(a.fadeStart)) / float64(150*time.Millisecond)
			if t >= 1 {
				t = 1
				a.fading = false
			}
			e := 1 - (1-t)*(1-t)*(1-t)
			a.alpha = byte(e * 255)
			a.ui.Dirty = true
			busy = busy || a.fading
		}
		if a.visible && a.ui.Dirty {
			a.render()
		}
		if !busy && !a.fading {
			a.animOn = false
			call(pKillTimer, a.hwnd, timerAnim)
		}
	case timerCaret:
		if !a.visible {
			return
		}
		fg := call(pGetForegroundWindow)
		if fg != a.hwnd && fg != 0 && fg != ball.hwnd {
			a.prevFg = fg // 钉住时记住最近使用的应用,粘贴目标才不会错
		}
		// 兜底:没能拿到焦点(或焦点被抢走)时自动收起(钉住时不收)
		if !a.set.KeepOpen && time.Since(a.shownAt) > 800*time.Millisecond && fg != a.hwnd {
			a.hide()
			return
		}
		a.ui.CaretOn = !a.ui.CaretOn
		a.ui.Dirty = true
		a.render()
	}
}

func (a *App) updateIME() {
	himc := call(pImmGetContext, a.hwnd)
	if himc == 0 {
		return
	}
	x, y := a.ui.CaretPos(a.text)
	gCF = compositionForm{Style: 2, CurPos: point{int32(x), int32(y)}}
	call(pImmSetCompositionWnd, himc, uintptr(unsafe.Pointer(&gCF)))
	gCand = candidateForm{Index: 0, Style: 0x40, CurPos: point{int32(x), int32(y + a.ui.px(26))}}
	call(pImmSetCandidateWindow, himc, uintptr(unsafe.Pointer(&gCand)))
	call(pImmReleaseContext, a.hwnd, himc)
}

// ---------- 设置变更 / 暂停 / 热键 ----------

func (a *App) onSetting(id int) {
	switch id {
	case SetGlass, SetBlur, SetTheme, SetAccent, SetOpacity:
		a.ui.Dark = a.isDark()
		a.base, a.mask = a.ui.BuildBase(a.snap)
		a.updateBall()
	case SetBall:
		a.syncBall()
	case SetAutoStart:
		setAutoStart(a.set.AutoStart)
	case SetScale:
		a.applyScale()
	case SetRetention, SetMaxItems:
		a.store.Prune(a.set)
		a.ui.Rebuild()
	case SetOpenFolder:
		shellOpen(a.dir, "")
	case SetExport:
		name := filepath.Join(a.dir, "export-"+time.Now().Format("20060102-150405")+".txt")
		if os.WriteFile(name, []byte("\xef\xbb\xbf"+a.store.ExportText()), 0o644) == nil {
			shellOpen(name, "")
			a.ui.Toast(T("t.exported"))
		}
	}
	a.set.Save(a.dir)
	a.ui.Dirty = true
}

func (a *App) applyScale() {
	a.ui.S = a.dpiScale * scaleVals[clampi(a.set.UIScale, 0, len(scaleVals)-1)]
	a.initSurface()
	a.show()
	a.ui.SettingsOn = true
	a.ui.Dirty = true
	a.render()
}

func (a *App) togglePause() {
	a.set.Paused = !a.set.Paused
	a.set.Save(a.dir)
	if a.set.Paused {
		a.ui.Toast(T("t.paused"))
	} else {
		a.ui.Toast(T("t.resumed"))
	}
	a.ui.Dirty = true
}

func (a *App) registerHotkeys() {
	call(pUnregisterHotKey, a.hwnd, 1)
	call(pUnregisterHotKey, a.hwnd, 2)
	a.hkOK = [3]bool{}
	reg := func(id uintptr, h HotkeySpec) {
		if h.IsSet() {
			a.hkOK[id] = call(pRegisterHotKey, a.hwnd, id, uintptr(h.Mods)|0x4000 /*MOD_NOREPEAT*/, uintptr(h.VK)) != 0
		}
	}
	reg(1, a.set.Hotkey)
	reg(2, a.set.PauseKey)
}

// onRecording:录制快捷键期间暂时释放全局热键,否则按下原组合会直接触发。
func (a *App) onRecording(on bool) {
	if on {
		call(pUnregisterHotKey, a.hwnd, 1)
		call(pUnregisterHotKey, a.hwnd, 2)
	} else {
		a.registerHotkeys()
	}
}

func (a *App) setHotkey(id int, spec HotkeySpec) bool {
	target, hid := &a.set.Hotkey, 1
	if id == SetPauseKey {
		target, hid = &a.set.PauseKey, 2
	}
	old := *target
	*target = spec
	a.registerHotkeys()
	if spec.IsSet() && !a.hkOK[hid] {
		*target = old
		a.registerHotkeys()
		return false
	}
	a.set.Save(a.dir)
	a.trayDelete()
	a.trayAdd()
	return true
}

func shellOpen(target, params string) {
	var pp uintptr
	if params != "" {
		pp = uintptr(unsafe.Pointer(u16(params)))
	}
	call(pShellExecuteW, 0, uintptr(unsafe.Pointer(u16("open"))), uintptr(unsafe.Pointer(u16(target))), pp, 0, 1)
}

// ---------- 剪贴板 ----------

func openClipboard(hwnd uintptr) bool {
	for i := 0; i < 8; i++ {
		if call(pOpenClipboard, hwnd) != 0 {
			return true
		}
		time.Sleep(15 * time.Millisecond)
	}
	return false
}

func (a *App) onClipboard() {
	if a.set.Paused {
		return
	}
	// 自己写入剪贴板触发的更新,不再记录
	owner := call(pGetClipboardOwner)
	if owner == a.hwnd && owner != 0 {
		return
	}
	// 密码管理器等声明“不要进入历史”的内容
	if f := call(pRegisterClipboardFormatW, uintptr(unsafe.Pointer(u16("ExcludeClipboardContentFromMonitorProcessing")))); f != 0 && call(pIsClipboardFormatAvail, f) != 0 {
		return
	}
	exe := ownerExe(owner)
	low := strings.ToLower(exe)
	for _, s := range a.set.IgnoreSources {
		if s != "" && strings.Contains(low, s) {
			return
		}
	}
	if a.set.IgnorePwd {
		for _, s := range pwdManagers {
			if strings.Contains(low, s) {
				return
			}
		}
	}
	src := friendlySource(exe)
	switch {
	case call(pIsClipboardFormatAvail, 15) != 0: // CF_HDROP:文件
		paths := readClipboardFiles(a.hwnd)
		if len(paths) == 0 {
			return
		}
		it := a.store.AddFrom(strings.Join(paths, "\n"), src, exe, "")
		if it.Kind != KFile {
			it.Kind = KFile
			it.prepare()
		}
	case call(pIsClipboardFormatAvail, 13) != 0: // 文本
		text, ok := readClipboardText(a.hwnd)
		if !ok {
			return
		}
		text = strings.ReplaceAll(text, "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
		if strings.TrimSpace(text) == "" {
			return
		}
		reason := ""
		if a.set.Sensitive != 2 {
			reason = secretReason(text)
			if reason != "" && a.set.Sensitive == 0 {
				return // 跳过:完全不记录
			}
		}
		a.store.AddFrom(text, src, exe, reason)
	case a.set.RecordImages && call(pIsClipboardFormatAvail, 8) != 0: // CF_DIB
		b := readClipboardBytes(a.hwnd, 8, 96<<20)
		if b == nil {
			return
		}
		img := parseDIB(b)
		if img == nil {
			return
		}
		a.store.AddImage(img, src, exe)
	default:
		return
	}
	a.store.Prune(a.set)
	if a.visible {
		a.ui.Rebuild()
		a.render()
	}
}

func readClipboardBytes(hwnd uintptr, format uintptr, limit int) []byte {
	if !openClipboard(hwnd) {
		return nil
	}
	defer call(pCloseClipboard)
	h := call(pGetClipboardData, format)
	if h == 0 {
		return nil
	}
	n := int(call(pGlobalSize, h))
	if n <= 0 || n > limit {
		return nil
	}
	p := call(pGlobalLock, h)
	if p == 0 {
		return nil
	}
	defer call(pGlobalUnlock, h)
	out := make([]byte, n)
	copy(out, unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
	return out
}

func readClipboardFiles(hwnd uintptr) []string {
	if !openClipboard(hwnd) {
		return nil
	}
	defer call(pCloseClipboard)
	h := call(pGetClipboardData, 15)
	if h == 0 {
		return nil
	}
	n := int(call(pDragQueryFileW, h, 0xFFFFFFFF, 0, 0))
	buf := make([]uint16, 1040)
	var out []string
	for i := 0; i < n && i < 200; i++ {
		l := int(call(pDragQueryFileW, h, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), 1040))
		if l > 0 && l <= len(buf) {
			out = append(out, syscall.UTF16ToString(buf[:l]))
		}
	}
	return out
}

func readClipboardText(hwnd uintptr) (string, bool) {
	if !openClipboard(hwnd) {
		return "", false
	}
	defer call(pCloseClipboard)
	h := call(pGetClipboardData, 13)
	if h == 0 {
		return "", false
	}
	p := call(pGlobalLock, h)
	if p == 0 {
		return "", false
	}
	defer call(pGlobalUnlock, h)
	n := int(call(pGlobalSize, h)) / 2
	if n > 1<<20 {
		n = 1 << 20
	}
	buf := unsafe.Slice((*uint16)(unsafe.Pointer(p)), n)
	end := 0
	for end < len(buf) && buf[end] != 0 {
		end++
	}
	return syscall.UTF16ToString(buf[:end]), true
}

func setClipboardData(hwnd uintptr, format uintptr, data []byte) bool {
	if !openClipboard(hwnd) {
		return false
	}
	defer call(pCloseClipboard)
	call(pEmptyClipboard)
	h := call(pGlobalAlloc, 0x2 /*GMEM_MOVEABLE*/, uintptr(len(data)))
	if h == 0 {
		return false
	}
	p := call(pGlobalLock, h)
	if p == 0 {
		return false
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(p)), len(data)), data)
	call(pGlobalUnlock, h)
	return call(pSetClipboardData, format, h) != 0
}

func setClipboardText(hwnd uintptr, s string) bool {
	s = strings.ReplaceAll(s, "\n", "\r\n")
	u, _ := utf16Of(s)
	b := make([]byte, 0, len(u)*2)
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	return setClipboardData(hwnd, 13, b)
}

// dropFiles 生成 CF_HDROP 数据(DROPFILES + 宽字符路径列表)。
func dropFiles(paths []string) []byte {
	b := make([]byte, 20)
	b[0], b[16] = 20, 1
	for _, p := range paths {
		u, _ := utf16Of(p)
		for _, c := range u {
			b = append(b, byte(c), byte(c>>8))
		}
	}
	return append(b, 0, 0)
}

func ownerExe(owner uintptr) string {
	if owner == 0 {
		return ""
	}
	gPid = 0
	call(pGetWindowThreadProcessId, owner, uintptr(unsafe.Pointer(&gPid)))
	if gPid == 0 {
		return ""
	}
	h := call(pOpenProcess, 0x1000 /*PROCESS_QUERY_LIMITED_INFORMATION*/, 0, uintptr(gPid))
	if h == 0 {
		return ""
	}
	defer call(pCloseHandle, h)
	gNameSz = uint32(len(gNameBuf))
	if call(pQueryFullProcessImageNm, h, 0, uintptr(unsafe.Pointer(&gNameBuf[0])), uintptr(unsafe.Pointer(&gNameSz))) == 0 {
		return ""
	}
	return filepath.Base(syscall.UTF16ToString(gNameBuf[:gNameSz]))
}

// 对条目执行操作:写入剪贴板,并(可选)粘贴回原窗口
func (a *App) doAction(it *Item, op int) {
	text := it.Text
	ok := true
	switch it.Kind {
	case KImage:
		img := a.store.LoadImage(it)
		ok = img != nil && setClipboardData(a.hwnd, 8, buildDIB(img))
	case KFile:
		ok = setClipboardData(a.hwnd, 15, dropFiles(strings.Split(text, "\n")))
	default:
		switch op {
		case OpPlain:
			text = strings.Join(strings.Fields(text), " ")
		case OpUpper:
			text = strings.ToUpper(text)
		case OpLower:
			text = strings.ToLower(text)
		}
		ok = setClipboardText(a.hwnd, text)
	}
	if ok && text == it.Text {
		a.store.Bump(it)
	}
	if op == OpCopy {
		// 仅复制:窗口保持打开,给出提示
		if ok {
			a.ui.Toast(T("t.copied"))
		}
		a.ui.Rebuild()
		a.render()
		return
	}
	prev := a.prevFg
	if !a.set.KeepOpen {
		a.hide()
	}
	if !ok || !a.set.AutoPaste {
		return
	}
	if prev != 0 {
		call(pSetForegroundWindow, prev)
	}
	time.Sleep(90 * time.Millisecond)
	// 若 Alt / Shift 仍被按住(例如 Alt+数字快选),先抬起,避免变成别的快捷键
	for _, vk := range []uintptr{0x12, 0x10} {
		if keyDown(vk) {
			call(pKeybdEvent, vk, 0, 2, 0)
		}
	}
	call(pKeybdEvent, 0x11, 0, 0, 0)
	call(pKeybdEvent, 'V', 0, 0, 0)
	call(pKeybdEvent, 'V', 0, 2, 0)
	call(pKeybdEvent, 0x11, 0, 2, 0)
}

// ---------- 弹出菜单(界面内玻璃菜单,跟随主题) ----------

func appendMenu(m uintptr, flags, id uintptr, text string) {
	p := u16(text)
	call(pAppendMenuW, m, flags, id, uintptr(unsafe.Pointer(p)))
	runtime.KeepAlive(p)
}

func chk(b bool) uintptr {
	if b {
		return 0x8
	}
	return 0
}

const mfSep = 0x800

func (a *App) accentOf(i int) Color {
	ac := accents[clampi(i, 0, len(accents)-1)]
	if a.ui.Dark {
		return ac[0]
	}
	return ac[1]
}

func (a *App) closeUnlessPinned() {
	if !a.set.KeepOpen {
		a.hide()
	}
}

func (a *App) itemMenu(it *Item) {
	var items []PopItem
	add := func(id int, label string) { items = append(items, PopItem{ID: id, Label: label}) }
	sep := func() { items = append(items, PopItem{Sep: true}) }
	add(mPaste, T("m.paste"))
	add(mCopy, T("m.copy"))
	if it.Kind != KImage && it.Kind != KFile {
		add(mPlain, T("m.plain"))
		add(mUpper, T("m.upper"))
		add(mLower, T("m.lower"))
	}
	sep()
	switch {
	case it.Kind == KLink:
		add(mOpen, T("m.open"))
	case it.Kind == KPath || it.Kind == KFile:
		add(mReveal, T("m.reveal"))
	case it.Kind == KContact && strings.Contains(it.Text, "@"):
		add(mMail, T("m.mail"))
	}
	if it.Pinned {
		add(mPin, T("m.unpin"))
	} else {
		add(mPin, T("m.pin"))
	}
	add(mMove, T("m.move")+"  ›")
	if it.Exe != "" {
		add(mIgnore, T("m.ignore"))
	}
	sep()
	items = append(items, PopItem{ID: mDelete, Label: T("m.delete"), Danger: true})

	a.ui.OpenPopup(items, func(id int) {
		switch id {
		case mPaste:
			a.doAction(it, OpPaste)
		case mCopy:
			a.doAction(it, OpCopy)
		case mPlain:
			a.doAction(it, OpPlain)
		case mUpper:
			a.doAction(it, OpUpper)
		case mLower:
			a.doAction(it, OpLower)
		case mOpen:
			t := strings.TrimSpace(it.Text)
			if !strings.Contains(t, "://") {
				t = "http://" + t
			}
			shellOpen(t, "")
			a.closeUnlessPinned()
		case mReveal:
			p := strings.TrimSpace(strings.SplitN(it.Text, "\n", 2)[0])
			shellOpen("explorer.exe", `/select,"`+p+`"`)
			a.closeUnlessPinned()
		case mMail:
			shellOpen("mailto:"+strings.TrimSpace(it.Text), "")
			a.closeUnlessPinned()
		case mPin:
			a.store.TogglePin(it.ID)
			a.ui.Rebuild()
		case mMove:
			a.moveMenu(it)
		case mIgnore:
			a.set.IgnoreSources = append(a.set.IgnoreSources, exeKey(it.Exe))
			a.set.Save(a.dir)
			a.ui.Toast(Tf("t.ignored", friendlySource(it.Exe)))
		case mDelete:
			for i, v := range a.ui.View {
				if v == it {
					a.ui.deleteAt(i)
					break
				}
			}
		}
		a.ui.Dirty = true
		a.afterInput()
	})
}

func (a *App) moveMenu(it *Item) {
	items := []PopItem{{ID: mNoGrp, Label: T("m.nogroup"), Checked: it.Group == 0}}
	for _, g := range a.store.Groups {
		items = append(items, PopItem{ID: mGrpBase + g.ID, Label: g.Name, Checked: it.Group == g.ID, HasDot: true, Dot: a.accentOf(g.Color)})
	}
	items = append(items, PopItem{Sep: true}, PopItem{ID: mNewGrp, Label: T("m.newgroup")})
	a.ui.OpenPopup(items, func(id int) {
		switch {
		case id == mNoGrp:
			a.ui.AssignGroup(it, 0)
			a.ui.Toast(T("t.unmoved"))
		case id == mNewGrp:
			a.ui.AssignNew = it.ID
			a.ui.BeginNaming(-1)
		case id >= mGrpBase:
			if g := a.store.GroupByID(id - mGrpBase); g != nil {
				a.ui.AssignGroup(it, g.ID)
				a.ui.Toast(Tf("t.moved", g.Name))
			}
		}
		a.ui.Dirty = true
		a.afterInput()
	})
}

func (a *App) groupMenu(filter int) {
	gid := filter - FilterGroupBase
	g := a.store.GroupByID(gid)
	if g == nil {
		return
	}
	items := []PopItem{{ID: gmRename, Label: T("gm.rename")}, {ID: gmColor, Label: T("gm.color")}, {Sep: true},
		{ID: gmDelete, Label: T("gm.delete"), Danger: true}}
	a.ui.OpenPopup(items, func(id int) {
		switch id {
		case gmRename:
			a.ui.BeginNaming(gid)
		case gmColor:
			a.store.RecolorGroup(gid)
		case gmDelete:
			name := g.Name
			a.store.DeleteGroup(gid)
			if a.ui.Filter == filter {
				a.ui.SetFilter(FilterAll)
			}
			a.ui.Toast(Tf("t.gdeleted", name))
		}
		a.ui.Rebuild()
		a.afterInput()
	})
}

func (a *App) langMenu() {
	items := []PopItem{{ID: langBase, Label: T("lang.auto"), Checked: a.set.Lang == "auto" || a.set.Lang == ""}, {Sep: true}}
	for i, c := range langCodes {
		items = append(items, PopItem{ID: langBase + 1 + i, Label: langNames[i], Checked: a.set.Lang == c})
	}
	a.ui.OpenPopup(items, func(id int) {
		if id == langBase {
			a.set.Lang = "auto"
		} else if i := id - langBase - 1; i >= 0 && i < len(langCodes) {
			a.set.Lang = langCodes[i]
		}
		ApplyLang(a.set.Lang, a.sysLang)
		a.text.ResetFonts()
		a.trayDelete()
		a.trayAdd()
		a.set.Save(a.dir)
		a.ui.Dirty = true
		a.afterInput()
	})
}

// ---------- 托盘 ----------

func (a *App) trayAdd() {
	gNID = notifyIconData{}
	gNID.Size = uint32(unsafe.Sizeof(gNID))
	gNID.HWnd = a.hwnd
	gNID.ID = 1
	gNID.Flags = 1 | 2 | 4
	gNID.CallbackMessage = wmTray
	gNID.Icon = a.icon
	tip := T("tray.tip")
	if a.set.Hotkey.IsSet() {
		tip += " · " + a.set.Hotkey.String()
	}
	copy(gNID.Tip[:127], syscall.StringToUTF16(tip))
	call(pShellNotifyIconW, 0, uintptr(unsafe.Pointer(&gNID)))
}

func (a *App) trayDelete() {
	gNID = notifyIconData{}
	gNID.Size = uint32(unsafe.Sizeof(gNID))
	gNID.HWnd = a.hwnd
	gNID.ID = 1
	call(pShellNotifyIconW, 2, uintptr(unsafe.Pointer(&gNID)))
}

func (a *App) trayMenu() {
	m := call(pCreatePopupMenu)
	show := T("tray.show")
	if a.set.Hotkey.IsSet() {
		show += "\t" + a.set.Hotkey.String()
	}
	appendMenu(m, 0, menuShow, show)
	appendMenu(m, chk(a.set.Paused), menuPause, T("tray.pause"))
	appendMenu(m, chk(a.set.AutoStart), menuAuto, T("r.autostart"))
	appendMenu(m, mfSep, 0, "")
	appendMenu(m, 0, menuFolder, T("r.folder"))
	appendMenu(m, 0, menuClear, T("tray.clear"))
	appendMenu(m, mfSep, 0, "")
	appendMenu(m, 0, menuExit, T("tray.exit"))
	call(pGetCursorPos, uintptr(unsafe.Pointer(&gPt)))
	call(pSetForegroundWindow, a.hwnd)
	// TPM_RIGHTBUTTON | TPM_BOTTOMALIGN
	call(pTrackPopupMenu, m, 0x2|0x20, uintptr(gPt.X), uintptr(gPt.Y), 0, a.hwnd, 0)
	call(pPostMessageW, a.hwnd, 0, 0, 0)
	call(pDestroyMenu, m)
}

func (a *App) onMenu(id int) {
	switch id {
	case menuShow:
		if !a.visible {
			a.show()
		}
	case menuPause:
		a.togglePause()
	case menuAuto:
		a.set.AutoStart = !a.set.AutoStart
		setAutoStart(a.set.AutoStart)
		a.set.Save(a.dir)
	case menuFolder:
		shellOpen(a.dir, "")
	case menuClear:
		a.store.ClearUnpinned()
		a.store.Save()
	case menuExit:
		call(pDestroyWindow, a.hwnd)
	}
}

func makeIcon() uintptr {
	size := int(call(pGetSystemMetrics, 49)) // SM_CXSMICON
	if size < 16 {
		size = 16
	}
	size *= 2 // 绘制得大一些再交给系统缩放,边缘更平滑
	c := drawAppIcon(size)
	gBIH = bitmapInfoHeader{Size: 40, Width: int32(size), Height: -int32(size), Planes: 1, Bits: 32}
	screen := call(pGetDC, 0)
	var bits uintptr
	color := call(pCreateDIBSection, screen, uintptr(unsafe.Pointer(&gBIH)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	call(pReleaseDC, 0, screen)
	if color == 0 || bits == 0 {
		return 0
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(bits)), size*size*4)
	copy(dst, c.Pix)
	gMaskBits = make([]byte, ((size+15)/16*2)*size)
	mask := call(pCreateBitmap, uintptr(size), uintptr(size), 1, 1, uintptr(unsafe.Pointer(&gMaskBits[0])))
	gII = iconInfo{IsIcon: 1, MaskBmp: mask, ColorBmp: color}
	return call(pCreateIconIndirect, uintptr(unsafe.Pointer(&gII)))
}

// ---------- 注册表 ----------

func regReadDWORD(sub, name string) (uint32, bool) {
	if call(pRegOpenKeyExW, 0x80000001, uintptr(unsafe.Pointer(u16(sub))), 0, 0x20019, uintptr(unsafe.Pointer(&gRegKey))) != 0 {
		return 0, false
	}
	defer call(pRegCloseKey, gRegKey)
	gRegSz = 4
	if call(pRegQueryValueExW, gRegKey, uintptr(unsafe.Pointer(u16(name))), 0,
		uintptr(unsafe.Pointer(&gRegTyp)), uintptr(unsafe.Pointer(&gRegU32)), uintptr(unsafe.Pointer(&gRegSz))) != 0 {
		return 0, false
	}
	return gRegU32, true
}

func setAutoStart(on bool) {
	if call(pRegCreateKeyExW, 0x80000001, uintptr(unsafe.Pointer(u16(runKey))), 0, 0, 0, 0x20006, 0,
		uintptr(unsafe.Pointer(&gRegKey)), 0) != 0 {
		return
	}
	defer call(pRegCloseKey, gRegKey)
	name := u16("ClipGlass")
	if !on {
		call(pRegDeleteValueW, gRegKey, uintptr(unsafe.Pointer(name)))
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	v, _ := utf16Of(`"` + exe + `" --tray`)
	call(pRegSetValueExW, gRegKey, uintptr(unsafe.Pointer(name)), 0, 1, uintptr(unsafe.Pointer(&v[0])), uintptr(len(v)*2))
}
