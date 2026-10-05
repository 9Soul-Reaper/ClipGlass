//go:build windows

package main

import (
	"runtime"
	"unsafe"
)

var pDeleteDC = gdi32.NewProc("DeleteDC")

// gdiText 使用 GDI 在 32 位 DIB 上绘制文字(随后由宿主统一恢复 alpha)。
type gdiText struct {
	dc    uintptr
	fonts map[TextStyle]uintptr
}

var gSize2 struct{ CX, CY int32 }
var gTextRect winRect

func newGDIText(dc uintptr) *gdiText {
	return &gdiText{dc: dc, fonts: map[TextStyle]uintptr{}}
}

func (g *gdiText) font(st TextStyle) uintptr {
	if f, ok := g.fonts[st]; ok {
		return f
	}
	weight := uintptr(400)
	if st.Bold {
		weight = 700
	}
	face := u16(CurFace())
	f := call(pCreateFontW, uintptr(int32(-st.Size)), 0, 0, 0, weight, 0, 0, 0,
		1 /*DEFAULT_CHARSET*/, 0, 0, 5 /*CLEARTYPE_QUALITY*/, 0, uintptr(unsafe.Pointer(face)))
	runtime.KeepAlive(face)
	g.fonts[st] = f
	return f
}

// ResetFonts 在界面语言(字体)切换后清空字体缓存。
func (g *gdiText) ResetFonts() {
	for _, f := range g.fonts {
		call(pDeleteObject, f)
	}
	g.fonts = map[TextStyle]uintptr{}
}

func (g *gdiText) SetClip(r Rect) {
	call(pSelectClipRgn, g.dc, 0)
	call(pIntersectClipRect, g.dc, uintptr(r.X), uintptr(r.Y), uintptr(r.X+r.W), uintptr(r.Y+r.H))
}

func (g *gdiText) Measure(s string, st TextStyle) int {
	if s == "" {
		return 0
	}
	old := call(pSelectObject, g.dc, g.font(st))
	us, _ := utf16Of(s)
	call(pGetTextExtentPoint, g.dc, uintptr(unsafe.Pointer(&us[0])), uintptr(len(us)-1), uintptr(unsafe.Pointer(&gSize2)))
	call(pSelectObject, g.dc, old)
	runtime.KeepAlive(us)
	return int(gSize2.CX)
}

func (g *gdiText) Draw(s string, r Rect, st TextStyle, col Color, align, lines int) {
	if s == "" || r.W <= 0 || r.H <= 0 {
		return
	}
	old := call(pSelectObject, g.dc, g.font(st))
	call(pSetBkMode, g.dc, 1)
	call(pSetTextColor, g.dc, uintptr(col.R)|uintptr(col.G)<<8|uintptr(col.B)<<16)
	flags := uintptr(0x800 | 0x8000) // DT_NOPREFIX | DT_END_ELLIPSIS
	switch align {
	case AlignCenter:
		flags |= 0x1
	case AlignRight:
		flags |= 0x2
	}
	if lines > 1 {
		flags |= 0x10 // DT_WORDBREAK
	} else {
		flags |= 0x20 | 0x4 // DT_SINGLELINE | DT_VCENTER
	}
	gTextRect = winRect{int32(r.X), int32(r.Y), int32(r.X + r.W), int32(r.Y + r.H)}
	us, _ := utf16Of(s)
	call(pDrawTextW, g.dc, uintptr(unsafe.Pointer(&us[0])), ^uintptr(0), uintptr(unsafe.Pointer(&gTextRect)), flags)
	runtime.KeepAlive(us)
	call(pGdiFlush) // 保证与直接写像素的绘制顺序一致
	call(pSelectObject, g.dc, old)
}

// utf16Of 转为以 0 结尾的 UTF-16(过滤内嵌 NUL)。
func utf16Of(s string) ([]uint16, error) {
	out := make([]uint16, 0, len(s)+1)
	for _, r := range s {
		if r == 0 {
			continue
		}
		if r >= 0x10000 {
			r -= 0x10000
			out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
		} else {
			out = append(out, uint16(r))
		}
	}
	return append(out, 0), nil
}
