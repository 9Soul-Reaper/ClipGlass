package main

import (
	"image"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := map[string]Kind{
		"https://www.example.com/docs?lang=zh-CN": KLink,
		"www.baidu.com":                       KLink,
		"zhang.wei@example.com":               KContact,
		"138-0000-1234":                       KContact,
		"021-8899-6677":                       KContact,
		"C:\\Users\\Public\\Documents\\a.txt": KPath,
		"/usr/local/bin":                      KPath,
		"2026100419384756":                    KNumber,
		"1,299.00":                            KNumber,
		"npm run build -- --mode production":  KCommand,
		"git commit -m \"fix\"":               KCommand,
		"SELECT * FROM users WHERE id > 1 ORDER BY id DESC;": KCode,
		"func main() {\n\tfmt.Println(1)\n}":                 KCode,
		"今天下午三点开会,请大家准时参加,谢谢。":                               KText,
		"Hello, nice to meet you.":                           KText,
		"请看这个链接 https://a.com/x 很有用":                         KText,
	}
	for in, want := range cases {
		if got := classify(in); got != want {
			t.Errorf("classify(%q)=%d want %d", in, got, want)
		}
	}
}

func TestSecrets(t *testing.T) {
	yes := []string{
		"4111 1111 1111 1111", "110101199003071234",
		"sk-proj-abcdefghijklmnopqrstuvwxyz123456",
		"ghp_abcdefghijklmnopqrstuvwxyz0123456789",
		"AKIAIOSFODNN7EXAMPLE",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIE",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abcdefghijk",
		"password=Hunter2abc", "API_KEY: abcd1234efgh",
		"您的验证码是 482913,5分钟内有效",
		"postgres://admin:s3cretpw@db.example.com:5432/app",
		"Authorization: Bearer abcdefghijklmnopqrstuvwxyz012345",
		"xK9#mP2$vL8@nQ4z",
	}
	for _, s := range yes {
		if secretReason(s) == "" {
			t.Errorf("should be sensitive: %q", s)
		}
	}
	no := []string{"hello", "138-0000-1234", "https://example.com/a?b=1", "2026100419384756",
		"今天下午三点开会", "zhang.wei@example.com", "#4C8DFF", "npm run build"}
	for _, s := range no {
		if r := secretReason(s); r != "" {
			t.Errorf("false positive %q -> %s", s, r)
		}
	}
}

func TestAccuracy(t *testing.T) {
	cases := map[string]Kind{
		"example.com/path": KLink, "localhost:3000": KLink, "README.md": KPath, "v1.2.3": KNumber, "50%": KNumber,
		"hsl(200,50%,50%)": KColor, "import os": KCode, "print('hi')": KCode, "SELECT 1": KCode,
		"a@x.com, b@y.org": KContact, "getUserProfileById2": KText, "你好": KText,
	}
	for in, want := range cases {
		if got := classify(in); got != want {
			t.Errorf("classify(%q)=%d want %d", in, got, want)
		}
	}
	for _, in := range []string{"getUserProfileById2", "MyClassName123", "hello_world_function_v2", "token = \"your_token_here\"",
		"API_KEY=${API_KEY}", "550e8400-e29b-41d4-a716-446655440000"} {
		if r := secretReason(in); r != "" {
			t.Errorf("false positive %q -> %s", in, r)
		}
	}
}

func TestClassifyMore(t *testing.T) {
	cases := map[string]Kind{
		"2026-10-05": KDate, "2026年10月5日": KDate, "192.168.1.1": KNetwork, "00:1A:2B:3C:4D:5E": KNetwork,
		"example.com": KNetwork, "123e4567-e89b-12d3-a456-426614174000": KID,
		"d41d8cd98f00b204e9800998ecf8427e": KID, `{"a":1,"b":[1,2]}`: KData,
		"a\tb\tc\n1\t2\t3": KTable, "广东省深圳市南山区科技园路1号": KAddress,
	}
	for in, want := range cases {
		if got := classify(in); got != want {
			t.Errorf("classify(%q)=%d want %d", in, got, want)
		}
	}
}

func TestI18N(t *testing.T) {
	for k, v := range strs {
		for i, x := range v {
			if x == "" {
				t.Errorf("empty %s[%d]", k, i)
			}
		}
	}
	for li, m := range moreLangs {
		for k := range m {
			if _, ok := strs[k]; !ok {
				t.Errorf("lang %d unknown key %s", li, k)
			}
		}
	}
	for k, v := range strs {
		n := strings.Count(v[1], "%")
		for li, m := range moreLangs {
			if x, ok := m[k]; ok && strings.Count(x, "%") != n {
				t.Errorf("lang %d key %s verb mismatch", li, k)
			}
		}
	}
}

func TestDIB(t *testing.T) {
	im := sampleImage2()
	back := parseDIB(buildDIB(im))
	if back == nil || back.Bounds() != im.Bounds() || back.Pix[0] != im.Pix[0] {
		t.Fatal("dib roundtrip")
	}
}

func sampleImage2() *image.NRGBA {
	im := image.NewNRGBA(image.Rect(0, 0, 7, 5))
	for i := range im.Pix {
		im.Pix[i] = byte(i*7 + 3)
		if i%4 == 3 {
			im.Pix[i] = 255
		}
	}
	return im
}

func TestStoreSearch(t *testing.T) {
	s := &Store{}
	s.Add("alpha beta", "Chrome")
	s.Add("gamma", "微信")
	s.Add("alpha beta", "") // 去重并置顶
	if len(s.Items) != 2 || s.Items[0].Text != "alpha beta" {
		t.Fatal("dedupe/bump failed")
	}
	if n := len(s.View(0, "ALPHA  chrome")); n != 1 {
		t.Fatalf("search got %d", n)
	}
	s.TogglePin(s.Items[1].ID)
	if v := s.View(0, ""); v[0].Text != "gamma" {
		t.Fatal("pinned first failed")
	}
	if len(s.View(FilterKindBase+int(KImage), ""))+len(s.View(FilterPinned, "")) != 1 {
		t.Fatal("pinned filter")
		s.AddGroup("工作")
		s.SetItemGroup(s.Items[0].ID, 1)
		if len(s.View(FilterGroupBase+1, "")) != 1 {
			t.Fatal("group filter")
		}
	}
}
