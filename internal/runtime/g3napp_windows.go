package runtime

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Ebiten embeds its own GLFW and, on import, creates a hidden
// "GLFW message window" plus the GLFW30 class. G3N uses go-gl's GLFW,
// which then cannot RegisterClass("GLFW30").
//
// FindWindow searches the whole desktop, so it walks every match and acts only
// on this process's. Taking the first one found another running GLFW program's
// window, which cannot be destroyed from here: the class stayed registered and
// the game died at launch whenever such a program was on top.
func releaseForeignGLFWClass() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	findWindowEx := user32.NewProc("FindWindowExW")
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
	self := windows.GetCurrentProcessId()
	var mine []uintptr
	var after uintptr
	for {
		hwnd, _, _ := findWindowEx.Call(0, after, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)))
		if hwnd == 0 {
			break
		}
		var pid uint32
		if _, err := windows.GetWindowThreadProcessId(windows.HWND(hwnd), &pid); err == nil && pid == self {
			mine = append(mine, hwnd)
		}
		after = hwnd
	}
	// Destroyed after the walk: removing a window mid-walk would lose its place.
	for _, hwnd := range mine {
		destroyWindow.Call(hwnd)
	}
	hinst, _, _ := getModuleHandle.Call(0)
	unregister.Call(uintptr(unsafe.Pointer(className)), hinst)
}
