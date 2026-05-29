//go:build windows

package prompt

import "syscall"

var procGetAsyncKeyState = syscall.NewLazyDLL("user32.dll").NewProc("GetAsyncKeyState")

const vkMenu = 0x12

func isAltKeyPressed() bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vkMenu))
	return ret&0x8000 != 0
}
