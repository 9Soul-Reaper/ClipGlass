package main

// v1.3 新增文案(悬浮球、玻璃浓度、钉住窗口等)。

func init() {
	for k, v := range map[string][5]string{
		"r.opacity":    {"玻璃浓度", "Glass density", "玻璃濃度", "ガラスの濃度", "유리 농도"},
		"r.opacity.d":  {"越浓文字越清晰,越淡越通透", "Denser = clearer text, lighter = more see-through", "越濃文字越清晰,越淡越通透", "濃いほど文字がくっきり、薄いほど透明", "진할수록 글자가 또렷하고 옅을수록 투명해요"},
		"op.0":         {"通透", "Airy", "通透", "すっきり", "투명"},
		"op.1":         {"均衡", "Balanced", "均衡", "標準", "균형"},
		"op.2":         {"清晰", "Dense", "清晰", "くっきり", "선명"},
		"r.ball":       {"悬浮球", "Floating ball", "懸浮球", "フローティングボール", "플로팅 버튼"},
		"r.ball.d":     {"桌面悬浮按钮:点击呼出,可拖动,右键打开菜单", "Draggable desktop button: click to open, right-click for menu", "桌面懸浮按鈕:點擊呼出,可拖動,右鍵開啟選單", "デスクトップのボタン:クリックで表示、ドラッグ移動、右クリックでメニュー", "바탕화면 버튼: 클릭해 열기, 드래그 이동, 우클릭 메뉴"},
		"r.keepopen":   {"钉住窗口", "Keep window open", "釘住視窗", "ウィンドウを固定", "창 고정"},
		"r.keepopen.d": {"切换应用或粘贴后不自动收起(Esc 或 × 关闭)", "Stays open after pasting or switching apps (Esc or × closes)", "切換應用程式或貼上後不自動收起(Esc 或 × 關閉)", "貼り付けやアプリ切替後も閉じません(Esc か × で閉じる)", "붙여넣기나 앱 전환 후에도 열려 있어요(Esc 또는 ×로 닫기)"},
		"t.copied":     {"已复制", "Copied", "已複製", "コピーしました", "복사했습니다"},
		"t.pinon":      {"窗口已钉住", "Window pinned", "視窗已釘住", "ウィンドウを固定しました", "창을 고정했습니다"},
		"t.pinoff":     {"已取消钉住", "Window unpinned", "已取消釘住", "固定を解除しました", "고정을 해제했습니다"},
	} {
		strs[k] = v
	}
	extra := [9]map[string]string{
		kv("r.opacity", "Densidad del cristal", "r.opacity.d", "Más denso = texto más nítido; más claro = más transparente", "op.0", "Transparente", "op.1", "Equilibrado", "op.2", "Nítido",
			"r.ball", "Botón flotante", "r.ball.d", "Botón en el escritorio: clic para abrir, arrastrar, clic derecho para el menú", "r.keepopen", "Mantener ventana abierta",
			"r.keepopen.d", "No se cierra al pegar ni al cambiar de app (Esc o × para cerrar)", "t.copied", "Copiado", "t.pinon", "Ventana fijada", "t.pinoff", "Ventana liberada"),
		kv("r.opacity", "Densité du verre", "r.opacity.d", "Plus dense = texte plus net ; plus clair = plus transparent", "op.0", "Transparent", "op.1", "Équilibré", "op.2", "Net",
			"r.ball", "Bouton flottant", "r.ball.d", "Bouton sur le bureau : clic pour ouvrir, glisser, clic droit pour le menu", "r.keepopen", "Garder la fenêtre ouverte",
			"r.keepopen.d", "Reste ouverte après le collage ou le changement d'app (Échap ou × pour fermer)", "t.copied", "Copié", "t.pinon", "Fenêtre épinglée", "t.pinoff", "Fenêtre libérée"),
		kv("r.opacity", "Glasdichte", "r.opacity.d", "Dichter = klarerer Text, heller = durchsichtiger", "op.0", "Durchsichtig", "op.1", "Ausgewogen", "op.2", "Klar",
			"r.ball", "Schwebende Kugel", "r.ball.d", "Desktop-Button: Klick öffnet, ziehbar, Rechtsklick für Menü", "r.keepopen", "Fenster offen halten",
			"r.keepopen.d", "Bleibt nach dem Einfügen oder App-Wechsel offen (Esc oder × schließt)", "t.copied", "Kopiert", "t.pinon", "Fenster angeheftet", "t.pinoff", "Fenster gelöst"),
		kv("r.opacity", "Densidade do vidro", "r.opacity.d", "Mais denso = texto mais nítido; mais leve = mais transparente", "op.0", "Transparente", "op.1", "Equilibrado", "op.2", "Nítido",
			"r.ball", "Botão flutuante", "r.ball.d", "Botão na área de trabalho: clique para abrir, arraste, clique direito para o menu", "r.keepopen", "Manter janela aberta",
			"r.keepopen.d", "Fica aberta após colar ou trocar de app (Esc ou × fecha)", "t.copied", "Copiado", "t.pinon", "Janela fixada", "t.pinoff", "Janela solta"),
		kv("r.opacity", "Плотность стекла", "r.opacity.d", "Плотнее — чётче текст, легче — прозрачнее", "op.0", "Прозрачно", "op.1", "Баланс", "op.2", "Чётко",
			"r.ball", "Плавающая кнопка", "r.ball.d", "Кнопка на рабочем столе: клик открывает, можно перетаскивать, правый клик — меню", "r.keepopen", "Не закрывать окно",
			"r.keepopen.d", "Остаётся открытым после вставки и смены приложения (Esc или × закрывает)", "t.copied", "Скопировано", "t.pinon", "Окно закреплено", "t.pinoff", "Окно откреплено"),
		kv("r.opacity", "Densità del vetro", "r.opacity.d", "Più denso = testo più nitido; più leggero = più trasparente", "op.0", "Trasparente", "op.1", "Equilibrato", "op.2", "Nitido",
			"r.ball", "Pulsante flottante", "r.ball.d", "Pulsante sul desktop: clic per aprire, trascina, clic destro per il menu", "r.keepopen", "Mantieni finestra aperta",
			"r.keepopen.d", "Resta aperta dopo l'incolla o il cambio di app (Esc o × per chiudere)", "t.copied", "Copiato", "t.pinon", "Finestra fissata", "t.pinoff", "Finestra sbloccata"),
		kv("r.opacity", "Cam yoğunluğu", "r.opacity.d", "Yoğun = net yazı, hafif = daha saydam", "op.0", "Saydam", "op.1", "Dengeli", "op.2", "Net",
			"r.ball", "Yüzen düğme", "r.ball.d", "Masaüstü düğmesi: tıkla aç, sürükle, sağ tık menü", "r.keepopen", "Pencereyi açık tut",
			"r.keepopen.d", "Yapıştırınca veya uygulama değişince açık kalır (Esc veya × kapatır)", "t.copied", "Kopyalandı", "t.pinon", "Pencere sabitlendi", "t.pinoff", "Sabitleme kaldırıldı"),
		kv("r.opacity", "Độ đậm của kính", "r.opacity.d", "Đậm hơn = chữ rõ hơn, nhạt hơn = trong suốt hơn", "op.0", "Trong suốt", "op.1", "Cân bằng", "op.2", "Rõ nét",
			"r.ball", "Nút nổi", "r.ball.d", "Nút trên màn hình: bấm để mở, kéo để di chuyển, chuột phải mở menu", "r.keepopen", "Giữ cửa sổ mở",
			"r.keepopen.d", "Không tự đóng sau khi dán hoặc đổi ứng dụng (Esc hoặc × để đóng)", "t.copied", "Đã sao chép", "t.pinon", "Đã ghim cửa sổ", "t.pinoff", "Đã bỏ ghim"),
		kv("r.opacity", "Kepadatan kaca", "r.opacity.d", "Lebih padat = teks lebih jelas, lebih tipis = lebih transparan", "op.0", "Transparan", "op.1", "Seimbang", "op.2", "Jelas",
			"r.ball", "Tombol melayang", "r.ball.d", "Tombol di desktop: klik untuk buka, seret, klik kanan untuk menu", "r.keepopen", "Biarkan jendela terbuka",
			"r.keepopen.d", "Tetap terbuka setelah menempel atau ganti aplikasi (Esc atau × untuk menutup)", "t.copied", "Disalin", "t.pinon", "Jendela disematkan", "t.pinoff", "Sematan dilepas"),
	}
	for i, m := range extra {
		for k, v := range m {
			moreLangs[i][k] = v
		}
	}
}
