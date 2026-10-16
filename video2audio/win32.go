//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	pRegisterClassExW     = user32.NewProc("RegisterClassExW")
	pCreateWindowExW      = user32.NewProc("CreateWindowExW")
	pDefWindowProcW       = user32.NewProc("DefWindowProcW")
	pGetMessageW          = user32.NewProc("GetMessageW")
	pTranslateMessage     = user32.NewProc("TranslateMessage")
	pDispatchMessageW     = user32.NewProc("DispatchMessageW")
	pPostQuitMessage      = user32.NewProc("PostQuitMessage")
	pShowWindow           = user32.NewProc("ShowWindow")
	pUpdateWindow         = user32.NewProc("UpdateWindow")
	pSendMessageW         = user32.NewProc("SendMessageW")
	pPostMessageW         = user32.NewProc("PostMessageW")
	pDestroyWindow        = user32.NewProc("DestroyWindow")
	pMessageBoxW          = user32.NewProc("MessageBoxW")
	pLoadCursorW          = user32.NewProc("LoadCursorW")
	pSetTimer             = user32.NewProc("SetTimer")
	pKillTimer            = user32.NewProc("KillTimer")
	pEnableWindow         = user32.NewProc("EnableWindow")
	pSetFocus             = user32.NewProc("SetFocus")
	pGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	pSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	pGetDpiForSystem      = user32.NewProc("GetDpiForSystem")
	pSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	pGetDC                = user32.NewProc("GetDC")
	pReleaseDC            = user32.NewProc("ReleaseDC")
	pIsWindowEnabled      = user32.NewProc("IsWindowEnabled")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	pGetStockObject   = gdi32.NewProc("GetStockObject")
	pDeleteObject     = gdi32.NewProc("DeleteObject")
	pCreateBitmap     = gdi32.NewProc("CreateBitmap")
	pCreateDIBSection = gdi32.NewProc("CreateDIBSection")

	pCreateIconIndirect = user32.NewProc("CreateIconIndirect")

	pDragAcceptFiles  = shell32.NewProc("DragAcceptFiles")
	pDragQueryFileW   = shell32.NewProc("DragQueryFileW")
	pDragFinish       = shell32.NewProc("DragFinish")
	pShellExecuteW    = shell32.NewProc("ShellExecuteW")
	pSHBrowseForFolderW = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")

	pGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")

	pInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")

	pCoTaskMemFree = ole32.NewProc("CoTaskMemFree")
)

func call(p *syscall.LazyProc, a ...uintptr) uintptr {
	r, _, _ := p.Call(a...)
	return r
}

// pinBuf 把最近的 UTF-16 缓冲固定住,避免只以 uintptr 传参时被 GC 提前回收。
var (
	pinBuf [32][]uint16
	pinIdx int
)

func u16(s string) *uint16 {
	buf := syscall.StringToUTF16(s)
	pinBuf[pinIdx%len(pinBuf)] = buf
	pinIdx++
	return &buf[0]
}

func utf16Ptr(b []uint16) *uint16 { return &b[0] }

type point struct{ X, Y int32 }
type winRect struct{ Left, Top, Right, Bottom int32 }

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type openFileNameW struct {
	LStructSize       uint32
	HwndOwner         uintptr
	HInstance         uintptr
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        uintptr
	DwReserved        uint32
	FlagsEx           uint32
}

type browseInfoW struct {
	HwndOwner      uintptr
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

type lvItem struct {
	Mask       uint32
	IItem      int32
	ISubItem   int32
	State      uint32
	StateMask  uint32
	PszText    *uint16
	CchTextMax int32
	IImage     int32
	LParam     uintptr
}

type lvColumn struct {
	Mask       int32
	Fmt        int32
	Cx         int32
	PszText    *uint16
	CchTextMax int32
	ISubItem   int32
}

type initCommonControlsEx struct {
	Size uint32
	ICC  uint32
}

type iconInfo struct {
	FIcon    uint32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// 窗口与控件消息 / 样式常量
const (
	wmDestroy    = 0x0002
	wmClose      = 0x0010
	wmSetIcon    = 0x0080
	wmSetFont    = 0x0030
	wmCommand    = 0x0111
	wmTimer      = 0x0113
	wmDropFiles  = 0x0233
	wmUser       = 0x0400
	wmSetFocus   = 0x0007

	wsOverlapped  = 0x00000000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsMinimizeBox = 0x00020000
	wsVisible     = 0x10000000
	wsChild       = 0x40000000
	wsTabStop     = 0x00010000
	wsBorder      = 0x00800000
	wsVScroll     = 0x00200000
	wsGroup       = 0x00020000

	wsExClientEdge = 0x00000200

	ssLeft     = 0x00000000
	ssCenter   = 0x00000001
	ssNotify   = 0x00000100
	bsPushButton     = 0x00000000
	bsAutoRadioButton = 0x00000009
	cbsDropDownList = 0x0003
	esAutoHScroll   = 0x0080

	lvsReport          = 0x0001
	lvsShowSelAlways   = 0x0008
	lvsExFullRowSelect = 0x00000020
	lvsExDoubleBuffer  = 0x00010000
	lvifText           = 0x0001
	lvcfFmt            = 0x0001
	lvcfWidth          = 0x0002
	lvcfText           = 0x0004
	lvcfmtLeft         = 0x0000

	lvmFirst           = 0x1000
	lvmInsertColumnW   = lvmFirst + 97
	lvmInsertItemW     = lvmFirst + 77
	lvmSetItemTextW    = lvmFirst + 116
	lvmDeleteAllItems  = lvmFirst + 9
	lvmSetExtendedListviewStyle = lvmFirst + 54

	pbmSetPos      = wmUser + 2
	pbmSetRange32  = wmUser + 6

	cbAddString = 0x0143
	cbSetCurSel = 0x014E
	cbGetCurSel = 0x0147
	cbReset     = 0x014B
	cbSetItemData = 0x0151
	cbGetItemData = 0x0150

	bmGetCheck = 0x00F0
	bmSetCheck = 0x00F1
	bstUnchecked = 0
	bstChecked   = 1

	stnClicked = 0
	bnClicked  = 0
	cbnSelChange = 1

	swShow        = 5
	swRestore     = 9
	idcArrow      = 32512
	iconBig       = 1
	iconSmall     = 0
	defaultGuiFont = 17

	smCxScreen = 0
	smCyScreen = 1

	mbOk              = 0x0
	mbOkCancel        = 0x1
	mbYesNo           = 0x4
	mbIconError       = 0x10
	mbIconWarning     = 0x30
	mbIconInformation = 0x40
	idOk              = 1
	idCancel          = 2
	idYes             = 6
	idNo              = 7

	ofnOverwritePrompt  = 0x00000002
	ofnHideReadOnly     = 0x00000004
	ofnPathMustExist    = 0x00000800
	ofnFileMustExist    = 0x00001000
	ofnAllowMultiSelect = 0x00000200
	ofnExplorer         = 0x00080000

	bifReturnOnlyFSDirs = 0x0001
	bifNewDialogStyle   = 0x0040

	swShOpen = 0x1F // SW_SHOWNORMAL? 用 swShow=5 更通用
)

// 传给系统调用的结构体统一放在包级变量里:Go 的栈可能移动,而包级变量地址固定。
var (
	gMsg  msg
	gOFN  openFileNameW
	gBI   browseInfoW
	gICII iconInfo
	gBIH  bitmapInfoHeader
	gICCE initCommonControlsEx
	gLVI  lvItem
	gLVC  lvColumn
)

// msgBox 弹一个对话框,返回按钮 ID。
func msgBox(hwnd uintptr, text, title string, flags uintptr) uintptr {
	return call(pMessageBoxW, hwnd, uintptr(unsafe.Pointer(u16(text))),
		uintptr(unsafe.Pointer(u16(title))), flags)
}

// send 向窗口/控件发消息。
func send(hwnd, msg, wp, lp uintptr) uintptr {
	return call(pSendMessageW, hwnd, msg, wp, lp)
}

// loword/hiword 取 wParam 低位/高位字。
func loword(v uintptr) uintptr { return v & 0xFFFF }
func hiword(v uintptr) uintptr { return (v >> 16) & 0xFFFF }

// setText 设置窗口或控件文本。
func setText(hwnd uintptr, s string) {
	call(pSendMessageW, hwnd, 0x000C /*wmSetText*/, 0, uintptr(unsafe.Pointer(u16(s))))
}

// getText 获取编辑框文本。
func getText(hwnd uintptr) string {
	n := int(send(hwnd, 0x000E /*wmGetTextLength*/, 0, 0))
	buf := make([]uint16, n+2)
	call(pSendMessageW, hwnd, 0x000D /*wmGetText*/, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}

// utf16s 把字符串编码为 UTF-16 缓冲(保留结尾 0)。
func utf16s(s string) []uint16 {
	return syscall.StringToUTF16(s)
}
