package main

// v1.6 新增文案(加密保存、完整备份、首次自动粘贴提示)。

func init() {
	for k, v := range map[string][5]string{
		"r.encrypt":      {"加密保存历史", "Encrypt saved history", "加密儲存歷史", "履歴を暗号化して保存", "기록 암호화 저장"},
		"r.encrypt.d":    {"用 Windows 账户加密历史与图片;换账户或重装后无法读取", "Encrypts history and images with your Windows account; unreadable on other accounts or PCs", "以 Windows 帳戶加密歷史與圖片;換帳戶或重裝後無法讀取", "Windowsアカウントで履歴と画像を暗号化(別アカウント/PCでは読めません)", "Windows 계정으로 기록과 이미지를 암호화(다른 계정/PC에서는 읽을 수 없음)"},
		"r.backup":       {"完整备份", "Full backup", "完整備份", "完全バックアップ", "전체 백업"},
		"r.backup.d":     {"导出历史、图片和分区为 zip(未加密,请妥善保管)", "Exports history, images and groups to a zip (unencrypted — keep it safe)", "匯出歷史、圖片與分區為 zip(未加密,請妥善保管)", "履歴・画像・グループをzipに出力(暗号化なし。保管に注意)", "기록·이미지·그룹을 zip으로 내보내기(암호화 안 됨, 안전하게 보관)"},
		"r.restore":      {"导入备份", "Import backup", "匯入備份", "バックアップを読み込む", "백업 가져오기"},
		"r.restore.d":    {"把备份 zip 命名为 import.zip 放进数据文件夹后点击导入(合并,不覆盖)", "Put the backup zip in the data folder as import.zip, then click Import (merges, never overwrites)", "把備份 zip 命名為 import.zip 放進資料夾後點擊匯入(合併,不覆蓋)", "バックアップzipを import.zip の名前でデータフォルダーに置いて実行(統合・上書きなし)", "백업 zip을 import.zip 이름으로 데이터 폴더에 넣고 가져오기(병합, 덮어쓰지 않음)"},
		"b.import":       {"导入", "Import", "匯入", "読み込む", "가져오기"},
		"t.backup":       {"已导出备份并打开所在位置", "Backup exported — opening its folder", "已匯出備份並開啟所在位置", "バックアップを出力しました", "백업을 내보냈습니다"},
		"t.restore.ask":  {"将导入 %d 条新记录,再点一次确认", "%d new items will be imported — click again to confirm", "將匯入 %d 筆新記錄,再點一次確認", "新規 %d 件を読み込みます。もう一度クリックで確定", "새 항목 %d개를 가져옵니다. 한 번 더 누르면 확정"},
		"t.restore.done": {"已导入 %d 条记录", "Imported %d items", "已匯入 %d 筆記錄", "%d 件を読み込みました", "%d개 항목을 가져왔습니다"},
		"t.restore.none": {"数据文件夹里没有可用的 import.zip", "No valid import.zip in the data folder", "資料夾中沒有可用的 import.zip", "データフォルダーに有効な import.zip がありません", "데이터 폴더에 유효한 import.zip이 없습니다"},
		"t.autopaste":    {"提示:选择条目后会自动按 Ctrl+V 粘贴,可在设置中关闭", "Tip: choosing an item auto-presses Ctrl+V — turn it off in Settings", "提示:選擇項目後會自動按 Ctrl+V 貼上,可在設定中關閉", "ヒント:選択すると自動で Ctrl+V を送信します(設定でオフにできます)", "팁: 항목을 선택하면 자동으로 Ctrl+V를 누릅니다(설정에서 끌 수 있음)"},
	} {
		strs[k] = v
	}
	extra := [9]map[string]string{
		kv("r.encrypt", "Cifrar el historial", "r.encrypt.d", "Cifra historial e imágenes con tu cuenta de Windows; ilegible en otras cuentas o PC", "r.backup", "Copia completa", "r.backup.d", "Exporta historial, imágenes y grupos a un zip (sin cifrar; guárdalo bien)",
			"r.restore", "Importar copia", "r.restore.d", "Pon el zip en la carpeta de datos como import.zip y pulsa Importar (fusiona, nunca sobrescribe)", "b.import", "Importar", "t.backup", "Copia exportada; abriendo su carpeta",
			"t.restore.ask", "Se importarán %d elementos nuevos; pulsa otra vez para confirmar", "t.restore.done", "%d elementos importados", "t.restore.none", "No hay un import.zip válido en la carpeta de datos",
			"t.autopaste", "Consejo: al elegir un elemento se pulsa Ctrl+V solo; desactívalo en Ajustes"),
		kv("r.encrypt", "Chiffrer l'historique", "r.encrypt.d", "Chiffre historique et images avec votre compte Windows ; illisible sur un autre compte ou PC", "r.backup", "Sauvegarde complète", "r.backup.d", "Exporte historique, images et groupes dans un zip (non chiffré — à garder en lieu sûr)",
			"r.restore", "Importer une sauvegarde", "r.restore.d", "Placez le zip dans le dossier de données sous le nom import.zip, puis cliquez sur Importer (fusion, sans écrasement)", "b.import", "Importer", "t.backup", "Sauvegarde exportée ; ouverture du dossier",
			"t.restore.ask", "%d nouveaux éléments seront importés — cliquez encore pour confirmer", "t.restore.done", "%d éléments importés", "t.restore.none", "Aucun import.zip valide dans le dossier de données",
			"t.autopaste", "Astuce : choisir un élément envoie Ctrl+V automatiquement ; désactivable dans les Réglages"),
		kv("r.encrypt", "Verlauf verschlüsseln", "r.encrypt.d", "Verschlüsselt Verlauf und Bilder mit deinem Windows-Konto; auf anderen Konten/PCs nicht lesbar", "r.backup", "Vollständige Sicherung", "r.backup.d", "Exportiert Verlauf, Bilder und Gruppen als ZIP (unverschlüsselt — sicher aufbewahren)",
			"r.restore", "Sicherung importieren", "r.restore.d", "ZIP als import.zip in den Datenordner legen und auf Importieren klicken (zusammenführen, nie überschreiben)", "b.import", "Importieren", "t.backup", "Sicherung exportiert — Ordner wird geöffnet",
			"t.restore.ask", "%d neue Einträge werden importiert — zum Bestätigen erneut klicken", "t.restore.done", "%d Einträge importiert", "t.restore.none", "Keine gültige import.zip im Datenordner",
			"t.autopaste", "Tipp: Beim Auswählen wird automatisch Strg+V gesendet — in den Einstellungen abschaltbar"),
		kv("r.encrypt", "Criptografar histórico", "r.encrypt.d", "Criptografa histórico e imagens com sua conta do Windows; ilegível em outras contas ou PCs", "r.backup", "Backup completo", "r.backup.d", "Exporta histórico, imagens e grupos em um zip (sem criptografia — guarde com cuidado)",
			"r.restore", "Importar backup", "r.restore.d", "Coloque o zip na pasta de dados como import.zip e clique em Importar (mescla, nunca sobrescreve)", "b.import", "Importar", "t.backup", "Backup exportado; abrindo a pasta",
			"t.restore.ask", "%d novos itens serão importados — clique de novo para confirmar", "t.restore.done", "%d itens importados", "t.restore.none", "Não há import.zip válido na pasta de dados",
			"t.autopaste", "Dica: ao escolher um item, Ctrl+V é enviado automaticamente; desative em Configurações"),
		kv("r.encrypt", "Шифровать историю", "r.encrypt.d", "Шифрует историю и изображения учётной записью Windows; на другой записи или ПК не читается", "r.backup", "Полная копия", "r.backup.d", "Экспорт истории, изображений и групп в zip (без шифрования — храните аккуратно)",
			"r.restore", "Импорт копии", "r.restore.d", "Положите zip в папку данных под именем import.zip и нажмите «Импорт» (слияние, без перезаписи)", "b.import", "Импорт", "t.backup", "Копия экспортирована; открываю папку",
			"t.restore.ask", "Будет импортировано новых записей: %d — нажмите ещё раз для подтверждения", "t.restore.done", "Импортировано записей: %d", "t.restore.none", "В папке данных нет подходящего import.zip",
			"t.autopaste", "Совет: после выбора записи автоматически нажимается Ctrl+V — отключается в настройках"),
		kv("r.encrypt", "Cifra cronologia", "r.encrypt.d", "Cifra cronologia e immagini con il tuo account Windows; illeggibile su altri account o PC", "r.backup", "Backup completo", "r.backup.d", "Esporta cronologia, immagini e gruppi in uno zip (non cifrato — conservalo con cura)",
			"r.restore", "Importa backup", "r.restore.d", "Metti lo zip nella cartella dati come import.zip e fai clic su Importa (unisce, non sovrascrive)", "b.import", "Importa", "t.backup", "Backup esportato; apro la cartella",
			"t.restore.ask", "Verranno importati %d nuovi elementi — clicca ancora per confermare", "t.restore.done", "%d elementi importati", "t.restore.none", "Nessun import.zip valido nella cartella dati",
			"t.autopaste", "Suggerimento: scegliendo un elemento viene premuto Ctrl+V da solo; disattivabile nelle Impostazioni"),
		kv("r.encrypt", "Geçmişi şifrele", "r.encrypt.d", "Geçmişi ve görselleri Windows hesabınla şifreler; başka hesap/PC'de okunamaz", "r.backup", "Tam yedek", "r.backup.d", "Geçmiş, görsel ve grupları zip olarak dışa aktarır (şifresiz — güvenle saklayın)",
			"r.restore", "Yedeği içe aktar", "r.restore.d", "Zip'i veri klasörüne import.zip adıyla koyup İçe aktar'a tıklayın (birleştirir, üzerine yazmaz)", "b.import", "İçe aktar", "t.backup", "Yedek dışa aktarıldı; klasör açılıyor",
			"t.restore.ask", "%d yeni öğe içe aktarılacak — onaylamak için tekrar tıklayın", "t.restore.done", "%d öğe içe aktarıldı", "t.restore.none", "Veri klasöründe geçerli import.zip yok",
			"t.autopaste", "İpucu: bir öğe seçince Ctrl+V otomatik basılır; Ayarlar'dan kapatabilirsiniz"),
		kv("r.encrypt", "Mã hóa lịch sử", "r.encrypt.d", "Mã hóa lịch sử và ảnh bằng tài khoản Windows; không đọc được trên tài khoản/máy khác", "r.backup", "Sao lưu đầy đủ", "r.backup.d", "Xuất lịch sử, ảnh và nhóm ra zip (không mã hóa — hãy giữ cẩn thận)",
			"r.restore", "Nhập bản sao lưu", "r.restore.d", "Đặt zip vào thư mục dữ liệu với tên import.zip rồi bấm Nhập (gộp, không ghi đè)", "b.import", "Nhập", "t.backup", "Đã xuất bản sao lưu; đang mở thư mục",
			"t.restore.ask", "Sẽ nhập %d mục mới — bấm lần nữa để xác nhận", "t.restore.done", "Đã nhập %d mục", "t.restore.none", "Không có import.zip hợp lệ trong thư mục dữ liệu",
			"t.autopaste", "Mẹo: chọn một mục sẽ tự nhấn Ctrl+V; có thể tắt trong Cài đặt"),
		kv("r.encrypt", "Enkripsi riwayat", "r.encrypt.d", "Mengenkripsi riwayat dan gambar dengan akun Windows; tidak terbaca di akun/PC lain", "r.backup", "Cadangan penuh", "r.backup.d", "Ekspor riwayat, gambar, dan grup ke zip (tidak terenkripsi — simpan dengan aman)",
			"r.restore", "Impor cadangan", "r.restore.d", "Taruh zip di folder data bernama import.zip lalu klik Impor (menggabungkan, tidak menimpa)", "b.import", "Impor", "t.backup", "Cadangan diekspor; membuka folder",
			"t.restore.ask", "%d item baru akan diimpor — klik lagi untuk konfirmasi", "t.restore.done", "%d item diimpor", "t.restore.none", "Tidak ada import.zip valid di folder data",
			"t.autopaste", "Tips: memilih item otomatis menekan Ctrl+V; bisa dimatikan di Pengaturan"),
	}
	for i, m := range extra {
		for k, v := range m {
			moreLangs[i][k] = v
		}
	}
}
