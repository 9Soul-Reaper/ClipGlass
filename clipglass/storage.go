package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---------- 静态加密(Windows 上由 DPAPI 实现) ----------

var protectFn, unprotectFn func([]byte) ([]byte, error)

const magicEnc = "CGENC1"

// seal 按设置加密写入内容;不可用时(非 Windows)原样返回。
func (s *Store) seal(b []byte) []byte {
	if s.Encrypt && protectFn != nil {
		if e, err := protectFn(b); err == nil {
			return append([]byte(magicEnc), e...)
		}
	}
	return b
}

func openSealed(b []byte) ([]byte, error) {
	if bytes.HasPrefix(b, []byte(magicEnc)) {
		if unprotectFn == nil {
			return nil, errors.New("encrypted data is not readable on this platform")
		}
		return unprotectFn(b[len(magicEnc):])
	}
	return b, nil
}

func (s *Store) readBlob(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return openSealed(b)
}

func (s *Store) writeBlob(path string, b []byte) error {
	return os.WriteFile(path, s.seal(b), 0o600)
}

// writeWithBackup 原子写入,并把上一版本保留为 .bak,用于损坏恢复 / 回滚。
func writeWithBackup(path string, b []byte) bool {
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) != nil {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		_ = os.Remove(path + ".bak")
		_ = os.Rename(path, path+".bak")
	}
	return os.Rename(tmp, path) == nil
}

// ---------- 历史文件格式(带版本号) ----------

const storeVersion = 1

type historyFile struct {
	V     int     `json:"v"`
	Items []*Item `json:"items"`
}

func (s *Store) marshalHistory() ([]byte, error) {
	disk := make([]*Item, 0, len(s.Items))
	for _, it := range s.Items {
		if !it.Mem { // 敏感内容永不落盘
			disk = append(disk, it)
		}
	}
	return json.Marshal(historyFile{V: storeVersion, Items: disk})
}

// parseHistory 同时兼容 v1 对象格式与旧版的纯数组格式。
func parseHistory(b []byte) ([]*Item, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil, errors.New("empty")
	}
	if b[0] == '[' {
		var items []*Item
		err := json.Unmarshal(b, &items)
		return items, err
	}
	var h historyFile
	if err := json.Unmarshal(b, &h); err != nil {
		return nil, err
	}
	if h.V > storeVersion {
		return nil, fmt.Errorf("history version %d is newer than supported", h.V)
	}
	return h.Items, nil
}

// loadHistory 读取历史:主文件损坏时保留损坏副本,并回退到 .bak。
func (s *Store) loadHistory() {
	path := filepath.Join(s.Dir, "history.json")
	for i, p := range []string{path, path + ".bak"} {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		plain, err := openSealed(raw)
		var items []*Item
		if err == nil {
			items, err = parseHistory(plain)
		}
		if err == nil {
			s.Items = items
			s.Recovered = i == 1
			return
		}
		if i == 0 { // 主文件坏了:改名保留,便于手工找回
			_ = os.Rename(p, filepath.Join(s.Dir, fmt.Sprintf("history.corrupt-%d.json", time.Now().Unix())))
		}
	}
}

// Reseal 切换加密设置后,重写历史、图片与缩略图。
func (s *Store) Reseal() {
	s.dirty = true
	for _, it := range s.Items {
		if it.Kind != KImage || it.Img == "" {
			continue
		}
		for _, p := range []string{s.imgPath(it.Img), s.thumbPath(it.Img)} {
			if b, err := s.readBlob(p); err == nil {
				_ = s.writeBlob(p, b)
			}
		}
	}
	s.purgeBak = true
	s.Save()
}

// RemoveText 删除与 text 完全相同的非敏感历史条目(识别为敏感后清理旧明文)。
func (s *Store) RemoveText(text string) {
	keep := s.Items[:0]
	for _, it := range s.Items {
		if it.Kind != KImage && it.Text == text && !it.Mem {
			s.dirty, s.purgeBak = true, true
			continue
		}
		keep = append(keep, it)
	}
	s.Items = keep
}

// ---------- 完整备份 / 导入 ----------

var reBackupName = regexp.MustCompile(`^(history\.json|groups\.json|manifest\.json|images/[0-9a-f]{16}(_t)?\.png)$`)

const maxBackupEntry = 128 << 20

// ExportBackup 把历史、分区、设置与图片打包为 zip(内容为明文)。
func (s *Store) ExportBackup(set *Settings) (string, error) {
	s.Save()
	name := filepath.Join(s.Dir, "ClipGlass-backup-"+time.Now().Format("20060102-150405")+".zip")
	f, err := os.Create(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	add := func(n string, b []byte) error {
		w, err := zw.Create(n)
		if err == nil {
			_, err = w.Write(b)
		}
		return err
	}
	hist, _ := s.marshalHistory()
	grp, _ := json.Marshal(s.Groups)
	setb, _ := json.Marshal(set)
	for n, b := range map[string][]byte{"manifest.json": []byte(`{"v":1,"app":"ClipGlass"}`), "history.json": hist, "groups.json": grp, "settings.json": setb} {
		if err := add(n, b); err != nil {
			return "", err
		}
	}
	for _, it := range s.Items {
		if it.Kind != KImage || it.Img == "" || it.Mem {
			continue
		}
		for _, p := range []string{s.imgPath(it.Img), s.thumbPath(it.Img)} {
			if b, err := s.readBlob(p); err == nil {
				if err := add("images/"+filepath.Base(p), b); err != nil {
					return "", err
				}
			}
		}
	}
	return name, zw.Close()
}

type importPlan struct {
	items  []*Item
	groups []Group
	files  map[string][]byte
}

func readBackup(path string) (*importPlan, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	p := &importPlan{files: map[string][]byte{}}
	var total int64
	for _, f := range zr.File {
		if !reBackupName.MatchString(f.Name) || f.UncompressedSize64 > maxBackupEntry {
			continue
		}
		total += int64(f.UncompressedSize64)
		if total > 1<<30 {
			return nil, errors.New("backup too large")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, maxBackupEntry))
		rc.Close()
		if err != nil {
			return nil, err
		}
		switch f.Name {
		case "history.json":
			if p.items, err = parseHistory(b); err != nil {
				return nil, err
			}
		case "groups.json":
			_ = json.Unmarshal(b, &p.groups)
		default:
			p.files[f.Name] = b
		}
	}
	if p.items == nil {
		return nil, errors.New("no history in backup")
	}
	return p, nil
}

func (s *Store) isDup(it *Item) bool {
	for _, x := range s.Items {
		if it.Kind == KImage && x.Kind == KImage && x.Img == it.Img {
			return true
		}
		if it.Kind != KImage && x.Kind != KImage && x.Text == it.Text {
			return true
		}
	}
	return false
}

// PreviewImport 返回 import.zip 中将被导入的新条目数(不修改任何数据)。
func (s *Store) PreviewImport() (int, error) {
	p, err := readBackup(filepath.Join(s.Dir, "import.zip"))
	if err != nil {
		return 0, err
	}
	n := 0
	for _, it := range p.items {
		if !s.isDup(it) {
			n++
		}
	}
	return n, nil
}

// ApplyImport 合并导入:已存在的内容跳过,分区按名称合并,不覆盖现有数据。
func (s *Store) ApplyImport() (int, error) {
	p, err := readBackup(filepath.Join(s.Dir, "import.zip"))
	if err != nil {
		return 0, err
	}
	gmap := map[int]int{}
	for _, g := range p.groups {
		found := 0
		for _, x := range s.Groups {
			if strings.EqualFold(x.Name, g.Name) {
				found = x.ID
			}
		}
		if found == 0 {
			if ng := s.AddGroup(g.Name); ng != nil {
				found = ng.ID
			}
		}
		gmap[g.ID] = found
	}
	_ = os.MkdirAll(filepath.Join(s.Dir, "images"), 0o755)
	n := 0
	var added []*Item
	for _, it := range p.items {
		if s.isDup(it) {
			continue
		}
		if it.Kind == KImage {
			if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(it.Img) {
				continue
			}
			body, ok := p.files["images/"+it.Img+".png"]
			if !ok {
				continue
			}
			if s.writeBlob(s.imgPath(it.Img), body) != nil {
				continue
			}
			if th, ok := p.files["images/"+it.Img+"_t.png"]; ok {
				_ = s.writeBlob(s.thumbPath(it.Img), th)
			}
		}
		s.nextID++
		it.ID = s.nextID
		it.Group = gmap[it.Group]
		it.Mem = false
		it.prepare()
		added = append(added, it)
		n++
	}
	s.Items = append(s.Items, added...)
	sort.SliceStable(s.Items, func(i, j int) bool { return s.Items[i].Time > s.Items[j].Time })
	s.dirty = true
	s.Save()
	return n, nil
}
