package runtime

import (
	"github.com/g3n/engine/window"
	"github.com/go-gl/glfw/v3.3/glfw"

	"bitshinbasic/internal/value"
)

func (w *World) fireHook(name string, args ...value.Value) {
	if name == "" || w.runner == nil {
		return
	}
	_, _ = w.runner.CallNamed(name, args)
}

func (w *World) blitzKey(k window.Key) int {
	for code, gk := range dikToKey {
		if gk == k {
			return code
		}
	}
	return 0
}

func (w *World) guiCapturesMouse() bool    { return w.guiWantM }
func (w *World) guiCapturesKeyboard() bool { return w.guiWantK }

func (w *World) windowCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"windowwidth": n(func(a []value.Value) (value.Value, error) {
			id := 0
			if len(a) >= 1 && w.extraOf(argI(a, 0, 0)) != nil {
				id = argI(a, 0, 0)
			}
			ww, _ := w.windowSizeOf(id)
			return value.Num(float64(ww)), nil
		}),
		"windowheight": n(func(a []value.Value) (value.Value, error) {
			id := 0
			if len(a) >= 1 && w.extraOf(argI(a, 0, 0)) != nil {
				id = argI(a, 0, 0)
			}
			_, hh := w.windowSizeOf(id)
			return value.Num(float64(hh)), nil
		}),
		"setwindowsize": n(func(a []value.Value) (value.Value, error) {
			id, ww, hh := 0, argI(a, 0, w.scrW), argI(a, 1, w.scrH)
			if len(a) >= 3 {
				id, ww, hh = argI(a, 0, 0), argI(a, 1, w.scrW), argI(a, 2, w.scrH)
			}
			w.applyWindowSize(id, ww, hh)
			return z()
		}),
		"windowx": n(func(a []value.Value) (value.Value, error) {
			id := 0
			if len(a) >= 1 && (argI(a, 0, 0) == 0 || w.extraOf(argI(a, 0, 0)) != nil) {
				id = argI(a, 0, 0)
			}
			x, _ := w.windowPosOf(id)
			return value.Num(float64(x)), nil
		}),
		"windowy": n(func(a []value.Value) (value.Value, error) {
			id := 0
			if len(a) >= 1 && (argI(a, 0, 0) == 0 || w.extraOf(argI(a, 0, 0)) != nil) {
				id = argI(a, 0, 0)
			}
			_, y := w.windowPosOf(id)
			return value.Num(float64(y)), nil
		}),
		"setwindowpos": n(func(a []value.Value) (value.Value, error) {
			id, x, y := 0, argI(a, 0, 0), argI(a, 1, 0)
			if len(a) >= 3 {
				id, x, y = argI(a, 0, 0), argI(a, 1, 0), argI(a, 2, 0)
			}
			w.applyWindowPos(id, x, y)
			return z()
		}),
		"setwindowtitle": n(func(a []value.Value) (value.Value, error) {
			if len(a) >= 2 && a[0].Kind == value.KindNum {
				w.applyWindowTitle(argI(a, 0, 0), argS(a, 1))
				return z()
			}
			w.applyWindowTitle(0, argS(a, 0))
			return z()
		}),
		"iconifywindow": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				gw.Iconify()
			}
			return z()
		}),
		"maximizewindow": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				gw.Maximize()
			}
			return z()
		}),
		"restorewindow": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				gw.Restore()
			}
			return z()
		}),
		"focuswindow": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				gw.Focus()
			}
			return z()
		}),
		"windowopacity": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				return value.Num(float64(gw.GetOpacity())), nil
			}
			return value.Num(1), nil
		}),
		"setwindowopacity": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				gw.SetOpacity(float32(argN(a, 0, 1)))
			}
			return z()
		}),
		"setswapinterval": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				gw.SetSwapInterval(argI(a, 0, 1))
			}
			return z()
		}),
		"windowshouldclose": n(func(a []value.Value) (value.Value, error) {
			if w.windowWantsClose() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"clipboard": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				return value.Str(gw.GetClipboardString()), nil
			}
			return value.Str(w.clipLocal), nil
		}),
		"setclipboard": n(func(a []value.Value) (value.Value, error) {
			s := argS(a, 0)
			w.clipLocal = s
			if gw := w.glfwWin(); gw != nil {
				gw.SetClipboardString(s)
			}
			return z()
		}),
		"setfullscreen": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				gw.SetFullscreen(argI(a, 0, 1) != 0)
			}
			return z()
		}),
		"monitorwidth": n(func(a []value.Value) (value.Value, error) {
			m := glfw.GetPrimaryMonitor()
			if m == nil {
				return value.Num(0), nil
			}
			vm := m.GetVideoMode()
			if vm == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(vm.Width)), nil
		}),
		"monitorheight": n(func(a []value.Value) (value.Value, error) {
			m := glfw.GetPrimaryMonitor()
			if m == nil {
				return value.Num(0), nil
			}
			vm := m.GetVideoMode()
			if vm == nil {
				return value.Num(0), nil
			}
			return value.Num(float64(vm.Height)), nil
		}),
	}
}

func (w *World) inputCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"setrawmouse": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				on := glfw.False
				if argI(a, 0, 1) != 0 {
					on = glfw.True
				}
				gw.SetInputMode(glfw.RawMouseMotion, on)
			}
			return z()
		}),
		"rawmouse": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				if gw.GetInputMode(glfw.RawMouseMotion) == glfw.True {
					return value.Num(1), nil
				}
			}
			return value.Num(0), nil
		}),
		"setcursormode": n(func(a []value.Value) (value.Value, error) {
			if gw := w.glfwWin(); gw != nil {
				mode := glfw.CursorNormal
				switch argI(a, 0, 0) {
				case 1:
					mode = glfw.CursorHidden
				case 2:
					mode = glfw.CursorDisabled
				}
				gw.SetInputMode(glfw.CursorMode, mode)
			}
			return z()
		}),
		"gamepadpresent": n(func(a []value.Value) (value.Value, error) {
			if padJoystick(argI(a, 0, 0)).IsGamepad() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"gamepadname": n(func(a []value.Value) (value.Value, error) {
			js := padJoystick(argI(a, 0, 0))
			if !js.IsGamepad() {
				return value.Str(""), nil
			}
			return value.Str(js.GetGamepadName()), nil
		}),
		"gamepadaxis": n(func(a []value.Value) (value.Value, error) {
			st := padState(argI(a, 0, 0))
			if st == nil {
				return value.Num(0), nil
			}
			i := argI(a, 1, 0)
			if i < 0 || i >= len(st.Axes) {
				return value.Num(0), nil
			}
			v := float64(st.Axes[i])
			if v > -w.gpDead && v < w.gpDead {
				v = 0
			}
			return value.Num(v), nil
		}),
		"gamepadbutton": n(func(a []value.Value) (value.Value, error) {
			st := padState(argI(a, 0, 0))
			if st == nil {
				return value.Num(0), nil
			}
			i := argI(a, 1, 0)
			if i < 0 || i >= len(st.Buttons) {
				return value.Num(0), nil
			}
			if st.Buttons[i] == glfw.Press {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"joystickpresent": n(func(a []value.Value) (value.Value, error) {
			if padJoystick(argI(a, 0, 0)).Present() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"setgamepaddeadzone": n(func(a []value.Value) (value.Value, error) {
			w.gpDead = argN(a, 0, 0.15)
			return value.Num(w.gpDead), nil
		}),
		"gamepaddeadzone": n(func(a []value.Value) (value.Value, error) {
			if w.gpDead <= 0 {
				return value.Num(0.15), nil
			}
			return value.Num(w.gpDead), nil
		}),
		"mouselook": n(func(a []value.Value) (value.Value, error) {
			return w.mouseLook(a)
		}),
	}
}

func (w *World) mouseLook(a []value.Value) (value.Value, error) {
	var e *Entity
	if len(a) >= 1 && argI(a, 0, 0) != 0 {
		ent, err := w.ent(argI(a, 0, 0))
		if err != nil {
			return value.Value{}, err
		}
		e = ent
	} else {
		for _, ent := range w.ents {
			if ent != nil && ent.cam != nil && ent.cam == w.cam {
				e = ent
				break
			}
		}
	}
	if e == nil || e.cam == nil {
		return value.Num(0), nil
	}
	sens := argN(a, 1, 0.15)
	pmin := argN(a, 2, -80)
	pmax := argN(a, 3, 80)
	e.yaw += w.mxs * float32(sens)
	e.pitch -= w.mys * float32(sens)
	if e.pitch > float32(pmax) {
		e.pitch = float32(pmax)
	}
	if e.pitch < float32(pmin) {
		e.pitch = float32(pmin)
	}
	w.applyRot(e)
	return value.Num(0), nil
}

func (w *World) armFreeLook() {
	if gw := w.glfwWin(); gw != nil {
		gw.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		gw.SetInputMode(glfw.RawMouseMotion, glfw.True)
	}
}

func (w *World) updateFreeLook(a []value.Value) (value.Value, error) {
	id := argI(a, 0, 0)
	if id == 0 && w.listenEnt != 0 {
		id = w.listenEnt
	}
	speed := argN(a, 1, 8)
	if _, err := w.mouseLook([]value.Value{value.Num(float64(id)), value.Num(0.12), value.Num(-85), value.Num(85)}); err != nil {
		return value.Value{}, err
	}
	n, err := w.nodeOf(id)
	if err != nil {
		return value.Value{}, err
	}
	dt := w.delta
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	spd := speed * dt
	if w.keyDown(KeyW) {
		n.TranslateZ(-float32(spd))
	}
	if w.keyDown(KeyS) {
		n.TranslateZ(float32(spd))
	}
	if w.keyDown(KeyA) {
		n.TranslateX(-float32(spd))
	}
	if w.keyDown(KeyD) {
		n.TranslateX(float32(spd))
	}
	return value.Num(float64(id)), nil
}

func padJoystick(i int) glfw.Joystick {
	if i < 0 {
		i = 0
	}
	return glfw.Joystick(int(glfw.Joystick1) + i)
}

func padState(i int) *glfw.GamepadState {
	js := padJoystick(i)
	if !js.IsGamepad() {
		return nil
	}
	return js.GetGamepadState()
}
