package main

// drawAppIcon 绘制应用图标(BGRA,非预乘 alpha),size 为边长像素。
func drawAppIcon(size int) *Canvas {
	c := NewCanvas(size, size)
	s := float64(size)
	// 圆角渐变底(蓝 -> 紫)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			t := (float64(x) + float64(y)) / (2 * s)
			i := (y*size + x) * 4
			c.Pix[i] = clampByte(255 - 25*t)   // B
			c.Pix[i+1] = clampByte(150 - 60*t) // G
			c.Pix[i+2] = clampByte(70 + 120*t) // R
			c.Pix[i+3] = 255
		}
	}
	// 夹板
	c.FillRRect(s*0.24, s*0.20, s*0.52, s*0.62, s*0.10, Color{255, 255, 255}, 0.95)
	// 夹子
	c.FillRRect(s*0.36, s*0.13, s*0.28, s*0.15, s*0.06, Color{226, 232, 255}, 1)
	c.FillRRect(s*0.42, s*0.08, s*0.16, s*0.10, s*0.05, Color{150, 165, 240}, 1)
	// 文本行
	lc := Color{110, 125, 220}
	c.FillRRect(s*0.33, s*0.44, s*0.34, s*0.05, s*0.025, lc, 0.9)
	c.FillRRect(s*0.33, s*0.55, s*0.34, s*0.05, s*0.025, lc, 0.9)
	c.FillRRect(s*0.33, s*0.66, s*0.22, s*0.05, s*0.025, lc, 0.9)
	// 玻璃高光
	c.VGradient(Rect{0, 0, size, size / 2}, Color{255, 255, 255}, 0.28, 0)
	// 圆角 alpha
	r := s * 0.24
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			d := sdRoundRect(float64(x)+0.5, float64(y)+0.5, s/2, s/2, s/2, s/2, r)
			c.Pix[(y*size+x)*4+3] = clampByte(clamp01(0.5-d) * 255)
		}
	}
	return c
}
