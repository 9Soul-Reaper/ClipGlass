package main

import (
	"encoding/json"
	"math"
	"regexp"
	"strings"
	"unicode"
)

// ---------- 敏感信息识别 ----------
// secretReason 返回敏感类型对应的 i18n key,不敏感则返回 ""。

var secretPatterns = []struct {
	key string
	re  *regexp.Regexp
}{
	{"sec.privkey", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY( BLOCK)?-----`)},
	{"sec.jwt", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`)},
	{"sec.apikey", regexp.MustCompile(`\b(sk-[A-Za-z0-9_\-]{20,}|sk-ant-[A-Za-z0-9_\-]{20,}|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,}|glpat-[A-Za-z0-9_\-]{18,}|A(KIA|SIA)[0-9A-Z]{16}|AIza[0-9A-Za-z_\-]{35}|xox[baprs]-[A-Za-z0-9\-]{10,}|[sr]k_(live|test)_[A-Za-z0-9]{16,}|whsec_[A-Za-z0-9]{20,}|SG\.[A-Za-z0-9_\-]{16,}\.[A-Za-z0-9_\-]{16,}|hf_[A-Za-z0-9]{30,}|npm_[A-Za-z0-9]{30,}|pypi-[A-Za-z0-9_\-]{30,}|ya29\.[A-Za-z0-9_\-]{30,}|AccountKey=[A-Za-z0-9+/=]{40,})`)},
	{"sec.apikey", regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/\-]{20,}=*`)},
	{"sec.conn", regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.\-]*://[^\s/:@]+:[^\s/@]{3,}@[^\s]+`)},
	{"sec.password", regexp.MustCompile(`(?i)\b(pass(word|wd)?|pwd|secret|token|api[_\-]?key|access[_\-]?key|private[_\-]?key|client[_\-]?secret|authorization|密码|口令|密钥)\s*[:=:=]\s*["']?([^\s"']{6,})`)},
	{"sec.otp", regexp.MustCompile(`(?i)(验证码|校验码|动态码|动态密码|安全码|verification code|security code|one[- ]time|otp|passcode|code is|code:)\D{0,14}\d{4,8}`)},
	{"sec.wallet", regexp.MustCompile(`(?i)^(0x)?[0-9a-f]{64}$`)},
	{"sec.ssn", regexp.MustCompile(`^\d{3}-\d{2}-\d{4}$`)},
}

var reIDCard2 = regexp.MustCompile(`^\d{17}[\dXx]$`)

func secretReason(text string) string {
	t := strings.TrimSpace(text)
	if t == "" || len(t) > 20000 {
		return ""
	}
	// 证件号 / 银行卡
	if len(t) <= 40 {
		if reIDCard2.MatchString(t) {
			return "sec.idcard"
		}
		var b strings.Builder
		onlyDigits := true
		for _, r := range t {
			switch {
			case r >= '0' && r <= '9':
				b.WriteRune(r)
			case r == ' ' || r == '-':
			default:
				onlyDigits = false
			}
		}
		if d := b.String(); onlyDigits && len(d) >= 13 && len(d) <= 19 && luhn(d) {
			return "sec.card"
		}
	}
	for _, p := range secretPatterns {
		if p.key == "sec.wallet" && len(t) > 70 {
			continue
		}
		if p.key == "sec.password" {
			for _, m := range p.re.FindAllStringSubmatch(t, -1) {
				if !isPlaceholder(m[len(m)-1]) {
					return p.key
				}
			}
			continue
		}
		if p.re.MatchString(t) {
			return p.key
		}
	}
	if looksLikeRandomSecret(t) {
		return "sec.random"
	}
	return ""
}

var rePlaceholder = regexp.MustCompile(`(?i)^(\$|%|<|\{\{|\[|your|my_|xxx|\*+$|\.{3}|…|changeme|example|placeholder|null$|none$|true$|false$|undefined$|env\.|process\.|os\.|config|secret_?key$|password$|token$)`)

// isPlaceholder 判断 `password=xxx` 里的值是不是占位符/变量引用。
func isPlaceholder(v string) bool {
	v = strings.Trim(v, `"'`+"`;,)")
	return len(v) < 6 || rePlaceholder.MatchString(v) || strings.HasPrefix(strings.ToLower(v), "x") && strings.Trim(strings.ToLower(v), "x") == ""
}

// 把字符串切成“词片段”:驼峰单词、数字串、其他单字符。标识符片段长,随机串片段短。
func avgPieceLen(t string) float64 {
	rs := []rune(t)
	pieces, i := 0, 0
	for i < len(rs) {
		r := rs[i]
		j := i + 1
		switch {
		case unicode.IsUpper(r):
			for j < len(rs) && unicode.IsLower(rs[j]) {
				j++
			}
			if j == i+1 { // 连续大写缩写
				for j < len(rs) && unicode.IsUpper(rs[j]) && !(j+1 < len(rs) && unicode.IsLower(rs[j+1])) {
					j++
				}
			}
		case unicode.IsLower(r):
			for j < len(rs) && unicode.IsLower(rs[j]) {
				j++
			}
		case unicode.IsDigit(r):
			for j < len(rs) && unicode.IsDigit(rs[j]) {
				j++
			}
		}
		pieces++
		i = j
	}
	return float64(len(rs)) / float64(pieces)
}

// 单个无空格的“看起来像随机密码/令牌”的字符串
func looksLikeRandomSecret(t string) bool {
	if len(t) < 12 || len(t) > 128 || strings.ContainsAny(t, " \t\r\n/\\") {
		return false
	}
	if classify0(t) || reUUID.MatchString(t) || isHashLike(t) {
		return false
	}
	if avgPieceLen(t) >= 2.2 {
		return false // 标识符 / 文件名 / 普通单词组合
	}
	var lower, upper, digit, sym int
	freq := map[rune]int{}
	n := 0
	for _, r := range t {
		n++
		freq[r]++
		switch {
		case unicode.IsLower(r):
			lower++
		case unicode.IsUpper(r):
			upper++
		case unicode.IsDigit(r):
			digit++
		case r < 128:
			sym++
		default:
			return false
		}
	}
	classes := 0
	for _, c := range []int{lower, upper, digit, sym} {
		if c > 0 {
			classes++
		}
	}
	if classes < 3 {
		return false
	}
	ent := 0.0
	for _, c := range freq {
		p := float64(c) / float64(n)
		ent -= p * math.Log2(p)
	}
	return ent >= 3.3
}

// classify0 排除明显不是密码的形态(邮箱、网址、版本号等)
func classify0(t string) bool {
	return reURL.MatchString(t) || reEmail.MatchString(t) || reHexColor.MatchString(t) || strings.Contains(t, "..")
}

// ---------- 更多内容类型 ----------

var (
	reDate1   = regexp.MustCompile(`^\d{4}[-/.年]\d{1,2}[-/.月]\d{1,2}[日号]?(\s+\d{1,2}[:：]\d{2}([:：]\d{2})?)?$`)
	reDate2   = regexp.MustCompile(`^\d{1,2}[:：]\d{2}([:：]\d{2})?$`)
	reDate3   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?(\.\d+)?(Z|[+-]\d{2}:?\d{2})?$`)
	reDate4   = regexp.MustCompile(`^\d{1,2}[/.\-]\d{1,2}[/.\-]\d{4}$`)
	reIPv4    = regexp.MustCompile(`^((25[0-5]|2[0-4]\d|1?\d?\d)\.){3}(25[0-5]|2[0-4]\d|1?\d?\d)(:\d{1,5})?(/\d{1,2})?$`)
	reIPv6    = regexp.MustCompile(`^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}(/\d{1,3})?$`)
	reMAC     = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$`)
	reDomain  = regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9\-]*\.)+([a-zA-Z]{2,})(:\d{1,5})?$`)
	reUUID    = regexp.MustCompile(`^[0-9a-fA-F]{8}-([0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}$`)
	reHexHash = regexp.MustCompile(`^(0x)?[0-9a-fA-F]+$`)
	reAddrCN  = regexp.MustCompile(`省|市|区|县|镇|乡|村|路|街|道|巷|弄|号|栋|幢|楼|室|单元`)
	reXMLish  = regexp.MustCompile(`(?s)^<[A-Za-z!?][^>]*>.*</[A-Za-z][^>]*>\s*$`)
)

var knownTLD = map[string]bool{"com": true, "net": true, "org": true, "cn": true, "io": true, "dev": true, "app": true, "co": true,
	"info": true, "biz": true, "xyz": true, "top": true, "me": true, "tv": true, "cc": true, "us": true, "uk": true, "de": true,
	"jp": true, "fr": true, "ru": true, "in": true, "edu": true, "gov": true, "ai": true, "cloud": true, "tech": true, "site": true,
	"online": true, "shop": true, "kr": true, "hk": true, "tw": true, "sg": true, "au": true, "ca": true, "br": true}

func isHashLike(t string) bool {
	h := strings.TrimPrefix(t, "0x")
	if !reHexHash.MatchString(t) {
		return false
	}
	switch len(h) {
	case 24, 32, 40, 56, 64, 96, 128:
		hasLetter, hasDigit := false, false
		for _, r := range h {
			if r >= '0' && r <= '9' {
				hasDigit = true
			} else {
				hasLetter = true
			}
		}
		return hasLetter && hasDigit
	}
	return false
}

// classifyMore 识别单行内容的更多类型,未命中返回 (0,false)。
func classifyMore(t string) (Kind, bool) {
	switch {
	case reDate1.MatchString(t), reDate2.MatchString(t), reDate3.MatchString(t), reDate4.MatchString(t):
		return KDate, true
	case reIPv4.MatchString(t), reMAC.MatchString(t):
		return KNetwork, true
	case strings.Count(t, ":") >= 2 && reIPv6.MatchString(t):
		return KNetwork, true
	case reUUID.MatchString(t), isHashLike(t):
		return KID, true
	}
	if !strings.ContainsAny(t, " /\\@") && reDomain.MatchString(t) {
		host := strings.SplitN(t, ":", 2)[0]
		if i := strings.LastIndex(host, "."); i >= 0 && knownTLD[strings.ToLower(host[i+1:])] {
			return KNetwork, true
		}
	}
	// 中文地址:含多个地址单位、以中文为主、较短
	if rs := []rune(t); len(rs) >= 6 && len(rs) <= 100 && cjkRatio(t) > 0.6 {
		if len(reAddrCN.FindAllString(t, -1)) >= 3 && !strings.ContainsAny(t, "。!?!?") {
			return KAddress, true
		}
	}
	return 0, false
}

// classifyBlock 识别多行/结构化内容:JSON、XML、表格。
func classifyBlock(t string) (Kind, bool) {
	if (strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}")) || (strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]")) {
		if json.Valid([]byte(t)) {
			return KData, true
		}
	}
	if reXMLish.MatchString(t) && strings.Count(t, "<") >= 2 {
		return KData, true
	}
	lines := strings.Split(strings.ReplaceAll(t, "\r", ""), "\n")
	if len(lines) >= 2 {
		tabs, commas, ok := -1, -1, true
		okc := true
		for _, l := range lines {
			if l == "" {
				continue
			}
			tc, cc := strings.Count(l, "\t"), strings.Count(l, ",")
			if tabs == -1 {
				tabs, commas = tc, cc
			}
			if tc == 0 || tc != tabs {
				ok = false
			}
			if cc < 2 || cc != commas {
				okc = false
			}
		}
		if ok || (okc && len(lines) >= 3) {
			return KTable, true
		}
	}
	return 0, false
}
