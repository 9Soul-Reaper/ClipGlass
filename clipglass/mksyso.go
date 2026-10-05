//go:build !windows

package main

// 生成 rsrc_windows_amd64.syso:内嵌 exe 图标(PNG 压缩的 256x256)与应用清单。
// 用法: go run . --mksyso

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
)

const manifestXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="ClipGlass" version="1.0.0.0"/>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges>
    <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
  </requestedPrivileges></security></trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1"><application>
    <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
    <supportedOS Id="{1f676c76-80e1-4239-95bb-83d0f6d0da78}"/>
    <supportedOS Id="{4a2f28e3-53b9-4441-ba9c-d69d4a4a6e38}"/>
    <supportedOS Id="{35138b9a-5d96-4fbd-8e2d-a2440225f93a}"/>
  </application></compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3"><windowsSettings>
    <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true</dpiAware>
    <longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware>
  </windowsSettings></application>
</assembly>`

func makeSyso(path string) error {
	ic := drawAppIcon(256)
	img := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			i := (y*256 + x) * 4
			img.SetNRGBA(x, y, color.NRGBA{ic.Pix[i+2], ic.Pix[i+1], ic.Pix[i], ic.Pix[i+3]})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return err
	}
	pngData := pngBuf.Bytes()

	grp := new(bytes.Buffer)
	binary.Write(grp, binary.LittleEndian, []uint16{0, 1, 1})
	grp.Write([]byte{0, 0, 0, 0}) // 宽 高(0=256) 色数 保留
	binary.Write(grp, binary.LittleEndian, uint16(1))
	binary.Write(grp, binary.LittleEndian, uint16(32))
	binary.Write(grp, binary.LittleEndian, uint32(len(pngData)))
	binary.Write(grp, binary.LittleEndian, uint16(1))

	blobs := [][]byte{pngData, grp.Bytes(), []byte(manifestXML)}
	types := []uint32{3, 14, 24}

	le := binary.LittleEndian
	sec := new(bytes.Buffer)
	dir := func(nid uint16) {
		binary.Write(sec, le, uint32(0))
		binary.Write(sec, le, uint32(0))
		binary.Write(sec, le, []uint16{0, 0, 0, nid})
	}
	ent := func(id, off uint32) { binary.Write(sec, le, id); binary.Write(sec, le, off) }

	dir(3)
	for i, t := range types {
		ent(t, 0x80000000|uint32(40+24*i))
	}
	for i := range types {
		dir(1)
		ent(1, 0x80000000|uint32(112+24*i))
	}
	for i := range types {
		dir(1)
		ent(0x409, uint32(184+16*i))
	}
	// 数据条目
	off := 232
	var relocs []uint32
	for i, b := range blobs {
		relocs = append(relocs, uint32(sec.Len()))
		binary.Write(sec, le, uint32(off))
		binary.Write(sec, le, uint32(len(b)))
		binary.Write(sec, le, uint32(0))
		binary.Write(sec, le, uint32(0))
		_ = i
		off += (len(b) + 3) &^ 3
	}
	for _, b := range blobs {
		sec.Write(b)
		for sec.Len()%4 != 0 {
			sec.WriteByte(0)
		}
	}

	out := new(bytes.Buffer)
	rawOff := 20 + 40
	relOff := rawOff + sec.Len()
	symOff := relOff + len(relocs)*10
	binary.Write(out, le, uint16(0x8664))
	binary.Write(out, le, uint16(1))
	binary.Write(out, le, uint32(0))
	binary.Write(out, le, uint32(symOff))
	binary.Write(out, le, uint32(1+1)) // 符号 + 辅助记录
	binary.Write(out, le, uint16(0))
	binary.Write(out, le, uint16(0))
	// 节头
	out.Write([]byte(".rsrc\x00\x00\x00"))
	binary.Write(out, le, uint32(0))
	binary.Write(out, le, uint32(0))
	binary.Write(out, le, uint32(sec.Len()))
	binary.Write(out, le, uint32(rawOff))
	binary.Write(out, le, uint32(relOff))
	binary.Write(out, le, uint32(0))
	binary.Write(out, le, uint16(len(relocs)))
	binary.Write(out, le, uint16(0))
	binary.Write(out, le, uint32(0x40000040))
	out.Write(sec.Bytes())
	for _, r := range relocs {
		binary.Write(out, le, r)
		binary.Write(out, le, uint32(0))
		binary.Write(out, le, uint16(3)) // IMAGE_REL_AMD64_ADDR32NB
	}
	// 符号表:.rsrc 节符号 + 辅助记录
	out.Write([]byte(".rsrc\x00\x00\x00"))
	binary.Write(out, le, uint32(0))
	binary.Write(out, le, uint16(1))
	binary.Write(out, le, uint16(0))
	out.WriteByte(3) // STATIC
	out.WriteByte(1) // 1 个辅助记录
	binary.Write(out, le, uint32(sec.Len()))
	binary.Write(out, le, uint16(len(relocs)))
	binary.Write(out, le, uint16(0))
	binary.Write(out, le, uint32(0))
	binary.Write(out, le, uint16(1))
	out.Write([]byte{0, 0, 0})
	binary.Write(out, le, uint32(4)) // 字符串表
	return os.WriteFile(path, out.Bytes(), 0o644)
}
