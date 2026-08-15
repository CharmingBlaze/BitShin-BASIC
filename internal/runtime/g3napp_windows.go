package runtime

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Ebiten embeds its own GLFW and, on import, creates a hidden
// "GLFW message window" plus the GLFW30 class. G3N uses go-gl's GLFW,
// which then cannot RegisterClass("GLFW30").
func releaseForeignGLFWClass() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	findWindow := user32.NewProc("FindWindowW")
	destroyWindow := user32.NewProc("DestroyWindow")
	unregister := user32.NewProc("UnregisterClassW")
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getModuleHandle := kernel32.NewProc("GetModuleHandleW")

	className, err := windows.UTF16PtrFromString("GLFW30")
	if err != nil {
		return
	}
	title, err := windows.UTF16PtrFromString("GLFW message window")
	if err != nil {
		return
	}
	hwnd, _, _ := findWindow.Call(uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)))
	if hwnd != 0 {
		destroyWindow.Call(hwnd)
	}
	hinst, _, _ := getModuleHandle.Call(0)
	unregister.Call(uintptr(unsafe.Pointer(className)), hinst)
}
