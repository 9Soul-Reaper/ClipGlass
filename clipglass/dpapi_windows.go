//go:build windows

package main

import (
	"errors"
	"syscall"
	"unsafe"
)

var (
	crypt32           = syscall.NewLazyDLL("crypt32.dll")
	pCryptProtectData = crypt32.NewProc("CryptProtectData")
	pCryptUnprotect   = crypt32.NewProc("CryptUnprotectData")
	pLocalFree        = syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree")
)

type dataBlob struct {
	cb uint32
	pb *byte
}

func dpapi(p *syscall.LazyProc, in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, errors.New("empty")
	}
	inB := dataBlob{uint32(len(in)), &in[0]}
	var out dataBlob
	r, _, e := p.Call(uintptr(unsafe.Pointer(&inB)), 0, 0, 0, 0, 0x1 /*CRYPTPROTECT_UI_FORBIDDEN*/, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, e
	}
	defer pLocalFree.Call(uintptr(unsafe.Pointer(out.pb)))
	res := make([]byte, out.cb)
	copy(res, unsafe.Slice(out.pb, out.cb))
	return res, nil
}

func init() {
	protectFn = func(b []byte) ([]byte, error) { return dpapi(pCryptProtectData, b) }
	unprotectFn = func(b []byte) ([]byte, error) { return dpapi(pCryptUnprotect, b) }
}
