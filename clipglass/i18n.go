package main

import "fmt"

// 语言顺序:简体中文 / English / 繁體中文 / 日本語 / 한국어
var langCodes = []string{"zh-CN", "en", "zh-TW", "ja", "ko", "es", "fr", "de", "pt", "ru", "it", "tr", "vi", "id"}
var langNames = []string{"简体中文", "English", "繁體中文", "日本語", "한국어", "Español", "Français", "Deutsch", "Português", "Русский", "Italiano", "Türkçe", "Tiếng Việt", "Bahasa Indonesia"}
var langFaces = []string{"Microsoft YaHei UI", "Microsoft YaHei UI", "Microsoft JhengHei UI", "Yu Gothic UI", "Malgun Gothic",
	"Segoe UI", "Segoe UI", "Segoe UI", "Segoe UI", "Segoe UI", "Segoe UI", "Segoe UI", "Segoe UI", "Segoe UI"}

var curLang = 0

// ApplyLang 接受 "auto" 或语言代码;auto 时使用 sys(由宿主按系统语言给出)。
func ApplyLang(setting, sys string) {
	code := setting
	if code == "" || code == "auto" {
		code = sys
	}
	for i, c := range langCodes {
		if c == code {
			curLang = i
			return
		}
	}
	curLang = 1
}

func CurFace() string { return langFaces[curLang] }

// langFromLangID 把 Windows LANGID 映射到支持的语言代码。
func langFromLangID(id uint16) string {
	switch id & 0x3ff {
	case 0x04:
		switch id {
		case 0x0404, 0x0c04, 0x1404:
			return "zh-TW"
		}
		return "zh-CN"
	case 0x11:
		return "ja"
	case 0x12:
		return "ko"
	case 0x0A:
		return "es"
	case 0x0C:
		return "fr"
	case 0x07:
		return "de"
	case 0x16:
		return "pt"
	case 0x19:
		return "ru"
	case 0x10:
		return "it"
	case 0x1F:
		return "tr"
	case 0x2A:
		return "vi"
	case 0x21:
		return "id"
	}
	return "en"
}

// T 取当前语言的文案;缺失时回退英文,再回退 key 本身。
func T(key string) string {
	if curLang >= 5 {
		if v, ok := moreLangs[curLang-5][key]; ok {
			return v
		}
	} else if v, ok := strs[key]; ok && v[curLang] != "" {
		return v[curLang]
	}
	if v, ok := strs[key]; ok && v[1] != "" {
		return v[1]
	}
	return key
}

func Tf(key string, a ...interface{}) string { return fmt.Sprintf(T(key), a...) }

// 每项依次为: 简中, English, 繁中, 日本語, 한국어
var strs = map[string][5]string{
	"title":    {"剪贴板", "Clipboard", "剪貼簿", "クリップボード", "클립보드"},
	"settings": {"设置", "Settings", "設定", "設定", "설정"},
	"back":     {"‹ 返回", "‹ Back", "‹ 返回", "‹ 戻る", "‹ 뒤로"},
	"pause":    {"暂停", "Pause", "暫停", "一時停止", "일시정지"},
	"resume":   {"已暂停 · 继续", "Paused · Resume", "已暫停 · 繼續", "停止中 · 再開", "정지됨 · 재개"},
	"search":   {"搜索剪贴板…  直接输入即可", "Search clipboard…  just start typing", "搜尋剪貼簿…  直接輸入即可", "クリップボードを検索…  そのまま入力", "클립보드 검색…  바로 입력"},
	"count":    {"%d 条记录", "%d items", "%d 筆記錄", "%d 件", "%d개 항목"},

	"chip.all":     {"全部", "All", "全部", "すべて", "전체"},
	"chip.link":    {"链接", "Links", "連結", "リンク", "링크"},
	"chip.code":    {"代码", "Code", "程式碼", "コード", "코드"},
	"chip.contact": {"联系", "Contact", "聯絡", "連絡先", "연락처"},
	"chip.path":    {"路径", "Paths", "路徑", "パス", "경로"},
	"chip.number":  {"数字", "Numbers", "數字", "数字", "숫자"},
	"chip.image":   {"图片", "Images", "圖片", "画像", "이미지"},
	"chip.pinned":  {"置顶", "Pinned", "置頂", "固定", "고정"},

	"kt.text":    {"文", "Tx", "文", "文", "문"},
	"kt.link":    {"链", "Lk", "連", "リ", "링"},
	"kt.code":    {"码", "</>", "碼", "コ", "코"},
	"kt.contact": {"联", "@", "聯", "連", "연"},
	"kt.path":    {"路", "Dir", "路", "パ", "경"},
	"kt.number":  {"数", "123", "數", "数", "숫"},
	"kt.image":   {"图", "Img", "圖", "画", "이"},

	"empty.t":   {"还没有内容", "Nothing here yet", "還沒有內容", "まだ何もありません", "아직 내용이 없습니다"},
	"empty.s":   {"复制文字或图片,它会出现在这里", "Copy text or an image and it shows up here", "複製文字或圖片,它會出現在這裡", "テキストや画像をコピーするとここに表示されます", "텍스트나 이미지를 복사하면 여기에 표시됩니다"},
	"nores.t":   {"没有匹配的结果", "No matches", "沒有符合的結果", "一致する結果がありません", "일치하는 결과가 없습니다"},
	"nores.s":   {"换个关键词或分类试试", "Try another keyword or category", "換個關鍵字或分類試試", "キーワードやカテゴリを変えてみてください", "다른 키워드나 분류를 시도해 보세요"},
	"hint.list": {"↑↓ 选择 · Enter 粘贴 · Alt+1~9 快选 · Del 删除 · Ctrl+P 置顶", "↑↓ Select · Enter Paste · Alt+1-9 Quick · Del Delete · Ctrl+P Pin", "↑↓ 選擇 · Enter 貼上 · Alt+1~9 快選 · Del 刪除 · Ctrl+P 置頂", "↑↓ 選択 · Enter 貼付 · Alt+1~9 即選択 · Del 削除 · Ctrl+P 固定", "↑↓ 선택 · Enter 붙여넣기 · Alt+1~9 빠른선택 · Del 삭제 · Ctrl+P 고정"},
	"hint.set":  {"点击任意一行即可切换 · Esc 返回", "Click a row to change it · Esc to go back", "點擊任意一列即可切換 · Esc 返回", "行をクリックして変更 · Esc で戻る", "행을 클릭해 변경 · Esc로 돌아가기"},

	"t.paused":    {"已暂停记录", "Recording paused", "已暫停記錄", "記録を一時停止しました", "기록을 일시정지했습니다"},
	"t.resumed":   {"已恢复记录", "Recording resumed", "已恢復記錄", "記録を再開しました", "기록을 재개했습니다"},
	"t.clear2":    {"再点一次确认清空", "Click again to confirm", "再點一次確認清空", "もう一度クリックで確定", "한 번 더 누르면 삭제됩니다"},
	"t.cleared":   {"已清空(保留置顶项)", "Cleared (pinned kept)", "已清空(保留置頂項)", "消去しました(固定は保持)", "삭제됨(고정 항목 유지)"},
	"t.hkset":     {"快捷键已设为 %s", "Shortcut set to %s", "快捷鍵已設為 %s", "ショートカットを %s に設定しました", "단축키를 %s(으)로 설정했습니다"},
	"t.hkfail":    {"该组合键已被占用,请换一个", "That shortcut is taken — try another", "該組合鍵已被占用,請換一個", "このショートカットは使用中です", "이미 사용 중인 단축키입니다"},
	"t.hkneed":    {"需包含 Ctrl / Alt / Shift / Win 之一", "Include Ctrl, Alt, Shift or Win", "需包含 Ctrl / Alt / Shift / Win 之一", "Ctrl / Alt / Shift / Win のいずれかを含めてください", "Ctrl / Alt / Shift / Win 중 하나를 포함하세요"},
	"t.hkrec":     {"请按下新的快捷键…(Esc 取消)", "Press the new shortcut… (Esc cancels)", "請按下新的快捷鍵…(Esc 取消)", "新しいショートカットを押してください(Esc で取消)", "새 단축키를 누르세요 (Esc 취소)"},
	"t.hkcleared": {"已清除快捷键", "Shortcut cleared", "已清除快捷鍵", "ショートカットを解除しました", "단축키를 해제했습니다"},
	"t.ignored":   {"已忽略来自 %s 的内容", "Ignoring clips from %s", "已忽略來自 %s 的內容", "%s からのコピーを無視します", "%s의 내용을 무시합니다"},
	"t.first":     {"按 %s 随时呼出", "Press %s anytime to open", "按 %s 隨時呼出", "%s でいつでも呼び出せます", "%s 키로 언제든 열 수 있습니다"},
	"t.hkbusy":    {"呼出热键被占用,请在设置中更换", "Open shortcut is in use — change it in Settings", "呼出熱鍵被占用,請在設定中更換", "呼び出しキーが使用中です。設定で変更してください", "열기 단축키가 사용 중입니다. 설정에서 변경하세요"},
	"t.exported":  {"已导出到数据文件夹", "Exported to the data folder", "已匯出至資料夾", "データフォルダーにエクスポートしました", "데이터 폴더로 내보냈습니다"},
	"t.ignreset":  {"已重置忽略列表", "Ignore list reset", "已重設忽略清單", "無視リストを初期化しました", "무시 목록을 초기화했습니다"},

	"s.general": {"通用", "General", "一般", "一般", "일반"},
	"s.look":    {"外观", "Appearance", "外觀", "外観", "모양"},
	"s.privacy": {"隐私与数据", "Privacy & Data", "隱私與資料", "プライバシーとデータ", "개인정보 및 데이터"},

	"r.lang":        {"语言", "Language", "語言", "言語", "언어"},
	"r.lang.d":      {"界面显示语言", "Interface language", "介面顯示語言", "表示言語", "표시 언어"},
	"lang.auto":     {"自动", "Auto", "自動", "自動", "자동"},
	"r.hotkey":      {"呼出快捷键", "Open shortcut", "呼出快捷鍵", "呼び出しキー", "열기 단축키"},
	"r.hotkey.d":    {"点击后按下新的组合键", "Click, then press a new combination", "點擊後按下新的組合鍵", "クリック後に新しい組み合わせを押す", "클릭 후 새 조합을 누르세요"},
	"r.pausekey":    {"暂停/恢复快捷键", "Pause/resume shortcut", "暫停/恢復快捷鍵", "一時停止/再開キー", "일시정지/재개 단축키"},
	"r.pausekey.d":  {"全局切换是否记录(可选,Delete 清除)", "Toggle recording globally (optional, Delete clears)", "全域切換是否記錄(可選,Delete 清除)", "記録の切り替え(任意、Delete で解除)", "기록 전환(선택, Delete로 해제)"},
	"hk.none":       {"未设置", "Not set", "未設定", "未設定", "설정 안 함"},
	"hk.rec":        {"请按键…", "Press keys…", "請按鍵…", "キーを押して…", "키를 누르세요…"},
	"r.pos":         {"窗口位置", "Window position", "視窗位置", "ウィンドウ位置", "창 위치"},
	"r.pos.d":       {"呼出时窗口出现的位置", "Where the window appears", "呼出時視窗出現的位置", "表示される場所", "표시 위치"},
	"pos.cursor":    {"鼠标附近", "Near cursor", "滑鼠附近", "カーソル付近", "커서 근처"},
	"pos.center":    {"屏幕中央", "Screen center", "螢幕中央", "画面中央", "화면 중앙"},
	"pos.last":      {"记住位置", "Remember", "記住位置", "位置を記憶", "위치 기억"},
	"r.autopaste":   {"选中后自动粘贴", "Paste automatically", "選取後自動貼上", "選択後に自動で貼り付け", "선택 시 자동 붙여넣기"},
	"r.autopaste.d": {"关闭则只复制到剪贴板", "Off: only copy to the clipboard", "關閉則只複製到剪貼簿", "オフの場合はコピーのみ", "끄면 복사만 합니다"},
	"r.autostart":   {"开机自启动", "Launch at startup", "開機自動啟動", "スタートアップで起動", "시작 시 실행"},
	"r.autostart.d": {"静默驻留系统托盘", "Runs quietly in the tray", "靜默常駐系統匣", "トレイに常駐", "트레이에서 조용히 실행"},

	"r.glass":     {"毛玻璃效果", "Frosted glass", "毛玻璃效果", "すりガラス効果", "젖빛 유리 효과"},
	"r.glass.d":   {"呼出时模糊并透出桌面背景", "Blurs the desktop behind the window", "呼出時模糊並透出桌面背景", "背面のデスクトップをぼかします", "뒤의 바탕화면을 흐리게 합니다"},
	"r.blur":      {"模糊强度", "Blur strength", "模糊強度", "ぼかしの強さ", "흐림 강도"},
	"r.blur.d":    {"越高越朦胧", "Higher is hazier", "越高越朦朧", "高いほど強くぼけます", "높을수록 흐려집니다"},
	"lvl.low":     {"低", "Low", "低", "弱", "낮음"},
	"lvl.mid":     {"中", "Medium", "中", "中", "보통"},
	"lvl.high":    {"高", "High", "高", "強", "높음"},
	"r.theme":     {"外观主题", "Theme", "佈景主題", "テーマ", "테마"},
	"r.theme.d":   {"自动跟随 Windows 深浅色", "Follows Windows light/dark", "自動跟隨 Windows 深淺色", "Windows の設定に従います", "Windows 설정을 따릅니다"},
	"th.auto":     {"跟随系统", "System", "跟隨系統", "システム", "시스템"},
	"th.light":    {"浅色", "Light", "淺色", "ライト", "라이트"},
	"th.dark":     {"深色", "Dark", "深色", "ダーク", "다크"},
	"r.accent":    {"主题色", "Accent color", "主題色", "アクセントカラー", "강조 색"},
	"r.accent.d":  {"选中、开关与标签的颜色", "Selection, switches and tabs", "選取、開關與標籤的顏色", "選択・スイッチ・タブの色", "선택, 스위치, 탭 색상"},
	"r.scale":     {"界面大小", "Interface size", "介面大小", "表示サイズ", "인터페이스 크기"},
	"r.scale.d":   {"缩放整个窗口(修改后窗口会重新打开)", "Scales the whole window (reopens it)", "縮放整個視窗(修改後視窗會重新開啟)", "ウィンドウ全体を拡大縮小(再表示されます)", "창 전체 크기 조절(창이 다시 열립니다)"},
	"r.compact":   {"紧凑列表", "Compact list", "緊湊清單", "コンパクト表示", "간결한 목록"},
	"r.compact.d": {"每条只显示一行,一屏看更多", "One line per item, see more at once", "每筆只顯示一行,一屏看更多", "1 項目 1 行で表示", "항목당 한 줄로 표시"},
	"r.anim":      {"动画效果", "Animations", "動畫效果", "アニメーション", "애니메이션"},
	"r.anim.d":    {"淡入与列表滑入", "Fade-in and list slide", "淡入與清單滑入", "フェードとスライド", "페이드 및 슬라이드"},

	"r.sensitive":   {"敏感内容处理", "Sensitive content", "敏感內容處理", "機密情報の扱い", "민감 정보 처리"},
	"r.sensitive.d": {"识别 API 密钥、令牌、私钥、验证码、银行卡等", "Detects API keys, tokens, private keys, OTPs, card numbers…", "辨識 API 金鑰、權杖、私鑰、驗證碼、銀行卡等", "API キー・トークン・秘密鍵・認証コード・カード番号などを検出", "API 키, 토큰, 개인 키, 인증 코드, 카드 번호 등을 감지"},
	"r.pwd":         {"忽略密码管理器", "Ignore password managers", "忽略密碼管理員", "パスワード管理アプリを無視", "비밀번호 관리자 무시"},
	"r.pwd.d":       {"KeePass / 1Password / Bitwarden 等", "KeePass, 1Password, Bitwarden…", "KeePass / 1Password / Bitwarden 等", "KeePass / 1Password / Bitwarden など", "KeePass / 1Password / Bitwarden 등"},
	"r.ignore":      {"自定义忽略的应用", "Ignored apps", "自訂忽略的應用程式", "無視するアプリ", "무시할 앱"},
	"r.ignore.d":    {"在条目上右键可添加;点击清空列表", "Add via right-click on an item; click to clear", "在項目上按右鍵可加入;點擊清空清單", "項目を右クリックで追加。クリックで消去", "항목을 우클릭해 추가, 클릭하면 비웁니다"},
	"r.images":      {"记录图片", "Record images", "記錄圖片", "画像を記録", "이미지 기록"},
	"r.images.d":    {"截图与复制的图片也进入历史", "Screenshots and copied images too", "截圖與複製的圖片也進入歷史", "スクリーンショットなども履歴に保存", "스크린샷과 복사한 이미지도 저장"},
	"r.keep":        {"历史保留时间", "Keep history for", "歷史保留時間", "履歴の保存期間", "기록 보관 기간"},
	"r.keep.d":      {"置顶项不会被清理", "Pinned items are never removed", "置頂項不會被清理", "固定した項目は削除されません", "고정 항목은 삭제되지 않습니다"},
	"keep.7":        {"7 天", "7 days", "7 天", "7 日", "7일"},
	"keep.30":       {"30 天", "30 days", "30 天", "30 日", "30일"},
	"keep.90":       {"90 天", "90 days", "90 天", "90 日", "90일"},
	"keep.0":        {"永久", "Forever", "永久", "無期限", "영구"},
	"r.max":         {"最大记录条数", "Maximum items", "最大記錄筆數", "最大保存数", "최대 항목 수"},
	"r.max.d":       {"超出后自动清理最旧的", "Oldest are removed beyond this", "超出後自動清理最舊的", "超えると古いものから削除", "초과 시 오래된 항목부터 삭제"},
	"n.items":       {"%d 条", "%d", "%d 筆", "%d 件", "%d개"},
	"r.folder":      {"打开数据文件夹", "Open data folder", "開啟資料夾", "データフォルダーを開く", "데이터 폴더 열기"},
	"r.folder.d":    {"历史与设置的保存位置", "Where history and settings live", "歷史與設定的儲存位置", "履歴と設定の保存場所", "기록과 설정이 저장된 위치"},
	"b.open":        {"打开", "Open", "開啟", "開く", "열기"},
	"r.export":      {"导出文本历史", "Export text history", "匯出文字歷史", "テキスト履歴をエクスポート", "텍스트 기록 내보내기"},
	"r.export.d":    {"保存为 txt 文件并打开", "Saves a .txt file and opens it", "儲存為 txt 檔案並開啟", "txt ファイルに保存して開く", "txt 파일로 저장하고 엽니다"},
	"b.export":      {"导出", "Export", "匯出", "出力", "내보내기"},
	"r.clear":       {"清空历史记录", "Clear history", "清空歷史記錄", "履歴を消去", "기록 지우기"},
	"r.clear.d":     {"仅清除未置顶的内容", "Unpinned items only", "僅清除未置頂的內容", "固定以外を消去", "고정되지 않은 항목만 삭제"},
	"r.clear.c":     {"再点一次确认,此操作不可撤销", "Click again to confirm — can't be undone", "再點一次確認,此操作無法復原", "もう一度クリックで確定(元に戻せません)", "한 번 더 누르면 삭제됩니다(되돌릴 수 없음)"},
	"b.clear":       {"清空", "Clear", "清空", "消去", "삭제"},
	"b.reset":       {"清空", "Reset", "清空", "初期化", "초기화"},
	"about":         {"ClipGlass 1.6 · 完全离线运行,不上传任何数据", "ClipGlass 1.6 · Fully offline — nothing leaves your PC", "ClipGlass 1.6 · 完全離線運作,不上傳任何資料", "ClipGlass 1.6 · 完全オフライン動作、データは送信されません", "ClipGlass 1.6 · 완전 오프라인, 데이터를 전송하지 않습니다"},

	"m.paste":  {"粘贴", "Paste", "貼上", "貼り付け", "붙여넣기"},
	"m.copy":   {"仅复制", "Copy only", "僅複製", "コピーのみ", "복사만"},
	"m.plain":  {"粘贴为单行纯文本", "Paste as single line", "貼上為單行純文字", "1 行のテキストで貼り付け", "한 줄 텍스트로 붙여넣기"},
	"m.upper":  {"粘贴为大写", "Paste as UPPERCASE", "貼上為大寫", "大文字で貼り付け", "대문자로 붙여넣기"},
	"m.lower":  {"粘贴为小写", "Paste as lowercase", "貼上為小寫", "小文字で貼り付け", "소문자로 붙여넣기"},
	"m.open":   {"打开链接", "Open link", "開啟連結", "リンクを開く", "링크 열기"},
	"m.reveal": {"在资源管理器中显示", "Show in Explorer", "在檔案總管中顯示", "エクスプローラーで表示", "탐색기에서 보기"},
	"m.mail":   {"发送邮件", "Send email", "傳送郵件", "メールを送信", "이메일 보내기"},
	"m.pin":    {"置顶", "Pin", "置頂", "固定", "고정"},
	"m.unpin":  {"取消置顶", "Unpin", "取消置頂", "固定を解除", "고정 해제"},
	"m.ignore": {"忽略此应用的内容", "Ignore clips from this app", "忽略此應用程式的內容", "このアプリのコピーを無視", "이 앱의 내용 무시"},
	"m.delete": {"删除", "Delete", "刪除", "削除", "삭제"},

	"tray.show":  {"显示剪贴板", "Show clipboard", "顯示剪貼簿", "クリップボードを表示", "클립보드 표시"},
	"tray.pause": {"暂停记录", "Pause recording", "暫停記錄", "記録を一時停止", "기록 일시정지"},
	"tray.clear": {"清空历史(保留置顶)", "Clear history (keep pinned)", "清空歷史(保留置頂)", "履歴を消去(固定は保持)", "기록 지우기(고정 유지)"},
	"tray.exit":  {"退出", "Exit", "結束", "終了", "종료"},
	"tray.tip":   {"ClipGlass 剪贴板", "ClipGlass Clipboard", "ClipGlass 剪貼簿", "ClipGlass クリップボード", "ClipGlass 클립보드"},

	"time.now":   {"刚刚", "Just now", "剛剛", "たった今", "방금"},
	"time.s":     {"%d 秒前", "%ds ago", "%d 秒前", "%d 秒前", "%d초 전"},
	"time.m":     {"%d 分钟前", "%d min ago", "%d 分鐘前", "%d 分前", "%d분 전"},
	"time.y":     {"昨天 %s", "Yesterday %s", "昨天 %s", "昨日 %s", "어제 %s"},
	"time.md":    {"%[1]d月%[2]d日", "%[1]d/%[2]d", "%[1]d月%[2]d日", "%[1]d月%[2]d日", "%[1]d월 %[2]d일"},
	"meta.lines": {"%d 行", "%d lines", "%d 行", "%d 行", "%d줄"},
	"meta.chars": {"%d 字", "%d chars", "%d 字", "%d 文字", "%d자"},
	"img.label":  {"图片", "Image", "圖片", "画像", "이미지"},

	"src.explorer": {"资源管理器", "Explorer", "檔案總管", "エクスプローラー", "탐색기"},
	"src.notepad":  {"记事本", "Notepad", "記事本", "メモ帳", "메모장"},
	"src.terminal": {"终端", "Terminal", "終端機", "ターミナル", "터미널"},
	"src.cmd":      {"命令行", "Command Prompt", "命令提示字元", "コマンドプロンプト", "명령 프롬프트"},
	"src.wechat":   {"微信", "WeChat", "微信", "WeChat", "WeChat"},
	"chip.text":    {"文本", "Text", "文字", "テキスト", "텍스트"},
	"chip.color":   {"颜色", "Colors", "顏色", "色", "색상"},
	"chip.command": {"命令", "Commands", "指令", "コマンド", "명령어"},
	"chip.data":    {"数据", "Data", "資料", "データ", "데이터"},
	"chip.table":   {"表格", "Tables", "表格", "表", "표"},
	"chip.file":    {"文件", "Files", "檔案", "ファイル", "파일"},
	"chip.date":    {"日期", "Dates", "日期", "日付", "날짜"},
	"chip.network": {"网络", "Network", "網路", "ネットワーク", "네트워크"},
	"chip.id":      {"标识", "IDs", "識別碼", "ID", "식별자"},
	"chip.address": {"地址", "Addresses", "地址", "住所", "주소"},
	"chip.secret":  {"敏感", "Sensitive", "敏感", "機密", "민감"},
	"kt.command":   {"命", ">_", "令", "命", "명"},
	"kt.data":      {"{}", "{}", "{}", "{}", "{}"},
	"kt.table":     {"表", "Tbl", "表", "表", "표"},
	"kt.file":      {"文件", "File", "檔案", "ファ", "파일"},
	"kt.date":      {"日", "Dt", "日", "日", "날"},
	"kt.network":   {"IP", "IP", "IP", "IP", "IP"},
	"kt.id":        {"ID", "ID", "ID", "ID", "ID"},
	"kt.address":   {"址", "Ad", "址", "住", "주"},
	"kt.secret":    {"密", "Key", "密", "鍵", "키"},
	"meta.files":   {"%d 个文件", "%d files", "%d 個檔案", "%d 個のファイル", "파일 %d개"},

	"g.new":      {"新建分区", "New group", "新增分區", "新規グループ", "새 그룹"},
	"g.name.ph":  {"输入分区名称,回车确认(Esc 取消)", "Group name — Enter to save, Esc to cancel", "輸入分區名稱,Enter 確認(Esc 取消)", "グループ名を入力 — Enter で確定、Esc で取消", "그룹 이름 입력 — Enter 저장, Esc 취소"},
	"t.gcreated": {"已创建分区「%s」", "Group “%s” created", "已建立分區「%s」", "グループ「%s」を作成しました", "그룹 “%s”을(를) 만들었습니다"},
	"t.gdeleted": {"已删除分区「%s」", "Group “%s” deleted", "已刪除分區「%s」", "グループ「%s」を削除しました", "그룹 “%s”을(를) 삭제했습니다"},
	"t.moved":    {"已移动到「%s」", "Moved to “%s”", "已移動到「%s」", "「%s」に移動しました", "“%s”(으)로 이동했습니다"},
	"t.unmoved":  {"已移出分区", "Removed from group", "已移出分區", "グループから外しました", "그룹에서 제거했습니다"},
	"m.move":     {"移动到分区", "Move to group", "移動到分區", "グループへ移動", "그룹으로 이동"},
	"m.nogroup":  {"无分区", "No group", "無分區", "グループなし", "그룹 없음"},
	"m.newgroup": {"新建分区…", "New group…", "新增分區…", "新規グループ…", "새 그룹…"},
	"gm.rename":  {"重命名分区", "Rename group", "重新命名分區", "グループ名を変更", "그룹 이름 바꾸기"},
	"gm.color":   {"更换颜色", "Change color", "更換顏色", "色を変更", "색상 변경"},
	"gm.delete":  {"删除分区", "Delete group", "刪除分區", "グループを削除", "그룹 삭제"},

	"sens.skip": {"跳过不记录", "Skip", "跳過不記錄", "記録しない", "기록 안 함"},
	"sens.keep": {"遮罩暂存 10 分钟", "Mask for 10 min", "遮罩暫存 10 分鐘", "マスクして10分保持", "가려서 10분 보관"},
	"sens.off":  {"不过滤", "Off", "不過濾", "無効", "사용 안 함"},

	"sec.apikey":   {"API 密钥/令牌", "API key / token", "API 金鑰/權杖", "API キー/トークン", "API 키/토큰"},
	"sec.privkey":  {"私钥", "Private key", "私密金鑰", "秘密鍵", "개인 키"},
	"sec.jwt":      {"JWT 令牌", "JWT token", "JWT 權杖", "JWT トークン", "JWT 토큰"},
	"sec.password": {"密码/口令", "Password", "密碼/口令", "パスワード", "비밀번호"},
	"sec.card":     {"银行卡号", "Card number", "銀行卡號", "カード番号", "카드 번호"},
	"sec.idcard":   {"证件号码", "ID number", "證件號碼", "ID 番号", "신분증 번호"},
	"sec.otp":      {"验证码", "One-time code", "驗證碼", "認証コード", "인증 코드"},
	"sec.conn":     {"含密码的连接串", "Connection string", "含密碼的連線字串", "認証情報付き接続文字列", "비밀번호 포함 연결 문자열"},
	"sec.wallet":   {"钱包私钥", "Wallet key", "錢包私鑰", "ウォレット秘密鍵", "지갑 개인 키"},
	"sec.ssn":      {"社保号", "SSN", "社會安全號", "社会保障番号", "사회보장번호"},
	"sec.random":   {"疑似密码", "Possible password", "疑似密碼", "パスワードの可能性", "비밀번호로 추정"},
}
