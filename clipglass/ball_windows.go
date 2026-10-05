//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	pSetCapture     = user32.NewProc("SetCapture")
	pReleaseCapture = user32.NewProc("ReleaseCapture")
)

const ballClass = "ClipGlassBallWnd"

type ballState struct {
	hwnd, dc, dib uintptr
	canvas        *Canvas
	size          int
	hover, down   bool
	moved         bool
	startPt       point
	startWin      point
}

var ball ballState

func (a *App) ballPx() int { return int(64 * a.dpiScale) }

// syncBall 按设置创建 / 显示 / 隐藏悬浮球。
func (a *App) syncBall() {
	if !a.set.Ball {
		if ball.hwnd != 0 {
			call(pShowWindow, ball.hwnd, 0)
		}
		return
	}
	if ball.hwnd == 0 {
		cls := wndClassEx{
			Size:      uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:   syscall.NewCallback(ballProc),
			Instance:  a.hinst,
			Cursor:    call(pLoadCursorW, 0, 32649), // IDC_HAND
			ClassName: u16(ballClass),
		}
		call(pRegisterClassExW, uintptr(unsafe.Pointer(&cls)))
		sz := a.ballPx()
		x, y := a.set.BallX, a.set.BallY
		if !a.set.BallSet {
			w := a.workAreaAt(0, 0)
			x = int(w.Right) - sz - 4
			y = int(w.Top) + int(w.Bottom-w.Top)*2/5
		}
		// WS_EX_LAYERED|TOPMOST|TOOLWINDOW|NOACTIVATE
		ball.hwnd = call(pCreateWindowExW, 0x80000|0x8|0x80|0x08000000, uintptr(unsafe.Pointer(u16(ballClass))), 0,
			0x80000000, uintptr(x), uintptr(y), uintptr(sz), uintptr(sz), 0, 0, a.hinst, 0)
		if ball.hwnd == 0 {
			return
		}
		ball.dc = call(pCreateCompatibleDC, 0)
		ball.size = sz
		gBIH = bitmapInfoHeader{Size: 40, Width: int32(sz), Height: -int32(sz), Planes: 1, Bits: 32}
		var bits uintptr
		ball.dib = call(pCreateDIBSection, ball.dc, uintptr(unsafe.Pointer(&gBIH)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
		call(pSelectObject, ball.dc, ball.dib)
		ball.canvas = &Canvas{W: sz, H: sz, Pix: unsafe.Slice((*byte)(unsafe.Pointer(bits)), sz*sz*4), Clip: Rect{0, 0, sz, sz}}
	}
	a.updateBall()
	call(pShowWindow, ball.hwnd, 8)
}

func (a *App) updateBall() {
	if ball.hwnd == 0 || !a.set.Ball {
		return
	}
	ac := accents[clampi(a.set.Accent, 0, len(accents)-1)]
	dark := a.isDark()
	col := ac[1]
	if dark {
		col = ac[0]
	}
	r := renderBall(ball.size, dark, col, ball.hover)
	copy(ball.canvas.Pix, r.Pix)
	alpha := byte(205)
	if ball.hover || ball.down {
		alpha = 255
	}
	gSizeW = sizeT{int32(ball.size), int32(ball.size)}
	gSrcPt = point{}
	gBlend = blendFunction{Op: 0, Flags: 0, Alpha: alpha, Format: 1}
	call(pUpdateLayeredWindow, ball.hwnd, 0, 0, uintptr(unsafe.Pointer(&gSizeW)), ball.dc,
		uintptr(unsafe.Pointer(&gSrcPt)), 0, uintptr(unsafe.Pointer(&gBlend)), 2)
}

func ballProc(hwnd, m, wp, lp uintptr) (ret uintptr) {
	a := app
	defer func() {
		if r := recover(); r != nil {
			logErr(r)
			ret = call(pDefWindowProcW, hwnd, m, wp, lp)
		}
	}()
	switch uint32(m) {
	case wmMouseMove:
		if !ball.hover {
			ball.hover = true
			gTME = trackMouseEvent{Size: uint32(unsafe.Sizeof(gTME)), Flags: 2, HWnd: hwnd}
			call(pTrackMouseEvent, uintptr(unsafe.Pointer(&gTME)))
			a.updateBall()
		}
		if ball.down {
			call(pGetCursorPos, uintptr(unsafe.Pointer(&gPt)))
			dx, dy := int(gPt.X-ball.startPt.X), int(gPt.Y-ball.startPt.Y)
			if !ball.moved && dx*dx+dy*dy > 25 {
				ball.moved = true
			}
			if ball.moved {
				call(pSetWindowPos, hwnd, ^uintptr(0), uintptr(int(ball.startWin.X)+dx), uintptr(int(ball.startWin.Y)+dy), 0, 0, 0x1|0x10)
			}
		}
		return 0
	case wmMouseLeave:
		ball.hover = false
		a.updateBall()
		return 0
	case wmLButtonDown:
		call(pGetCursorPos, uintptr(unsafe.Pointer(&ball.startPt)))
		call(pGetWindowRect, hwnd, uintptr(unsafe.Pointer(&gRect)))
		ball.startWin = point{gRect.Left, gRect.Top}
		ball.down, ball.moved = true, false
		call(pSetCapture, hwnd)
		a.updateBall()
		return 0
	case wmLButtonUp:
		wasDown := ball.down
		ball.down = false
		call(pReleaseCapture)
		if wasDown && ball.moved {
			a.snapBall()
		} else if wasDown {
			if a.visible {
				a.hide()
			} else {
				a.fromBall = true
				a.show()
			}
		}
		a.updateBall()
		return 0
	case wmRButtonUp:
		a.trayMenu()
		return 0
	}
	return call(pDefWindowProcW, hwnd, m, wp, lp)
}

// snapBall 拖动结束后吸附到最近的左右边缘并记住位置。
func (a *App) snapBall() {
	call(pGetWindowRect, ball.hwnd, uintptr(unsafe.Pointer(&gRect)))
	cx, cy := int(gRect.Left)+ball.size/2, int(gRect.Top)+ball.size/2
	w := a.workAreaAt(cx, cy)
	x, y := int(gRect.Left), int(gRect.Top)
	if cx < int(w.Left+w.Right)/2 {
		x = int(w.Left) + 4
	} else {
		x = int(w.Right) - ball.size - 4
	}
	y = clampi(y, int(w.Top)+4, int(w.Bottom)-ball.size-4)
	call(pSetWindowPos, ball.hwnd, ^uintptr(0), uintptr(x), uintptr(y), 0, 0, 0x1|0x10)
	a.set.BallX, a.set.BallY, a.set.BallSet = x, y, true
	a.set.Save(a.dir)
}

// ballRect 返回悬浮球的屏幕矩形。
func (a *App) ballRect() (winRect, bool) {
	if ball.hwnd == 0 || !a.set.Ball {
		return winRect{}, false
	}
	call(pGetWindowRect, ball.hwnd, uintptr(unsafe.Pointer(&gRect)))
	return gRect, true
}
