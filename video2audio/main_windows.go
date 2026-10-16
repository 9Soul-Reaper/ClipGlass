//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

const (
	className = "VideoToAudioWnd"
	appTitle  = "视频转音频 VideoToAudio 1.0"
	appVer    = "1.0"
)

// 控件 ID
const (
	idAdd    = 101
	idClear  = 102
	idStart  = 103
	idCancel = 104
	idBrowse = 105
	idDrop   = 106

	idFmt       = 201
	idBr        = 202
	idRadSame   = 203
	idRadSub    = 204
	idRadCustom = 205
	idEditDir   = 206

	idList   = 301
	idProg   = 302
	idStatus = 303
	idCount  = 304

	idTimerUI = 1
)

// rowSt 列表行状态。
type rowSt int

const (
	rowPending rowSt = iota
	rowRunning
	rowDone
	rowFail
)

type row struct {
	Input   string
	Status  rowSt
	Percent float64
	Msg     string
	OutPath string
}

type gui struct {
	hwnd    uintptr
	hList   uintptr
	hDrop   uintptr
	hProg   uintptr
	hStatus uintptr
	hCount  uintptr
	hFmt    uintptr
	hBr     uintptr
	hEdit   uintptr
	hRad    [3]uintptr

	hStart  uintptr
	hCancel uintptr
	hAdd    uintptr
	hClear  uintptr
	hBrowse uintptr

	items     []row
	formatIdx int
	mode      OutMode
	running   bool
	finished  int
	runner    *Runner
	font      uintptr
}

var g *gui

func runGUI(files []string) {
	runtime.LockOSThread()
	call(pSetProcessDPIAware)

	dpi := uintptr(96)
	if pGetDpiForSystem.Find() == nil {
		if d := call(pGetDpiForSystem); d != 0 {
			dpi = d
		}
	}
	scale := float64(dpi) / 96
	sc := func(v int) int { return int(float64(v)*scale + 0.5) }

	g = &gui{mode: OutSame, formatIdx: 0}

	hinst := call(pGetModuleHandleW, 0)
	g.font = call(pGetStockObject, defaultGuiFont)

	// 公共控件(ListView / Progress)
	gICCE = initCommonControlsEx{Size: 8, ICC: 0x1 | 0x20 | 0x4000} // LISTVIEW | PROGRESS | STANDARD
	call(pInitCommonControlsEx, uintptr(unsafe.Pointer(&gICCE)))

	cls := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  hinst,
		Cursor:    call(pLoadCursorW, 0, idcArrow),
		ClassName: u16(className),
	}
	// 背景用系统窗口色:COLOR_WINDOW+1
	cls.Background = call(pGetStockObject, 0 /*WHITE_BRUSH*/)
	if call(pRegisterClassExW, uintptr(unsafe.Pointer(&cls))) == 0 {
		return
	}

	winW, winH := sc(700), sc(620)
	sx := (int(call(pGetSystemMetrics, smCxScreen)) - winW) / 2
	sy := (int(call(pGetSystemMetrics, smCyScreen)) - winH) / 2
	if sx < 0 {
		sx = 0
	}
	if sy < 0 {
		sy = 0
	}

	hwnd := call(pCreateWindowExW, 0,
		uintptr(unsafe.Pointer(u16(className))),
		uintptr(unsafe.Pointer(u16(appTitle))),
		wsOverlapped|wsCaption|wsSysMenu|wsMinimizeBox|wsVisible,
		uintptr(sx), uintptr(sy), uintptr(winW), uintptr(winH),
		0, 0, hinst, 0)
	if hwnd == 0 {
		return
	}
	g.hwnd = hwnd

	// 图标
	if ico := makeIcon(); ico != 0 {
		send(hwnd, wmSetIcon, iconBig, ico)
		send(hwnd, wmSetIcon, iconSmall, ico)
	}

	// ---- 拖放面板 ----
	g.hDrop = mkControl(hwnd, hinst, "STATIC", "拖入视频文件到此处,或点击「添加文件」(支持批量)",
		wsChild|wsVisible|ssCenter|ssNotify, wsExClientEdge,
		14, 14, 672, 88, idDrop, sc)

	// ---- 按钮行 ----
	g.hAdd = mkControl(hwnd, hinst, "BUTTON", "添加文件(&A)", wsChild|wsVisible|wsTabStop|bsPushButton, 0,
		14, 114, 110, 30, idAdd, sc)
	g.hClear = mkControl(hwnd, hinst, "BUTTON", "清空列表(&C)", wsChild|wsVisible|wsTabStop|bsPushButton, 0,
		132, 114, 110, 30, idClear, sc)
	g.hCount = mkControl(hwnd, hinst, "STATIC", "共 0 个文件", wsChild|wsVisible|ssLeft, 0,
		254, 118, 300, 22, idCount, sc)

	// ---- 列表 ----
	g.hList = mkControl(hwnd, hinst, "SysListView32", "",
		wsChild|wsVisible|wsTabStop|wsBorder|lvsReport|lvsShowSelAlways, 0,
		14, 154, 672, 260, idList, sc)
	send(g.hList, lvmSetExtendedListviewStyle, lvsExFullRowSelect|lvsExDoubleBuffer,
		lvsExFullRowSelect|lvsExDoubleBuffer)
	lvInsertCol(g.hList, 0, sc(400), "视频文件")
	lvInsertCol(g.hList, 1, sc(266), "状态")

	// ---- 输出格式行 ----
	mkControl(hwnd, hinst, "STATIC", "输出格式:", wsChild|wsVisible|ssLeft, 0,
		14, 430, 72, 22, 0, sc)
	g.hFmt = mkControl(hwnd, hinst, "COMBOBOX", "",
		wsChild|wsVisible|wsTabStop|cbsDropDownList|wsVScroll, 0,
		88, 426, 300, 240, idFmt, sc)
	mkControl(hwnd, hinst, "STATIC", "音质:", wsChild|wsVisible|ssLeft, 0,
		400, 430, 48, 22, 0, sc)
	g.hBr = mkControl(hwnd, hinst, "COMBOBOX", "",
		wsChild|wsVisible|wsTabStop|cbsDropDownList|wsVScroll, 0,
		452, 426, 150, 240, idBr, sc)

	for _, f := range FmtList {
		send(g.hFmt, cbAddString, 0, uintptr(unsafe.Pointer(u16(f.Label))))
	}
	send(g.hFmt, cbSetCurSel, 0, 0)
	g.fillBitrates()

	// ---- 输出位置行 ----
	g.hRad[0] = mkControl(hwnd, hinst, "BUTTON", "保存到源文件夹",
		wsChild|wsVisible|wsTabStop|bsAutoRadioButton|wsGroup, 0,
		14, 468, 160, 24, idRadSame, sc)
	g.hRad[1] = mkControl(hwnd, hinst, "BUTTON", "保存到 audio 子文件夹",
		wsChild|wsVisible|wsTabStop|bsAutoRadioButton, 0,
		180, 468, 220, 24, idRadSub, sc)
	g.hRad[2] = mkControl(hwnd, hinst, "BUTTON", "自定义文件夹:",
		wsChild|wsVisible|wsTabStop|bsAutoRadioButton, 0,
		410, 468, 120, 24, idRadCustom, sc)
	send(g.hRad[0], bmSetCheck, bstChecked, 0)

	g.hEdit = mkControl(hwnd, hinst, "EDIT", "",
		wsChild|wsVisible|wsTabStop|wsBorder|esAutoHScroll, 0,
		14, 502, 556, 26, idEditDir, sc)
	g.hBrowse = mkControl(hwnd, hinst, "BUTTON", "浏览(&B)...",
		wsChild|wsVisible|wsTabStop|bsPushButton, 0,
		580, 501, 106, 28, idBrowse, sc)
	send(g.hEdit, wmSetFont, g.font, 1)
	g.setMode(OutSame)

	// ---- 操作行 ----
	g.hStart = mkControl(hwnd, hinst, "BUTTON", "开始转换(&S)",
		wsChild|wsVisible|wsTabStop|bsPushButton, 0,
		14, 544, 160, 34, idStart, sc)
	g.hCancel = mkControl(hwnd, hinst, "BUTTON", "取消",
		wsChild|wsVisible|wsTabStop|bsPushButton, 0,
		184, 545, 100, 32, idCancel, sc)
	g.hProg = mkControl(hwnd, hinst, "msctls_progress32", "",
		wsChild|wsVisible, 0,
		300, 552, 386, 18, idProg, sc)
	send(g.hProg, pbmSetRange32, 0, 100)
	send(g.hProg, pbmSetPos, 0, 0)

	g.hStatus = mkControl(hwnd, hinst, "STATIC", "就绪 · 添加视频后点击「开始转换」",
		wsChild|wsVisible|ssLeft, 0,
		14, 588, 672, 22, idStatus, sc)

	// 统一字体
	for _, h := range []uintptr{g.hDrop, g.hAdd, g.hClear, g.hCount, g.hFmt, g.hBr,
		g.hRad[0], g.hRad[1], g.hRad[2], g.hEdit, g.hBrowse, g.hStart, g.hCancel,
		g.hStatus, g.hList} {
		send(h, wmSetFont, g.font, 1)
	}

	call(pDragAcceptFiles, hwnd, 1)
	call(pSetTimer, hwnd, idTimerUI, 100, 0)
	call(pEnableWindow, g.hCancel, 0)

	// 命令行带文件:直接加入
	g.addFiles(files)

	call(pShowWindow, hwnd, swShow)
	call(pUpdateWindow, hwnd)
	call(pSetForegroundWindow, hwnd)

	for {
		r := call(pGetMessageW, uintptr(unsafe.Pointer(&gMsg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		call(pTranslateMessage, uintptr(unsafe.Pointer(&gMsg)))
		call(pDispatchMessageW, uintptr(unsafe.Pointer(&gMsg)))
	}
}

// mkControl 创建一个子控件(自动按 DPI 缩放坐标),返回句柄。
func mkControl(parent, hinst uintptr, class, text string, style, exStyle uintptr,
	x, y, w, h, id int, sc func(int) int) uintptr {
	return call(pCreateWindowExW, exStyle,
		uintptr(unsafe.Pointer(u16(class))),
		uintptr(unsafe.Pointer(u16(text))),
		style,
		uintptr(sc(x)), uintptr(sc(y)), uintptr(sc(w)), uintptr(sc(h)),
		parent, uintptr(id), hinst, 0)
}

func wndProc(hwnd, m, wp, lp uintptr) (ret uintptr) {
	if g == nil {
		return call(pDefWindowProcW, hwnd, m, wp, lp)
	}
	switch m {
	case wmCommand:
		id := int32(loword(wp))
		code := int32(hiword(wp))
		switch id {
		case idAdd, idDrop:
			g.pickFiles()
		case idClear:
			if !g.running {
				g.items = nil
				g.refreshList()
				g.setStatus("已清空列表")
			}
		case idStart:
			g.start()
		case idCancel:
			g.cancel()
		case idBrowse:
			g.browseDir()
		case idFmt:
			if code == cbnSelChange {
				if sel := int(send(g.hFmt, cbGetCurSel, 0, 0)); sel >= 0 {
					g.formatIdx = sel
					g.fillBitrates()
				}
			}
		case idRadSame:
			g.setMode(OutSame)
		case idRadSub:
			g.setMode(OutSub)
		case idRadCustom:
			g.setMode(OutCustom)
		}
	case wmDropFiles:
		g.onDrop(wp)
	case wmTimer:
		if wp == idTimerUI {
			g.drainEvents()
		}
	case wmClose:
		if g.running {
			if msgBox(hwnd, "正在转换,确定要退出吗?", appTitle, mbYesNo|mbIconWarning) != idYes {
				return 0
			}
			g.cancel()
		}
		call(pDestroyWindow, hwnd)
		return 0
	case wmDestroy:
		call(pPostQuitMessage, 0)
		return 0
	}
	return call(pDefWindowProcW, hwnd, m, wp, lp)
}

// ---------- 文件列表 ----------

func (g *gui) addFiles(list []string) {
	added := 0
	for _, f := range list {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		dup := false
		for i := range g.items {
			if sameFilePath(g.items[i].Input, f) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		g.items = append(g.items, row{Input: f, Status: rowPending, Msg: "等待中"})
		added++
	}
	if added > 0 {
		g.refreshList()
	}
	g.updateCount()
}

func (g *gui) refreshList() {
	send(g.hList, lvmDeleteAllItems, 0, 0)
	for i := range g.items {
		lvInsertRow(g.hList, i, filepath.Base(g.items[i].Input))
		lvSetText(g.hList, i, 1, g.statusText(g.items[i]))
	}
}

func (g *gui) statusText(r row) string {
	switch r.Status {
	case rowRunning:
		return fmt.Sprintf("转换中 %d%%", int(r.Percent*100+0.5))
	case rowDone:
		return "完成 → " + filepath.Base(r.OutPath)
	case rowFail:
		return r.Msg
	}
	return "等待中"
}

func (g *gui) updateRow(i int) {
	if i < 0 || i >= len(g.items) {
		return
	}
	lvSetText(g.hList, i, 1, g.statusText(g.items[i]))
}

func (g *gui) updateCount() {
	setText(g.hCount, fmt.Sprintf("共 %d 个文件", len(g.items)))
}

func (g *gui) setStatus(s string) {
	setText(g.hStatus, s)
}

// ---------- 文件选择 ----------

var videoFilter = utf16z("媒体文件", "*.mp4;*.mkv;*.avi;*.mov;*.flv;*.wmv;*.webm;*.m4v;*.ts;*.m2ts;*.mts;*.3gp;*.mpg;*.mpeg;*.vob;*.rm;*.rmvb;*.ogv;*.f4v;*.asf;*.divx;*.mp3;*.wav;*.m4a;*.wma;*.aac;*.flac;*.ogg;*.opus", "所有文件", "*.*")

func (g *gui) pickFiles() {
	if g.running {
		return
	}
	buf := make([]uint16, 65536)
	gOFN = openFileNameW{
		LStructSize: uint32(unsafe.Sizeof(openFileNameW{})),
		HwndOwner:   g.hwnd,
		LpstrFilter: utf16Ptr(videoFilter),
		LpstrFile:   utf16Ptr(buf),
		NMaxFile:    uint32(len(buf)),
		LpstrTitle:  u16("选择视频文件"),
		Flags:       ofnExplorer | ofnAllowMultiSelect | ofnFileMustExist | ofnPathMustExist | ofnHideReadOnly,
	}
	if call(pGetOpenFileNameW, uintptr(unsafe.Pointer(&gOFN))) == 0 {
		return
	}
	g.addFiles(parseMultiSelect(buf))
}

// parseMultiSelect 解析 GetOpenFileName 多选缓冲:目录\0文件1\0文件2\0\0
func parseMultiSelect(buf []uint16) []string {
	var parts []string
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] != 0 {
			continue
		}
		s := syscall.UTF16ToString(buf[start:i])
		if s == "" {
			break
		}
		parts = append(parts, s)
		start = i + 1
	}
	if len(parts) == 0 {
		return nil
	}
	if len(parts) == 1 {
		return parts // 单选:完整路径
	}
	dir := parts[0]
	var out []string
	for _, name := range parts[1:] {
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

func (g *gui) onDrop(hDrop uintptr) {
	n := int(call(pDragQueryFileW, hDrop, 0xFFFFFFFF, 0, 0))
	var list []string
	for i := 0; i < n; i++ {
		buf := make([]uint16, 32768)
		call(pDragQueryFileW, hDrop, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		list = append(list, syscall.UTF16ToString(buf))
	}
	call(pDragFinish, hDrop)
	g.addFiles(list)
}

// ---------- 输出位置 ----------

func (g *gui) setMode(m OutMode) {
	g.mode = m
	ids := [3]OutMode{OutSame, OutSub, OutCustom}
	for i := 0; i < 3; i++ {
		v := uintptr(bstUnchecked)
		if ids[i] == m {
			v = bstChecked
		}
		send(g.hRad[i], bmSetCheck, v, 0)
	}
	if m == OutCustom {
		call(pEnableWindow, g.hEdit, 1)
		call(pEnableWindow, g.hBrowse, 1)
	} else {
		call(pEnableWindow, g.hEdit, 0)
		call(pEnableWindow, g.hBrowse, 0)
	}
}

func (g *gui) browseDir() {
	disp := make([]uint16, 260)
	gBI = browseInfoW{
		HwndOwner:      g.hwnd,
		PszDisplayName: utf16Ptr(disp),
		LpszTitle:      u16("选择音频输出文件夹"),
		UlFlags:        bifReturnOnlyFSDirs | bifNewDialogStyle,
	}
	pidl := call(pSHBrowseForFolderW, uintptr(unsafe.Pointer(&gBI)))
	if pidl == 0 {
		return
	}
	buf := make([]uint16, 1024)
	call(pSHGetPathFromIDListW, pidl, uintptr(unsafe.Pointer(&buf[0])))
	call(pCoTaskMemFree, pidl)
	if p := syscall.UTF16ToString(buf); p != "" {
		setText(g.hEdit, p)
		g.setMode(OutCustom)
	}
}

// ---------- 格式 / 音质 ----------

func (g *gui) fillBitrates() {
	send(g.hBr, cbReset, 0, 0)
	brs := fmtDef(FmtList[g.formatIdx].Key).Bitrates
	if len(brs) == 0 {
		send(g.hBr, cbAddString, 0, uintptr(unsafe.Pointer(u16("—"))))
		send(g.hBr, cbSetCurSel, 0, 0)
		call(pEnableWindow, g.hBr, 0)
		return
	}
	call(pEnableWindow, g.hBr, 1)
	for _, b := range brs {
		send(g.hBr, cbAddString, 0, uintptr(unsafe.Pointer(u16(b))))
	}
	sel := 1
	if sel >= len(brs) {
		sel = len(brs) - 1
	}
	send(g.hBr, cbSetCurSel, uintptr(sel), 0)
}

func (g *gui) currentJob() Job {
	job := Job{
		Format:  FmtList[g.formatIdx].Key,
		Mode:    g.mode,
		Bitrate: "",
	}
	if brs := fmtDef(job.Format).Bitrates; len(brs) > 0 {
		if sel := int(send(g.hBr, cbGetCurSel, 0, 0)); sel >= 0 && sel < len(brs) {
			job.Bitrate = brs[sel]
		}
	}
	if g.mode == OutCustom {
		job.CustomDir = getText(g.hEdit)
	}
	return job
}

// ---------- 转换控制 ----------

func (g *gui) start() {
	if g.running {
		return
	}
	if len(g.items) == 0 {
		msgBox(g.hwnd, "请先添加视频文件。", appTitle, mbOk|mbIconInformation)
		return
	}
	job := g.currentJob()
	if job.Mode == OutCustom && strings.TrimSpace(job.CustomDir) == "" {
		msgBox(g.hwnd, "请填写或浏览选择输出文件夹。", appTitle, mbOk|mbIconWarning)
		return
	}
	ff, err := locateFFmpeg()
	if err != nil {
		msgBox(g.hwnd, err.Error(), appTitle, mbOk|mbIconError)
		return
	}

	files := make([]string, len(g.items))
	for i := range g.items {
		files[i] = g.items[i].Input
		g.items[i].Status = rowPending
		g.items[i].Percent = 0
		g.items[i].Msg = "等待中"
		g.updateRow(i)
	}
	g.running = true
	g.finished = 0
	g.setRunningUI(true)
	send(g.hProg, pbmSetPos, 0, 0)
	g.setStatus("正在准备转换...")
	g.runner = NewRunner(ff)
	g.runner.Start(files, job)
}

func (g *gui) cancel() {
	if g.running && g.runner != nil {
		g.runner.Cancel()
		g.setStatus("正在取消...")
	}
}

func (g *gui) setRunningUI(on bool) {
	for _, h := range []uintptr{g.hAdd, g.hClear, g.hFmt, g.hBr, g.hStart,
		g.hRad[0], g.hRad[1], g.hRad[2]} {
		v := uintptr(1)
		if on {
			v = 0
		}
		call(pEnableWindow, h, v)
	}
	if on {
		call(pEnableWindow, g.hCancel, 1)
	} else {
		call(pEnableWindow, g.hCancel, 0)
		if g.mode == OutCustom {
			call(pEnableWindow, g.hEdit, 1)
			call(pEnableWindow, g.hBrowse, 1)
		} else {
			call(pEnableWindow, g.hEdit, 0)
			call(pEnableWindow, g.hBrowse, 0)
		}
	}
}

func (g *gui) drainEvents() {
	if g.runner == nil {
		return
	}
	for {
		select {
		case ev := <-g.runner.EvCh:
			g.applyEv(ev)
		default:
			return
		}
	}
}

func (g *gui) applyEv(ev Ev) {
	total := len(g.items)
	switch ev.Type {
	case EvFileStart:
		if ev.Index < total {
			g.items[ev.Index].Status = rowRunning
			g.items[ev.Index].Percent = 0
			g.updateRow(ev.Index)
			g.setStatus(fmt.Sprintf("正在转换 %d/%d:%s",
				ev.Index+1, total, filepath.Base(g.items[ev.Index].Input)))
		}
	case EvProgress:
		if ev.Index < total {
			g.items[ev.Index].Percent = ev.Percent
			g.updateRow(ev.Index)
			if total > 0 {
				overall := (float64(g.finished) + ev.Percent) / float64(total)
				send(g.hProg, pbmSetPos, uintptr(int(overall*100)), 0)
			}
		}
	case EvFileDone:
		if ev.Index < total {
			g.items[ev.Index].Status = rowDone
			g.items[ev.Index].OutPath = ev.OutPath
			g.updateRow(ev.Index)
			g.finished++
		}
	case EvFileFail:
		if ev.Index < total {
			g.items[ev.Index].Status = rowFail
			g.items[ev.Index].Msg = "失败:" + ev.Err
			g.updateRow(ev.Index)
			g.finished++
		}
	case EvQueueDone:
		g.running = false
		g.setRunningUI(false)
		send(g.hProg, pbmSetPos, 100, 0)
		if ev.Fail == 0 {
			g.setStatus(fmt.Sprintf("全部完成!共 %d 个文件", ev.OK))
		} else {
			g.setStatus(fmt.Sprintf("完成 %d 个,失败 %d 个(失败原因见状态列)", ev.OK, ev.Fail))
		}
	}
}

// ---------- 列表控件 ----------

func lvInsertCol(h uintptr, idx, w int, title string) {
	gLVC = lvColumn{
		Mask:    lvcfFmt | lvcfWidth | lvcfText,
		Fmt:     lvcfmtLeft,
		Cx:      int32(w),
		PszText: u16(title),
	}
	send(h, lvmInsertColumnW, uintptr(idx), uintptr(unsafe.Pointer(&gLVC)))
}

func lvInsertRow(h uintptr, idx int, name string) {
	gLVI = lvItem{Mask: lvifText, IItem: int32(idx), PszText: u16(name)}
	send(h, lvmInsertItemW, 0, uintptr(unsafe.Pointer(&gLVI)))
}

func lvSetText(h uintptr, idx, sub int, s string) {
	gLVI = lvItem{Mask: lvifText, ISubItem: int32(sub), PszText: u16(s)}
	// LVM_SETITEMTEXT: wParam = 行号,lParam = LVITEM(iSubItem + pszText)
	send(h, lvmSetItemTextW, uintptr(idx), uintptr(unsafe.Pointer(&gLVI)))
}

// utf16z 把若干段字符串编码为 \0 分隔、以 \0\0 结尾的 UTF-16 缓冲。
func utf16z(parts ...string) []uint16 {
	var out []uint16
	for _, p := range parts {
		for _, r := range p {
			if r < 0x10000 {
				out = append(out, uint16(r))
			} else {
				r -= 0x10000
				out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
			}
		}
		out = append(out, 0)
	}
	return append(out, 0)
}

// makeIcon 绘制应用图标(蓝紫渐变 + 均衡器竖条),返回 HICON。
func makeIcon() uintptr {
	const s = 32
	bih := bitmapInfoHeader{Size: 40, Width: s, Height: -s, Planes: 1, BitCount: 32}
	var bits uintptr
	hdc := call(pGetDC, 0)
	dib := call(pCreateDIBSection, hdc, uintptr(unsafe.Pointer(&bih)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	call(pReleaseDC, 0, hdc)
	if dib == 0 || bits == 0 {
		return 0
	}
	px := unsafe.Slice((*byte)(unsafe.Pointer(bits)), s*s*4)
	// 渐变背景
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			t := (float64(x) + float64(y)) / (2 * s)
			i := (y*s + x) * 4
			px[i] = byte(255 - 60*t)   // B
			px[i+1] = byte(150 - 60*t) // G
			px[i+2] = byte(70 + 120*t) // R
			px[i+3] = 255
		}
	}
	// 均衡器竖条(白色)
	bars := []struct{ x, h int }{{6, 12}, {12, 22}, {18, 16}, {24, 26}}
	for _, b := range bars {
		y0 := (s - b.h) / 2
		for y := y0; y < y0+b.h; y++ {
			for x := b.x; x < b.x+3 && x < s; x++ {
				i := (y*s + x) * 4
				px[i], px[i+1], px[i+2], px[i+3] = 255, 255, 255, 255
			}
		}
	}
	// 1bpp 掩码:圆角外透明
	const r = 8
	mask := make([]byte, s*s/8)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			dx, dy := 0.0, 0.0
			switch {
			case x < r && y < r:
				dx, dy = float64(r-x), float64(r-y)
			case x >= s-r && y < r:
				dx, dy = float64(x-(s-r-1)), float64(r-y)
			case x < r && y >= s-r:
				dx, dy = float64(r-x), float64(y-(s-r-1))
			case x >= s-r && y >= s-r:
				dx, dy = float64(x-(s-r-1)), float64(y-(s-r-1))
			}
			if dx > 0 && dx*dx+dy*dy > r*r {
				mask[y*(s/8)+x/8] |= 1 << (7 - x%8)
			}
		}
	}
	hbmMask := call(pCreateBitmap, s, s, 1, 1, uintptr(unsafe.Pointer(&mask[0])))
	gICII = iconInfo{FIcon: 1, HbmMask: hbmMask, HbmColor: dib}
	hicon := call(pCreateIconIndirect, uintptr(unsafe.Pointer(&gICII)))
	call(pDeleteObject, hbmMask)
	call(pDeleteObject, dib)
	return hicon
}
