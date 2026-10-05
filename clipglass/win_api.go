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
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	imm32    = syscall.NewLazyDLL("imm32.dll")

	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pGetMessageW              = user32.NewProc("GetMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pPostQuitMessage          = user32.NewProc("PostQuitMessage")
	pPostMessageW             = user32.NewProc("PostMessageW")
	pShowWindow               = user32.NewProc("ShowWindow")
	pSetWindowPos             = user32.NewProc("SetWindowPos")
	pGetWindowRect            = user32.NewProc("GetWindowRect")
	pIsWindowVisible          = user32.NewProc("IsWindowVisible")
	pSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	pGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	pSetFocus                 = user32.NewProc("SetFocus")
	pGetCursorPos             = user32.NewProc("GetCursorPos")
	pMonitorFromPoint         = user32.NewProc("MonitorFromPoint")
	pGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	pSetTimer                 = user32.NewProc("SetTimer")
	pKillTimer                = user32.NewProc("KillTimer")
	pRegisterHotKey           = user32.NewProc("RegisterHotKey")
	pUnregisterHotKey         = user32.NewProc("UnregisterHotKey")
	pAddClipboardFormatListen = user32.NewProc("AddClipboardFormatListener")
	pRemoveClipboardListener  = user32.NewProc("RemoveClipboardFormatListener")
	pOpenClipboard            = user32.NewProc("OpenClipboard")
	pCloseClipboard           = user32.NewProc("CloseClipboard")
	pEmptyClipboard           = user32.NewProc("EmptyClipboard")
	pGetClipboardData         = user32.NewProc("GetClipboardData")
	pSetClipboardData         = user32.NewProc("SetClipboardData")
	pIsClipboardFormatAvail   = user32.NewProc("IsClipboardFormatAvailable")
	pRegisterClipboardFormatW = user32.NewProc("RegisterClipboardFormatW")
	pGetClipboardOwner        = user32.NewProc("GetClipboardOwner")
	pGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	pAttachThreadInput        = user32.NewProc("AttachThreadInput")
	pKeybdEvent               = user32.NewProc("keybd_event")
	pGetKeyState              = user32.NewProc("GetKeyState")
	pLoadCursorW              = user32.NewProc("LoadCursorW")
	pUpdateLayeredWindow      = user32.NewProc("UpdateLayeredWindow")
	pGetDC                    = user32.NewProc("GetDC")
	pReleaseDC                = user32.NewProc("ReleaseDC")
	pSetProcessDPIAware       = user32.NewProc("SetProcessDPIAware")
	pGetDpiForSystem          = user32.NewProc("GetDpiForSystem")
	pTrackMouseEvent          = user32.NewProc("TrackMouseEvent")
	pCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	pAppendMenuW              = user32.NewProc("AppendMenuW")
	pTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	pDestroyMenu              = user32.NewProc("DestroyMenu")
	pCreateIconIndirect       = user32.NewProc("CreateIconIndirect")
	pFindWindowW              = user32.NewProc("FindWindowW")
	pRegisterWindowMessageW   = user32.NewProc("RegisterWindowMessageW")
	pDestroyWindow            = user32.NewProc("DestroyWindow")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pBitBlt             = gdi32.NewProc("BitBlt")
	pCreateFontW        = gdi32.NewProc("CreateFontW")
	pSetTextColor       = gdi32.NewProc("SetTextColor")
	pSetBkMode          = gdi32.NewProc("SetBkMode")
	pDrawTextW          = user32.NewProc("DrawTextW")
	pGetTextExtentPoint = gdi32.NewProc("GetTextExtentPoint32W")
	pIntersectClipRect  = gdi32.NewProc("IntersectClipRect")
	pSelectClipRgn      = gdi32.NewProc("SelectClipRgn")
	pCreateBitmap       = gdi32.NewProc("CreateBitmap")

	pGetModuleHandleW        = kernel32.NewProc("GetModuleHandleW")
	pCreateMutexW            = kernel32.NewProc("CreateMutexW")
	pGlobalAlloc             = kernel32.NewProc("GlobalAlloc")
	pGlobalLock              = kernel32.NewProc("GlobalLock")
	pGlobalUnlock            = kernel32.NewProc("GlobalUnlock")
	pGlobalSize              = kernel32.NewProc("GlobalSize")
	pOpenProcess             = kernel32.NewProc("OpenProcess")
	pCloseHandle             = kernel32.NewProc("CloseHandle")
	pQueryFullProcessImageNm = kernel32.NewProc("QueryFullProcessImageNameW")
	pGetCurrentThreadId      = kernel32.NewProc("GetCurrentThreadId")

	pShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	pShellExecuteW    = shell32.NewProc("ShellExecuteW")

	pRegCreateKeyExW  = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	pRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	pRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	pRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	pRegCloseKey      = advapi32.NewProc("RegCloseKey")

	pImmGetContext         = imm32.NewProc("ImmGetContext")
	pImmReleaseContext     = imm32.NewProc("ImmReleaseContext")
	pImmSetCompositionWnd  = imm32.NewProc("ImmSetCompositionWindow")
	pImmSetCandidateWindow = imm32.NewProc("ImmSetCandidateWindow")
)

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

type bitmapInfoHeader struct {
	Size          uint32
	Width, Height int32
	Planes, Bits  uint16
	Compression   uint32
	SizeImage     uint32
	XPels, YPels  int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type blendFunction struct {
	Op, Flags, Alpha, Format byte
}

type sizeT struct{ CX, CY int32 }

type monitorInfo struct {
	Size    uint32
	Monitor winRect
	Work    winRect
	Flags   uint32
}

type trackMouseEvent struct {
	Size  uint32
	Flags uint32
	HWnd  uintptr
	Hover uint32
}

type iconInfo struct {
	IsIcon   uint32
	XHot     uint32
	YHot     uint32
	MaskBmp  uintptr
	ColorBmp uintptr
}

type notifyIconData struct {
	Size            uint32
	HWnd            uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GuidItem        [16]byte
	BalloonIcon     uintptr
}

type compositionForm struct {
	Style  uint32
	CurPos point
	Area   winRect
}

type candidateForm struct {
	Index  uint32
	Style  uint32
	CurPos point
	Area   winRect
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func call(p *syscall.LazyProc, a ...uintptr) uintptr {
	r, _, _ := p.Call(a...)
	return r
}

func ptr(v unsafe.Pointer) uintptr { return uintptr(v) }

func boolArg(b bool) uintptr {
	if b {
		return 1
	}
	return 0
}

var pGetSystemMetrics = user32.NewProc("GetSystemMetrics")

var pGdiFlush = gdi32.NewProc("GdiFlush")

var pDragQueryFileW = shell32.NewProc("DragQueryFileW")
var pGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")

var (
	pGetWindowTextW      = user32.NewProc("GetWindowTextW")
	pTerminateProcess    = kernel32.NewProc("TerminateProcess")
	pWaitForSingleObject = kernel32.NewProc("WaitForSingleObject")
)
