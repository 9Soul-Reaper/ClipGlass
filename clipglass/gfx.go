package main

import "math"

// ---------- 基础类型 ----------

type Rect struct{ X, Y, W, H int }

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

func (r Rect) Intersect(o Rect) Rect {
	x0, y0 := maxi(r.X, o.X), maxi(r.Y, o.Y)
	x1, y1 := mini(r.X+r.W, o.X+o.W), mini(r.Y+r.H, o.Y+o.H)
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

type Color struct{ R, G, B uint8 }

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ---------- 画布(BGRA,与 Windows DIB 内存布局一致) ----------

type Canvas struct {
	W, H int
	Pix  []byte
	Clip Rect
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{W: w, H: h, Pix: make([]byte, w*h*4), Clip: Rect{0, 0, w, h}}
}

func (c *Canvas) SetClip(r Rect) { c.Clip = r.Intersect(Rect{0, 0, c.W, c.H}) }

func (c *Canvas) blend(x, y int, col Color, a float64) {
	if x < c.Clip.X || y < c.Clip.Y || x >= c.Clip.X+c.Clip.W || y >= c.Clip.Y+c.Clip.H {
		return
	}
	i := (y*c.W + x) * 4
	p := c.Pix
	p[i] = uint8(float64(p[i]) + (float64(col.B)-float64(p[i]))*a + 0.5)
	p[i+1] = uint8(float64(p[i+1]) + (float64(col.G)-float64(p[i+1]))*a + 0.5)
	p[i+2] = uint8(float64(p[i+2]) + (float64(col.R)-float64(p[i+2]))*a + 0.5)
}

// 圆角矩形有向距离场(<0 在内部)
func sdRoundRect(px, py, cx, cy, hw, hh, r float64) float64 {
	qx := math.Abs(px-cx) - (hw - r)
	qy := math.Abs(py-cy) - (hh - r)
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

func (c *Canvas) bbox(x, y, w, h float64) (x0, y0, x1, y1 int) {
	x0 = maxi(int(math.Floor(x))-1, c.Clip.X)
	y0 = maxi(int(math.Floor(y))-1, c.Clip.Y)
	x1 = mini(int(math.Ceil(x+w))+1, c.Clip.X+c.Clip.W)
	y1 = mini(int(math.Ceil(y+h))+1, c.Clip.Y+c.Clip.H)
	return
}

func (c *Canvas) FillRRect(x, y, w, h, r float64, col Color, a float64) {
	if a <= 0 || w <= 0 || h <= 0 {
		return
	}
	hw, hh := w/2, h/2
	r = math.Min(r, math.Min(hw, hh))
	cx, cy := x+hw, y+hh
	x0, y0, x1, y1 := c.bbox(x, y, w, h)
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := sdRoundRect(float64(px)+0.5, float64(py)+0.5, cx, cy, hw, hh, r)
			if cov := clamp01(0.5 - d); cov > 0 {
				c.blend(px, py, col, a*cov)
			}
		}
	}
}

func (c *Canvas) StrokeRRect(x, y, w, h, r, width float64, col Color, a float64) {
	if a <= 0 {
		return
	}
	hw, hh := w/2, h/2
	r = math.Min(r, math.Min(hw, hh))
	cx, cy := x+hw, y+hh
	x0, y0, x1, y1 := c.bbox(x, y, w, h)
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := sdRoundRect(float64(px)+0.5, float64(py)+0.5, cx, cy, hw, hh, r)
			cov := clamp01(0.5-d) * clamp01(0.5+d+width)
			if cov > 0 {
				c.blend(px, py, col, a*cov)
			}
		}
	}
}

func (c *Canvas) FillCircle(cx, cy, r float64, col Color, a float64) {
	c.FillRRect(cx-r, cy-r, r*2, r*2, r, col, a)
}

func (c *Canvas) StrokeCircle(cx, cy, r, width float64, col Color, a float64) {
	x0, y0, x1, y1 := c.bbox(cx-r-width, cy-r-width, (r+width)*2, (r+width)*2)
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := math.Hypot(float64(px)+0.5-cx, float64(py)+0.5-cy)
			cov := clamp01(width/2 + 0.5 - math.Abs(d-r))
			if cov > 0 {
				c.blend(px, py, col, a*cov)
			}
		}
	}
}

func (c *Canvas) Line(x1, y1, x2, y2, width float64, col Color, a float64) {
	minx, maxx := math.Min(x1, x2)-width, math.Max(x1, x2)+width
	miny, maxy := math.Min(y1, y2)-width, math.Max(y1, y2)+width
	bx0, by0, bx1, by1 := c.bbox(minx, miny, maxx-minx, maxy-miny)
	dx, dy := x2-x1, y2-y1
	l2 := dx*dx + dy*dy
	for py := by0; py < by1; py++ {
		for px := bx0; px < bx1; px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			t := 0.0
			if l2 > 0 {
				t = clamp01(((fx-x1)*dx + (fy-y1)*dy) / l2)
			}
			d := math.Hypot(fx-(x1+t*dx), fy-(y1+t*dy))
			if cov := clamp01(width/2 + 0.5 - d); cov > 0 {
				c.blend(px, py, col, a*cov)
			}
		}
	}
}

// 垂直渐变叠加(用于玻璃高光)
func (c *Canvas) VGradient(r Rect, col Color, a0, a1 float64) {
	r = r.Intersect(c.Clip)
	if r.Empty() {
		return
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		t := float64(y-r.Y) / float64(maxi(r.H-1, 1))
		a := a0 + (a1-a0)*t
		for x := r.X; x < r.X+r.W; x++ {
			c.blend(x, y, col, a)
		}
	}
}

// ---------- 毛玻璃:下采样 -> 盒式模糊 x3 -> 双线性放大 ----------

func blurLine(src, dst []byte, off, step, n, r int) {
	div := 2*r + 1
	for ch := 0; ch < 3; ch++ {
		sum := 0
		for i := -r; i <= r; i++ {
			j := i
			if j < 0 {
				j = 0
			}
			if j >= n {
				j = n - 1
			}
			sum += int(src[off+j*step+ch])
		}
		for x := 0; x < n; x++ {
			dst[off+x*step+ch] = byte(sum / div)
			a := x + r + 1
			if a >= n {
				a = n - 1
			}
			b := x - r
			if b < 0 {
				b = 0
			}
			sum += int(src[off+a*step+ch]) - int(src[off+b*step+ch])
		}
	}
}

func boxBlur(c *Canvas, r, passes int) {
	if r < 1 {
		return
	}
	tmp := make([]byte, len(c.Pix))
	for p := 0; p < passes; p++ {
		for y := 0; y < c.H; y++ {
			blurLine(c.Pix, tmp, y*c.W*4, 4, c.W, r)
		}
		for x := 0; x < c.W; x++ {
			blurLine(tmp, c.Pix, x*4, c.W*4, c.H, r)
		}
	}
}

func downscale(src *Canvas, k int) *Canvas {
	w, h := (src.W+k-1)/k, (src.H+k-1)/k
	dst := NewCanvas(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sb, sg, sr, n int
			for yy := y * k; yy < mini(y*k+k, src.H); yy++ {
				for xx := x * k; xx < mini(x*k+k, src.W); xx++ {
					i := (yy*src.W + xx) * 4
					sb += int(src.Pix[i])
					sg += int(src.Pix[i+1])
					sr += int(src.Pix[i+2])
					n++
				}
			}
			i := (y*w + x) * 4
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = byte(sb/n), byte(sg/n), byte(sr/n), 255
		}
	}
	return dst
}

func upscale(src *Canvas, w, h, k int) *Canvas {
	dst := NewCanvas(w, h)
	for y := 0; y < h; y++ {
		fy := (float64(y)+0.5)/float64(k) - 0.5
		y0 := int(math.Floor(fy))
		ty := fy - float64(y0)
		ya, yb := clampi(y0, 0, src.H-1), clampi(y0+1, 0, src.H-1)
		for x := 0; x < w; x++ {
			fx := (float64(x)+0.5)/float64(k) - 0.5
			x0 := int(math.Floor(fx))
			tx := fx - float64(x0)
			xa, xb := clampi(x0, 0, src.W-1), clampi(x0+1, 0, src.W-1)
			o := (y*w + x) * 4
			for ch := 0; ch < 3; ch++ {
				p00 := float64(src.Pix[(ya*src.W+xa)*4+ch])
				p10 := float64(src.Pix[(ya*src.W+xb)*4+ch])
				p01 := float64(src.Pix[(yb*src.W+xa)*4+ch])
				p11 := float64(src.Pix[(yb*src.W+xb)*4+ch])
				v := (p00*(1-tx)+p10*tx)*(1-ty) + (p01*(1-tx)+p11*tx)*ty
				dst.Pix[o+ch] = byte(v + 0.5)
			}
			dst.Pix[o+3] = 255
		}
	}
	return dst
}

// frost 返回模糊后的副本。radius 单位:像素。
func frost(src *Canvas, radius int) *Canvas {
	k := 4
	if radius < 12 {
		k = 2
	}
	small := downscale(src, k)
	boxBlur(small, maxi(1, radius/k), 3)
	return upscale(small, src.W, src.H, k)
}

func tintAndSaturate(c *Canvas, tint Color, tintA, sat float64, noise int) {
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			i := (y*c.W + x) * 4
			b, g, r := float64(c.Pix[i]), float64(c.Pix[i+1]), float64(c.Pix[i+2])
			l := 0.299*r + 0.587*g + 0.114*b
			r = l + (r-l)*sat
			g = l + (g-l)*sat
			b = l + (b-l)*sat
			r += (float64(tint.R) - r) * tintA
			g += (float64(tint.G) - g) * tintA
			b += (float64(tint.B) - b) * tintA
			if noise > 0 {
				h := uint32(x)*73856093 ^ uint32(y)*19349663
				h ^= h >> 13
				h *= 1274126177
				n := float64(int(h>>24)%(noise*2+1) - noise)
				r, g, b = r+n, g+n, b+n
			}
			c.Pix[i] = clampByte(b)
			c.Pix[i+1] = clampByte(g)
			c.Pix[i+2] = clampByte(r)
			c.Pix[i+3] = 255
		}
	}
}

func clampByte(v float64) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v + 0.5)
}

// 没有桌面截图时的兜底背景:柔和的彩色渐变
func fallbackBackdrop(w, h int, dark bool) *Canvas {
	c := NewCanvas(w, h)
	var c0, c1, c2 Color
	if dark {
		c0, c1, c2 = Color{40, 52, 96}, Color{88, 56, 120}, Color{30, 84, 110}
	} else {
		c0, c1, c2 = Color{170, 200, 255}, Color{225, 190, 250}, Color{170, 235, 225}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			u, v := float64(x)/float64(w), float64(y)/float64(h)
			i := (y*w + x) * 4
			mix := func(a, b, c uint8) byte {
				return clampByte(float64(a)*(1-u)*(1-v) + float64(b)*u*(1-v) + float64(c)*v)
			}
			c.Pix[i], c.Pix[i+1], c.Pix[i+2], c.Pix[i+3] = mix(c0.B, c1.B, c2.B), mix(c0.G, c1.G, c2.G), mix(c0.R, c1.R, c2.R), 255
		}
	}
	return c
}

// ---------- 成品窗口:圆角遮罩 + 阴影 + 边缘高光,输出预乘 BGRA ----------

// finalizeFrame 把 canvas 中 (m,m,iw,ih) 的内部区域裁成圆角,
// 外圈生成柔和阴影,并写入预乘 alpha。
func finalizeFrame(c *Canvas, m, iw, ih int, radius float64, dark bool, shadowPx float64) {
	cx, cy := float64(m)+float64(iw)/2, float64(m)+float64(ih)/2
	hw, hh := float64(iw)/2, float64(ih)/2
	maxA := 0.38
	if !dark {
		maxA = 0.30
	}
	rim := Color{255, 255, 255}
	rimA := 0.30
	if !dark {
		rimA = 0.75
	}
	ri := int(radius) + 2
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			i := (y*c.W + x) * 4
			// 快速路径:远离边缘与圆角的内部像素,直接不透明
			if (x >= m+ri && x < m+iw-ri && y >= m+2 && y < m+ih-2) ||
				(x >= m+2 && x < m+iw-2 && y >= m+ri && y < m+ih-ri) {
				c.Pix[i+3] = 255
				continue
			}
			d := sdRoundRect(float64(x)+0.5, float64(y)+0.5+shadowPx*0.25, cx, cy, hw, hh, radius)
			dEdge := sdRoundRect(float64(x)+0.5, float64(y)+0.5, cx, cy, hw, hh, radius)
			cov := clamp01(0.5 - dEdge)
			// 阴影(偏下)
			sa := 0.0
			if d > -shadowPx*2 {
				t := clamp01(1 - math.Max(d, 0)/shadowPx)
				sa = maxA * t * t
			}
			b, g, r := float64(c.Pix[i]), float64(c.Pix[i+1]), float64(c.Pix[i+2])
			if dEdge < 0.5 {
				// 内缘高光边
				band := clamp01(0.5-dEdge) * clamp01(1.4+dEdge)
				if band > 0 {
					k := rimA * band
					r += (float64(rim.R) - r) * k
					g += (float64(rim.G) - g) * k
					b += (float64(rim.B) - b) * k
				}
			}
			alpha := cov + (1-cov)*sa
			// 预乘:内容按 cov 贡献,阴影为黑色
			c.Pix[i] = clampByte(b * cov)
			c.Pix[i+1] = clampByte(g * cov)
			c.Pix[i+2] = clampByte(r * cov)
			c.Pix[i+3] = clampByte(alpha * 255)
		}
	}
}

// Blit 把带 alpha(非预乘)的 src 叠加到 (dx,dy)。
func (c *Canvas) Blit(src *Canvas, dx, dy int) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			i := (y*src.W + x) * 4
			a := float64(src.Pix[i+3]) / 255
			if a > 0 {
				c.blend(dx+x, dy+y, Color{src.Pix[i+2], src.Pix[i+1], src.Pix[i]}, a)
			}
		}
	}
}

// DrawCover 把 src 缩放绘制到 (x,y,size,size) 的圆角方块内(双线性采样)。
func (c *Canvas) DrawCover(src *Canvas, x, y, size, r float64) {
	x0, y0, x1, y1 := c.bbox(x, y, size, size)
	hw := size / 2
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			d := sdRoundRect(float64(px)+0.5, float64(py)+0.5, x+hw, y+hw, hw, hw, r)
			cov := clamp01(0.5 - d)
			if cov <= 0 {
				continue
			}
			fx := (float64(px)+0.5-x)/size*float64(src.W) - 0.5
			fy := (float64(py)+0.5-y)/size*float64(src.H) - 0.5
			ix, iy := int(math.Floor(fx)), int(math.Floor(fy))
			tx, ty := fx-float64(ix), fy-float64(iy)
			xa, xb := clampi(ix, 0, src.W-1), clampi(ix+1, 0, src.W-1)
			ya, yb := clampi(iy, 0, src.H-1), clampi(iy+1, 0, src.H-1)
			var ch [3]float64
			for k := 0; k < 3; k++ {
				p00 := float64(src.Pix[(ya*src.W+xa)*4+k])
				p10 := float64(src.Pix[(ya*src.W+xb)*4+k])
				p01 := float64(src.Pix[(yb*src.W+xa)*4+k])
				p11 := float64(src.Pix[(yb*src.W+xb)*4+k])
				ch[k] = (p00*(1-tx)+p10*tx)*(1-ty) + (p01*(1-tx)+p11*tx)*ty
			}
			c.blend(px, py, Color{clampByte(ch[2]), clampByte(ch[1]), clampByte(ch[0])}, cov)
		}
	}
}
