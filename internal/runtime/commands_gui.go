package runtime

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"
	implgl "github.com/AllenDang/cimgui-go/impl/opengl3"
	"github.com/g3n/engine/window"

	"bitshinbasic/internal/value"
)

func (w *World) guiCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"guibegin": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if w.guiWinOpen {
				imgui.End()
			}
			title := argS(a, 0)
			if title == "" {
				title = "Panel"
			}
			ok := imgui.Begin(title)
			w.guiWinOpen = true
			if ok {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiend": n(func(a []value.Value) (value.Value, error) {
			if w.guiWinOpen {
				imgui.End()
				w.guiWinOpen = false
			}
			return z()
		}),
		"guibutton": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.Button(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guitext": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			imgui.Text(argS(a, 0))
			return z()
		}),
		"guislider": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			label := argS(a, 0)
			lo := float32(argN(a, 1, 0))
			hi := float32(argN(a, 2, 1))
			p := w.guiSlide[label]
			if p == nil {
				v := float32(argN(a, 3, float64(lo)))
				p = &v
				w.guiSlide[label] = p
			}
			imgui.SliderFloat(label, p, lo, hi)
			return value.Num(float64(*p)), nil
		}),
		"guicheckbox": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			label := argS(a, 0)
			p := w.guiCheck[label]
			if p == nil {
				on := argI(a, 1, 0) != 0
				p = &on
				w.guiCheck[label] = p
			}
			imgui.Checkbox(label, p)
			if *p {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiinputtext": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			label := argS(a, 0)
			p := w.guiInput[label]
			if p == nil {
				s := argS(a, 1)
				p = &s
				w.guiInput[label] = p
			}
			imgui.InputTextWithHint(label, "", p, 0, nil)
			return value.Str(*p), nil
		}),
		"guisameline": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			imgui.SameLine()
			return z()
		}),
		"guiseparator": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			imgui.Separator()
			return z()
		}),
		"wantcapturemouse": n(func(a []value.Value) (value.Value, error) {
			if w.guiWantM {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"wantcapturekeyboard": n(func(a []value.Value) (value.Value, error) {
			if w.guiWantK {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guidemo": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			open := true
			imgui.ShowDemoWindowV(&open)
			return z()
		}),
	}
}

func (w *World) guiInit() error {
	if w.guiReady {
		return nil
	}
	if err := w.require3D(); err != nil {
		return fmt.Errorf("Gui*: %w", err)
	}
	if w.glfwWin() == nil {
		return fmt.Errorf("Gui*: Graphics3D GLFW window is missing")
	}
	imgui.CreateContext()
	imgui.StyleColorsDark()
	if !implgl.Init() {
		return fmt.Errorf("Gui*: ImGui OpenGL3 init failed")
	}
	w.guiReady = true
	return nil
}

func (w *World) guiEnsureFrame() error {
	if err := w.guiInit(); err != nil {
		return err
	}
	if w.guiFrame {
		w.guiUsed = true
		return nil
	}
	w.guiPumpIO()
	implgl.NewFrame()
	imgui.NewFrame()
	w.guiFrame = true
	w.guiUsed = true
	return nil
}

func (w *World) guiPumpIO() {
	io := imgui.CurrentIO()
	if io == nil {
		return
	}
	ww, hh := w.scrW, w.scrH
	if w.app != nil {
		ww, hh = w.app.GetSize()
	}
	io.SetDisplaySize(imgui.NewVec2(float32(ww), float32(hh)))
	dt := float32(w.delta)
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	io.SetDeltaTime(dt)
	io.AddMousePosEvent(w.mx, w.my)
	io.AddMouseButtonEvent(0, len(w.mouse) > 1 && w.mouse[1])
	io.AddMouseButtonEvent(1, len(w.mouse) > 2 && w.mouse[2])
	io.AddMouseButtonEvent(2, len(w.mouse) > 3 && w.mouse[3])
	if w.mzs != 0 {
		io.AddMouseWheelEvent(0, w.mzs)
	}
	for _, ch := range w.guiChars {
		io.AddInputCharacter(uint32(ch))
	}
	w.guiChars = w.guiChars[:0]
	if w.guiPrevKeys == nil {
		w.guiPrevKeys = map[window.Key]bool{}
	}
	seen := map[window.Key]bool{}
	for code, down := range w.keys {
		gk, ok := dikToKey[code]
		if !ok {
			continue
		}
		seen[gk] = true
		if w.guiPrevKeys[gk] != down {
			if ik := imguiFromWindowKey(gk); ik != imgui.KeyNone {
				io.AddKeyEvent(ik, down)
			}
			w.guiPrevKeys[gk] = down
		}
	}
	for gk, was := range w.guiPrevKeys {
		if was && !seen[gk] {
			if ik := imguiFromWindowKey(gk); ik != imgui.KeyNone {
				io.AddKeyEvent(ik, false)
			}
			w.guiPrevKeys[gk] = false
		}
	}
}

func (w *World) guiPresent() {
	if !w.guiReady || !w.guiFrame {
		w.guiWantM = false
		w.guiWantK = false
		return
	}
	if w.guiWinOpen {
		imgui.End()
		w.guiWinOpen = false
	}
	imgui.Render()
	if dd := imgui.CurrentDrawData(); dd != nil {
		implgl.RenderDrawData(dd)
	}
	if io := imgui.CurrentIO(); io != nil {
		w.guiWantM = io.WantCaptureMouse()
		w.guiWantK = io.WantCaptureKeyboard()
	}
	w.guiFrame = false
	w.guiUsed = false
}

func (w *World) guiForwardKey(k window.Key, mods window.ModifierKey, down bool) {
	if !w.guiReady {
		return
	}
	io := imgui.CurrentIO()
	if io == nil {
		return
	}
	if ik := imguiFromWindowKey(k); ik != imgui.KeyNone {
		io.AddKeyEvent(ik, down)
	}
	_ = mods
}

func (w *World) guiForwardMouse(button int, down bool) {
	if !w.guiReady {
		return
	}
	io := imgui.CurrentIO()
	if io == nil {
		return
	}
	io.AddMouseButtonEvent(int32(button), down)
}

func (w *World) guiForwardScroll(x, y float64) {
	if !w.guiReady {
		return
	}
	io := imgui.CurrentIO()
	if io == nil {
		return
	}
	io.AddMouseWheelEvent(float32(x), float32(y))
}

func imguiFromWindowKey(k window.Key) imgui.Key {
	switch k {
	case window.KeyTab:
		return imgui.KeyTab
	case window.KeyLeft:
		return imgui.KeyLeftArrow
	case window.KeyRight:
		return imgui.KeyRightArrow
	case window.KeyUp:
		return imgui.KeyUpArrow
	case window.KeyDown:
		return imgui.KeyDownArrow
	case window.KeyPageUp:
		return imgui.KeyPageUp
	case window.KeyPageDown:
		return imgui.KeyPageDown
	case window.KeyHome:
		return imgui.KeyHome
	case window.KeyEnd:
		return imgui.KeyEnd
	case window.KeyInsert:
		return imgui.KeyInsert
	case window.KeyDelete:
		return imgui.KeyDelete
	case window.KeyBackspace:
		return imgui.KeyBackspace
	case window.KeySpace:
		return imgui.KeySpace
	case window.KeyEnter:
		return imgui.KeyEnter
	case window.KeyEscape:
		return imgui.KeyEscape
	case window.KeyLeftControl:
		return imgui.KeyLeftCtrl
	case window.KeyLeftShift:
		return imgui.KeyLeftShift
	case window.KeyLeftAlt:
		return imgui.KeyLeftAlt
	case window.KeyRightControl:
		return imgui.KeyRightCtrl
	case window.KeyRightShift:
		return imgui.KeyRightShift
	case window.KeyRightAlt:
		return imgui.KeyRightAlt
	case window.KeyA:
		return imgui.KeyA
	case window.KeyB:
		return imgui.KeyB
	case window.KeyC:
		return imgui.KeyC
	case window.KeyD:
		return imgui.KeyD
	case window.KeyE:
		return imgui.KeyE
	case window.KeyF:
		return imgui.KeyF
	case window.KeyG:
		return imgui.KeyG
	case window.KeyH:
		return imgui.KeyH
	case window.KeyI:
		return imgui.KeyI
	case window.KeyJ:
		return imgui.KeyJ
	case window.KeyK:
		return imgui.KeyK
	case window.KeyL:
		return imgui.KeyL
	case window.KeyM:
		return imgui.KeyM
	case window.KeyN:
		return imgui.KeyN
	case window.KeyO:
		return imgui.KeyO
	case window.KeyP:
		return imgui.KeyP
	case window.KeyQ:
		return imgui.KeyQ
	case window.KeyR:
		return imgui.KeyR
	case window.KeyS:
		return imgui.KeyS
	case window.KeyT:
		return imgui.KeyT
	case window.KeyU:
		return imgui.KeyU
	case window.KeyV:
		return imgui.KeyV
	case window.KeyW:
		return imgui.KeyW
	case window.KeyX:
		return imgui.KeyX
	case window.KeyY:
		return imgui.KeyY
	case window.KeyZ:
		return imgui.KeyZ
	case window.Key0:
		return imgui.Key0
	case window.Key1:
		return imgui.Key1
	case window.Key2:
		return imgui.Key2
	case window.Key3:
		return imgui.Key3
	case window.Key4:
		return imgui.Key4
	case window.Key5:
		return imgui.Key5
	case window.Key6:
		return imgui.Key6
	case window.Key7:
		return imgui.Key7
	case window.Key8:
		return imgui.Key8
	case window.Key9:
		return imgui.Key9
	default:
		return imgui.KeyNone
	}
}
