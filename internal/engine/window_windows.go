//go:build windows

package engine

import (
	"syscall"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
)

var (
	dwmapi                    = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	currentWindow             *glfw.Window
)

const (
	DWMWA_USE_IMMERSIVE_DARK_MODE = 20
	DWMWA_CAPTION_COLOR           = 35
	DWMWA_BORDER_COLOR            = 34
)

func SetDarkTitleBar(window *glfw.Window) {
	currentWindow = window
	hwnd := window.GetWin32Window()
	if hwnd == nil {
		return
	}

	var useDarkMode int32 = 1
	procDwmSetWindowAttribute.Call(
		uintptr(unsafe.Pointer(hwnd)),
		DWMWA_USE_IMMERSIVE_DARK_MODE,
		uintptr(unsafe.Pointer(&useDarkMode)),
		unsafe.Sizeof(useDarkMode),
	)

	var borderColor uint32 = 0x00000000
	procDwmSetWindowAttribute.Call(
		uintptr(unsafe.Pointer(hwnd)),
		DWMWA_BORDER_COLOR,
		uintptr(unsafe.Pointer(&borderColor)),
		unsafe.Sizeof(borderColor),
	)

	var captionColor uint32 = 0x00202020
	procDwmSetWindowAttribute.Call(
		uintptr(unsafe.Pointer(hwnd)),
		DWMWA_CAPTION_COLOR,
		uintptr(unsafe.Pointer(&captionColor)),
		unsafe.Sizeof(captionColor),
	)
}

func SetWindowBorderColor(r, g, b float32) {
	if currentWindow == nil {
		return
	}
	hwnd := currentWindow.GetWin32Window()
	if hwnd == nil {
		return
	}

	colorBGR := uint32(uint8(b*255)) | uint32(uint8(g*255))<<8 | uint32(uint8(r*255))<<16
	procDwmSetWindowAttribute.Call(
		uintptr(unsafe.Pointer(hwnd)),
		DWMWA_BORDER_COLOR,
		uintptr(unsafe.Pointer(&colorBGR)),
		unsafe.Sizeof(colorBGR),
	)

	procDwmSetWindowAttribute.Call(
		uintptr(unsafe.Pointer(hwnd)),
		DWMWA_CAPTION_COLOR,
		uintptr(unsafe.Pointer(&colorBGR)),
		unsafe.Sizeof(colorBGR),
	)
}
