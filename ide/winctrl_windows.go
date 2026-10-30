//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type winRect struct {
	Left, Top, Right, Bottom int32
}

type monitorInfo struct {
	cbSize    uint32
	rcMonitor winRect
	rcWork    winRect
	dwFlags   uint32
}

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcId    = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procShowWindow               = user32.NewProc("ShowWindow")
	procIsIconic                 = user32.NewProc("IsIconic")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procMonitorFromWindow        = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procSystemParametersInfoW    = user32.NewProc("SystemParametersInfoW")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop         = user32.NewProc("BringWindowToTop")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	procIsWindow                 = user32.NewProc("IsWindow")
	procLoadCursorW              = user32.NewProc("LoadCursorW")
	procSetCursor                = user32.NewProc("SetCursor")
)

var resizeBusy atomic.Bool

const (
	swMinimize              = 6
	swRestore               = 9
	swpNoZOrder             = 0x0004
	swpShowWindow           = 0x0040
	monitorDefaultToNearest = 2
	spiGetWorkArea          = 0x0030
	wmClose                 = 0x0010
	swShow                  = 5
	swpNoMove               = 0x0002
	swpNoSize               = 0x0001
	swpNoActivate           = 0x0010
	vkLButton               = 0x01
	minWinW                 = 800
	minWinH                 = 500
)

type winPoint struct {
	X, Y int32
}

func (a *App) shellHWND() uintptr {
	if a.neuCmd == nil || a.neuCmd.Process == nil {
		return 0
	}
	return hwndForPID(uint32(a.neuCmd.Process.Pid))
}

func hwndForPID(pid uint32) uintptr {
	var best uintptr
	var bestArea int32
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		var owner uint32
		procGetWindowThreadProcId.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
		if owner != pid {
			return 1
		}
		vis, _, _ := procIsWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1
		}
		var r winRect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		area := (r.Right - r.Left) * (r.Bottom - r.Top)
		if area > bestArea {
			bestArea = area
			best = hwnd
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return best
}

func forceForeground(hwnd uintptr) {
	fg, _, _ := procGetForegroundWindow.Call()
	fgTid, _, _ := procGetWindowThreadProcId.Call(fg, 0)
	tgtTid, _, _ := procGetWindowThreadProcId.Call(hwnd, 0)
	if fgTid != 0 && tgtTid != 0 && fgTid != tgtTid {
		procAttachThreadInput.Call(fgTid, tgtTid, 1)
		defer procAttachThreadInput.Call(fgTid, tgtTid, 0)
	}
	procShowWindow.Call(hwnd, swShow)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
	procBringWindowToTop.Call(hwnd)
	procSetForegroundWindow.Call(hwnd)
}

func killProcessTree(pid int) {
	if pid <= 0 {
		return
	}
	// Process.Kill leaves child windows alive. /T ends the tree.
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	_ = cmd.Run()
}

func raiseProcessWindows(pid int) {
	if pid <= 0 {
		return
	}
	procAllowSetForegroundWindow.Call(uintptr(pid))
	deadline := time.Now().Add(6 * time.Second)
	var seen uintptr
	for time.Now().Before(deadline) {
		hwnd := hwndForPID(uint32(pid))
		if hwnd != 0 {
			forceForeground(hwnd)
			if hwnd == seen {
				return
			}
			seen = hwnd
		}
		time.Sleep(120 * time.Millisecond)
	}
}

func placeWindow(hwnd uintptr, r winRect) {
	w := r.Right - r.Left
	h := r.Bottom - r.Top
	if w < 200 {
		w = 980
	}
	if h < 200 {
		h = 620
	}
	procSetWindowPos.Call(hwnd, 0, uintptr(r.Left), uintptr(r.Top), uintptr(w), uintptr(h), swpNoZOrder|swpShowWindow)
}

func workArea(hwnd uintptr) winRect {
	mon, _, _ := procMonitorFromWindow.Call(hwnd, monitorDefaultToNearest)
	if mon != 0 {
		mi := monitorInfo{cbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
		ok, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
		if ok != 0 {
			return mi.rcWork
		}
	}
	var r winRect
	procSystemParametersInfoW.Call(spiGetWorkArea, 0, uintptr(unsafe.Pointer(&r)), 0)
	return r
}

func (a *App) WindowMinimise() {
	hwnd := a.shellHWND()
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(hwnd, swMinimize)
}

func (a *App) WindowToggleMaximise() {
	hwnd := a.shellHWND()
	if hwnd == 0 {
		return
	}
	iconic, _, _ := procIsIconic.Call(hwnd)
	if iconic != 0 {
		procShowWindow.Call(hwnd, swRestore)
	}
	a.winMu.Lock()
	defer a.winMu.Unlock()
	if a.winMax {
		if a.winSaved != nil {
			placeWindow(hwnd, *a.winSaved)
		} else {
			procShowWindow.Call(hwnd, swRestore)
		}
		a.winMax = false
		return
	}
	var cur winRect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&cur)))
	a.winSaved = &cur
	placeWindow(hwnd, workArea(hwnd))
	a.winMax = true
}

func resizeEdgeOK(edge string) bool {
	switch edge {
	case "n", "s", "e", "w", "ne", "nw", "se", "sw":
		return true
	default:
		return false
	}
}

func resizeCursor(edge string) uintptr {
	switch edge {
	case "n", "s":
		return 32645
	case "e", "w":
		return 32644
	case "nw", "se":
		return 32642
	default:
		return 32643
	}
}

func mouseLeftDown() bool {
	state, _, _ := procGetAsyncKeyState.Call(vkLButton)
	return state&0x8000 != 0
}

func applyEdge(edge string, start winRect, dx, dy int32) (x, y, w, h int32) {
	x, y = start.Left, start.Top
	w = start.Right - start.Left
	h = start.Bottom - start.Top
	if strings.Contains(edge, "e") {
		w += dx
	}
	if strings.Contains(edge, "s") {
		h += dy
	}
	if strings.Contains(edge, "w") {
		w -= dx
		x = start.Left + dx
	}
	if strings.Contains(edge, "n") {
		h -= dy
		y = start.Top + dy
	}
	if w < minWinW {
		if strings.Contains(edge, "w") {
			x = start.Right - minWinW
		}
		w = minWinW
	}
	if h < minWinH {
		if strings.Contains(edge, "n") {
			y = start.Bottom - minWinH
		}
		h = minWinH
	}
	return x, y, w, h
}

// WindowStartResize tracks the pointer until the button is released.
// The IDE window is borderless, so Windows has no resize frame to drag.
func (a *App) WindowStartResize(edge string) {
	if !resizeEdgeOK(edge) || !resizeBusy.CompareAndSwap(false, true) {
		return
	}
	defer resizeBusy.Store(false)
	hwnd := a.shellHWND()
	if hwnd == 0 || !mouseLeftDown() {
		return
	}
	var start winRect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&start)))
	var origin winPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&origin)))
	a.winMu.Lock()
	a.winMax = false
	a.winMu.Unlock()
	cursor, _, _ := procLoadCursorW.Call(0, resizeCursor(edge))
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) && mouseLeftDown() {
		alive, _, _ := procIsWindow.Call(hwnd)
		if alive == 0 {
			return
		}
		var cur winPoint
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&cur)))
		x, y, w, h := applyEdge(edge, start, cur.X-origin.X, cur.Y-origin.Y)
		procSetWindowPos.Call(hwnd, 0, uintptr(int(x)), uintptr(int(y)), uintptr(int(w)), uintptr(int(h)), uintptr(swpNoZOrder|swpNoActivate))
		if cursor != 0 {
			procSetCursor.Call(cursor)
		}
		time.Sleep(16 * time.Millisecond)
	}
}

func iabs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// WindowStartMove drags the borderless window from the title bar.
// A short press stays put so a double-click can still maximize.
func (a *App) WindowStartMove() {
	if !resizeBusy.CompareAndSwap(false, true) {
		return
	}
	defer resizeBusy.Store(false)
	hwnd := a.shellHWND()
	if hwnd == 0 || !mouseLeftDown() {
		return
	}
	var start winRect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&start)))
	var origin winPoint
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&origin)))
	a.winMu.Lock()
	maximized := a.winMax
	var saved *winRect
	if a.winSaved != nil {
		c := *a.winSaved
		saved = &c
	}
	a.winMu.Unlock()
	cursor, _, _ := procLoadCursorW.Call(0, 32646)
	deadline := time.Now().Add(2 * time.Minute)
	dragging := false
	for time.Now().Before(deadline) && mouseLeftDown() {
		alive, _, _ := procIsWindow.Call(hwnd)
		if alive == 0 {
			return
		}
		var cur winPoint
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&cur)))
		dx := cur.X - origin.X
		dy := cur.Y - origin.Y
		if !dragging {
			if iabs(dx) < 4 && iabs(dy) < 4 {
				time.Sleep(16 * time.Millisecond)
				continue
			}
			dragging = true
			if maximized {
				w := int32(980)
				h := int32(620)
				if saved != nil {
					w = saved.Right - saved.Left
					h = saved.Bottom - saved.Top
				}
				if w < minWinW {
					w = minWinW
				}
				if h < minWinH {
					h = minWinH
				}
				span := start.Right - start.Left
				rel := int32(0)
				if span > 0 {
					rel = origin.X - start.Left
					if rel < 0 {
						rel = 0
					}
					if rel > span {
						rel = span
					}
					rel = int32(int64(rel) * int64(w) / int64(span))
				}
				nx := cur.X - rel
				ny := cur.Y - 16
				procSetWindowPos.Call(hwnd, 0, uintptr(int(nx)), uintptr(int(ny)), uintptr(int(w)), uintptr(int(h)), uintptr(swpNoZOrder|swpNoActivate))
				start = winRect{Left: nx, Top: ny, Right: nx + w, Bottom: ny + h}
				origin = cur
				dx, dy = 0, 0
				a.winMu.Lock()
				a.winMax = false
				a.winMu.Unlock()
			}
		}
		w := start.Right - start.Left
		h := start.Bottom - start.Top
		procSetWindowPos.Call(hwnd, 0, uintptr(int(start.Left+dx)), uintptr(int(start.Top+dy)), uintptr(int(w)), uintptr(int(h)), uintptr(swpNoZOrder|swpNoActivate))
		if cursor != 0 {
			procSetCursor.Call(cursor)
		}
		time.Sleep(16 * time.Millisecond)
	}
}

func (a *App) WindowClose() {
	hwnd := a.shellHWND()
	if hwnd != 0 {
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
		return
	}
	if a.neuCmd != nil && a.neuCmd.Process != nil {
		_ = a.neuCmd.Process.Kill()
	}
}
