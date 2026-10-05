package main

import "math"

// renderBall 绘制悬浮球(预乘 alpha 的 BGRA),size 为画布边长像素。
func renderBall(size int, dark bool, acc Color, hover bool) *Canvas {
	c := NewCanvas(size, size)
	s := float64(size)
	cx, cy := s/2, s/2
	R := s * 0.38
	body := Color{246, 249, 255}
	bodyA := 0.90
	if dark {
		body = Color{24, 28, 42}
		bodyA = 0.88
	}
	for i := 0; i < len(c.Pix); i += 4 {
		c.Pix[i], c.Pix[i+1], c.Pix[i+2] = body.B, body.G, body.R
	}
	// 强调色光晕 + 顶部高光,与主面板同一套玻璃语言
	for k := 0; k < 4; k++ {
		c.FillCircle(cx+R*0.55, cy-R*0.6, R*(1.0-0.17*float64(k)), acc, 0.07)
	}
	ga := 0.45
	if dark {
		ga = 0.16
	}
	c.VGradient(Rect{0, int(cy - R), size, int(R)}, Color{255, 255, 255}, ga, 0)
	ic := drawAppIcon(int(R * 1.2))
	c.Blit(ic, int(cx)-ic.W/2, int(cy)-ic.H/2)
	if hover {
		c.StrokeCircle(cx, cy, R-1.2, 2.2, acc, 0.95)
	} else {
		rim := Color{255, 255, 255}
		ra := 0.55
		if !dark {
			rim, ra = Color{255, 255, 255}, 0.9
		}
		c.StrokeCircle(cx, cy, R-0.8, 1.4, rim, ra*0.6)
	}
	// 圆形遮罩 + 阴影,转为预乘
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			d := math.Hypot(px-cx, py-cy) - R
			cov := clamp01(0.5-d) * bodyA
			d2 := math.Hypot(px-cx, py-cy-s*0.035) - R
			sh := 0.0
			if d2 > -2 {
				sh = 0.38 * math.Exp(-math.Max(d2, 0)/(s*0.045))
			}
			a := cov + sh*(1-cov)
			i := (y*size + x) * 4
			// 颜色按 cov 预乘(阴影是黑色,不贡献颜色)
			c.Pix[i] = clampByte(float64(c.Pix[i]) * cov)
			c.Pix[i+1] = clampByte(float64(c.Pix[i+1]) * cov)
			c.Pix[i+2] = clampByte(float64(c.Pix[i+2]) * cov)
			c.Pix[i+3] = clampByte(a * 255)
		}
	}
	return c
}
