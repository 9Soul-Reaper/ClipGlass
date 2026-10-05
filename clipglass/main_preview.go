//go:build !windows

package main

// 非 Windows 平台仅用于渲染预览图(文字用占位块代替),方便开发时检查布局与毛玻璃效果。

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"
)

type stubText struct{ c *Canvas }

func (s *stubText) w(str string, st TextStyle) int {
	n := 0.0
	for _, r := range str {
		if r < 128 {
			n += 0.55
		} else {
			n += 1.0
		}
	}
	return int(n * float64(st.Size))
}
func (s *stubText) Measure(str string, st TextStyle) int { return s.w(str, st) }
func (s *stubText) SetClip(r Rect)                       {}
func (s *stubText) Draw(str string, r Rect, st TextStyle, col Color, align, lines int) {
	w := s.w(str, st)
	maxLines := maxi(lines, 1)
	if w > r.W*maxLines {
		w = r.W * maxLines
	}
	h := int(float64(st.Size) * 0.62)
	for ln := 0; ln < maxLines && w > 0; ln++ {
		lw := minInt(w, r.W)
		w -= lw
		x := r.X
		switch align {
		case AlignCenter:
			x = r.X + (r.W-lw)/2
		case AlignRight:
			x = r.X + r.W - lw
		}
		y := r.Y + (r.H-h)/2
		if lines > 1 {
			y = r.Y + ln*int(float64(st.Size)*1.4) + (int(float64(st.Size)*1.4)-h)/2
		}
		s.c.FillRRect(float64(x), float64(y), float64(lw), float64(h), float64(h)/2, col, 0.85)
	}
}

func savePNG(c *Canvas, path string) {
	img := image.NewNRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			i := (y*c.W + x) * 4
			bg := 120.0
			if ((x/16)+(y/16))%2 == 0 {
				bg = 150
			}
			a := float64(c.Pix[i+3]) / 255
			mix := func(v byte) uint8 { return uint8(float64(v) + bg*(1-a)) }
			img.SetNRGBA(x, y, color.NRGBA{mix(c.Pix[i+2]), mix(c.Pix[i+1]), mix(c.Pix[i]), 255})
		}
	}
	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}

func fakeDesktop(w, h int) *Canvas {
	c := fallbackBackdrop(w, h, false)
	c.FillRRect(float64(w)*0.1, float64(h)*0.15, float64(w)*0.5, float64(h)*0.25, 12, Color{255, 120, 90}, 1)
	c.FillRRect(float64(w)*0.4, float64(h)*0.5, float64(w)*0.5, float64(h)*0.3, 12, Color{40, 160, 120}, 1)
	c.FillCircle(float64(w)*0.8, float64(h)*0.2, float64(w)*0.14, Color{250, 210, 60}, 1)
	for y := 0; y < h; y += 28 {
		c.Line(0, float64(y), float64(w), float64(y), 2, Color{255, 255, 255}, 0.5)
	}
	return c
}

func sampleImage() *image.NRGBA {
	im := image.NewNRGBA(image.Rect(0, 0, 320, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			i := y*im.Stride + x*4
			im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = byte(x*255/320), byte(y*255/200), 180, 255
		}
	}
	return im
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--mksyso" {
		if err := makeSyso("rsrc_windows_amd64.syso"); err != nil {
			panic(err)
		}
		fmt.Println("wrote rsrc_windows_amd64.syso")
		return
	}
	dir, _ := os.MkdirTemp("", "clipglass")
	st := &Store{Dir: dir}
	samples := []struct{ t, src string }{
		{"https://www.example.com/docs/getting-started?lang=zh-CN", "Chrome"},
		{"zhang.wei@example.com", "Outlook"},
		{"SELECT * FROM users WHERE created_at > '2026-01-01' ORDER BY id DESC;", "VS Code"},
		{"今天下午三点在会议室讨论新版本的发布计划,请大家提前准备好各自模块的进度汇报和风险清单。", "微信"},
		{"#4C8DFF", "Figma"},
		{"C:\\Users\\Public\\Documents\\季度报告.xlsx", "资源管理器"},
		{"2026100419384756", "Chrome"},
		{"138-0000-1234", "微信"},
		{"npm run build -- --mode production", "终端"},
		{"sk-proj-abcdefghijklmnopqrstuvwxyz123456", "Chrome"},
		{"2026-10-05 14:30", "Outlook"},
		{"192.168.1.100", "终端"},
	}
	for i := len(samples) - 1; i >= 0; i-- {
		it := st.Add(samples[i].t, samples[i].src)
		it.Time = time.Now().Add(-time.Duration(i*i*7) * time.Minute).Unix()
	}
	img := st.AddImage(sampleImage(), "Snipping Tool", "snippingtool.exe")
	img.Time = time.Now().Add(-3 * time.Minute).Unix()
	// 图片放到第 3 位
	st.Items = append([]*Item{st.Items[1], st.Items[2], st.Items[0]}, st.Items[3:]...)
	st.Items[3].Pinned = true
	st.AddGroup("工作")
	st.AddGroup("Projects")
	st.AddGroup("临时")
	for i, it := range st.Items {
		if i%4 == 1 {
			it.Group = 1
		} else if i%5 == 2 {
			it.Group = 2
		}
	}
	sec := st.AddFrom("sk-ant-api03-abcdefghijklmnopqrstuvwxyz0123", "Chrome", "chrome.exe", "sec.apikey")
	sec.Time = time.Now().Unix() - 120
	fl := st.Add("C:\\Users\\Public\\a.zip\nC:\\Users\\Public\\b.png", "资源管理器")
	fl.Kind, fl.lines = KFile, 2

	type shot struct {
		name    string
		dark    bool
		lang    string
		compact bool
		set     bool
		pop     int
	}
	shots := []shot{
		{"dark_list", true, "zh-CN", false, false, 0},
		{"light_list_en", false, "en", false, false, 0},
		{"dark_compact_ja", true, "ja", true, false, 0},
		{"light_settings_zh", false, "zh-CN", false, true, 0},
		{"dark_settings_en", true, "en", false, true, 0},
		{"dark_list_ru", true, "ru", false, false, 0},
		{"light_list_de", false, "de", false, false, 0},
		{"dark_pop_lang", true, "zh-CN", false, true, 1},
		{"light_pop_item", false, "zh-CN", false, false, 2},
	}
	for _, sh := range shots {
		set := defaultSettings()
		set.Compact = sh.compact
		ApplyLang(sh.lang, "en")
		ui := NewUI(1.5, st, set)
		ui.Dark = sh.dark
		ui.OnAction = func(*Item, int) {}
		ui.OnHide = func() {}
		ui.OnPauseToggle = func() {}
		ui.OnContext = func(*Item) {}
		ui.OnHotkey = func(int, HotkeySpec) bool { return true }
		in := ui.InnerRect()
		base, mask := ui.BuildBase(fakeDesktop(in.W, in.H))
		W, H := ui.WinSize()
		ui.SettingsOn = sh.set
		ui.Rebuild()
		ui.enterStart = time.Time{}
		ui.Hover = Hit{Act: ActItem, Arg: 1}
		ui.Sel = 1
		ui.MX, ui.MY = in.X+in.W-60, in.Y+260
		switch sh.pop {
		case 1:
			var it []PopItem
			it = append(it, PopItem{ID: 1, Label: "自动", Checked: true}, PopItem{Sep: true})
			for i, n := range langNames {
				it = append(it, PopItem{ID: 2 + i, Label: n})
			}
			ui.OpenPopup(it, nil)
			ui.Pop.Sel = 3
		case 2:
			ui.OpenPopup([]PopItem{{ID: 1, Label: "粘贴"}, {ID: 2, Label: "仅复制"}, {ID: 3, Label: "粘贴为大写"}, {Sep: true},
				{ID: 4, Label: "置顶"}, {ID: 5, Label: "移动到分区  ›"}, {Sep: true}, {ID: 6, Label: "删除", Danger: true}}, nil)
			ui.Pop.Sel = 1
		}
		c := NewCanvas(W, H)
		copy(c.Pix, base.Pix)
		ui.Draw(c, &stubText{c})
		for i := range mask {
			c.Pix[i*4+3] = mask[i]
		}
		name := "/tmp/claude-0/preview_" + sh.name + ".png"
		savePNG(c, name)
		fmt.Println("wrote", name)
	}
}
