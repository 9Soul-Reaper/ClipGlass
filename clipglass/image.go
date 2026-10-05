package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
)

// parseDIB 解析剪贴板 CF_DIB 数据(24/32 位,BI_RGB 或 BI_BITFIELDS)。
func parseDIB(b []byte) *image.NRGBA {
	if len(b) < 40 {
		return nil
	}
	le := binary.LittleEndian
	hs := int(le.Uint32(b[0:]))
	if hs < 40 || hs > len(b) {
		return nil
	}
	w := int(int32(le.Uint32(b[4:])))
	h := int(int32(le.Uint32(b[8:])))
	bpp := int(le.Uint16(b[14:]))
	comp := le.Uint32(b[16:])
	clrUsed := int(le.Uint32(b[32:]))
	topDown := h < 0
	if topDown {
		h = -h
	}
	if w <= 0 || h <= 0 || w*h > 120_000_000 {
		return nil
	}
	if (bpp != 24 && bpp != 32) || (comp != 0 && comp != 3) {
		return nil
	}
	off := hs
	rm, gm, bm := uint32(0x00FF0000), uint32(0x0000FF00), uint32(0x000000FF)
	if comp == 3 {
		switch {
		case hs == 40 && len(b) >= 52:
			rm, gm, bm = le.Uint32(b[40:]), le.Uint32(b[44:]), le.Uint32(b[48:])
			off += 12
		case hs >= 52:
			rm, gm, bm = le.Uint32(b[40:]), le.Uint32(b[44:]), le.Uint32(b[48:])
		}
	} else if clrUsed > 0 && clrUsed < 1024 {
		off += clrUsed * 4
	}
	stride := ((w*bpp + 31) / 32) * 4
	if off+stride*h > len(b) {
		return nil
	}
	shift := func(m uint32) uint {
		if m == 0 {
			return 0
		}
		s := uint(0)
		for m&1 == 0 {
			m >>= 1
			s++
		}
		return s
	}
	rs, gs, bs := shift(rm), shift(gm), shift(bm)
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := h - 1 - y
		if topDown {
			sy = y
		}
		row := b[off+sy*stride:]
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			if bpp == 24 {
				dst[x*4], dst[x*4+1], dst[x*4+2] = row[x*3+2], row[x*3+1], row[x*3]
			} else {
				px := le.Uint32(row[x*4:])
				dst[x*4] = byte((px & rm) >> rs)
				dst[x*4+1] = byte((px & gm) >> gs)
				dst[x*4+2] = byte((px & bm) >> bs)
			}
			dst[x*4+3] = 255
		}
	}
	return img
}

// buildDIB 生成 32 位自下而上的 CF_DIB 数据。
func buildDIB(img *image.NRGBA) []byte {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	out := make([]byte, 40+w*h*4)
	le := binary.LittleEndian
	le.PutUint32(out[0:], 40)
	le.PutUint32(out[4:], uint32(w))
	le.PutUint32(out[8:], uint32(h))
	le.PutUint16(out[12:], 1)
	le.PutUint16(out[14:], 32)
	le.PutUint32(out[20:], uint32(w*h*4))
	for y := 0; y < h; y++ {
		src := img.Pix[(h-1-y)*img.Stride:]
		dst := out[40+y*w*4:]
		for x := 0; x < w; x++ {
			dst[x*4], dst[x*4+1], dst[x*4+2], dst[x*4+3] = src[x*4+2], src[x*4+1], src[x*4], 255
		}
	}
	return out
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if enc.Encode(&buf, img) != nil {
		return nil
	}
	return buf.Bytes()
}

// coverThumb 居中裁成正方形并缩放到 size×size(面积平均)。
func coverThumb(img *image.NRGBA, size int) *image.NRGBA {
	bw, bh := img.Bounds().Dx(), img.Bounds().Dy()
	side := bw
	if bh < side {
		side = bh
	}
	ox, oy := (bw-side)/2, (bh-side)/2
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		y0, y1 := oy+y*side/size, oy+(y+1)*side/size
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < size; x++ {
			x0, x1 := ox+x*side/size, ox+(x+1)*side/size
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, bl, n int
			// 大图最多采样 ~8x8 个点,够平滑且很快
			sy, sx := (y1-y0+7)/8, (x1-x0+7)/8
			for yy := y0; yy < y1; yy += sy {
				for xx := x0; xx < x1; xx += sx {
					i := yy*img.Stride + xx*4
					r += int(img.Pix[i])
					g += int(img.Pix[i+1])
					bl += int(img.Pix[i+2])
					n++
				}
			}
			o := y*out.Stride + x*4
			out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = byte(r/n), byte(g/n), byte(bl/n), 255
		}
	}
	return out
}

// 把 NRGBA 转为 BGRA 画布(用于绘制缩略图)
func canvasFromNRGBA(img *image.NRGBA) *Canvas {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	c := NewCanvas(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s := y*img.Stride + x*4
			d := (y*w + x) * 4
			c.Pix[d], c.Pix[d+1], c.Pix[d+2], c.Pix[d+3] = img.Pix[s+2], img.Pix[s+1], img.Pix[s], img.Pix[s+3]
		}
	}
	return c
}
