package runtime

import (
	"fmt"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"
	implgl "github.com/AllenDang/cimgui-go/impl/opengl3"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/window"
	"github.com/go-gl/gl/v3.3-core/gl"

	"moonbasic/internal/value"
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
			var flags imgui.WindowFlags
			if argI(a, 1, 0) != 0 {
				flags |= imgui.WindowFlagsMenuBar
			}
			ok := imgui.BeginV(title, nil, flags)
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
			imgui.TextUnformatted(argS(a, 0))
			return z()
		}),
		"guislider": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			label := argS(a, 0)
			lo := float32(argN(a, 1, 0))
			hi := float32(argN(a, 2, 1))
			if hi < lo {
				lo, hi = hi, lo
			}
			p := w.guiSlide[label]
			if p == nil {
				v := float32(argN(a, 3, float64(lo)))
				p = &v
				w.guiSlide[label] = p
			} else if len(a) > 3 {
				*p = float32(argN(a, 3, float64(*p)))
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
			} else if len(a) > 1 {
				*p = argI(a, 1, 0) != 0
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
			} else if len(a) > 1 {
				s := argS(a, 1)
				if s != *p {
					*p = s
				}
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
		"guibeginmenubar": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.BeginMenuBar() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiendmenubar": n(func(a []value.Value) (value.Value, error) {
			imgui.EndMenuBar()
			return z()
		}),
		"guimenu": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.BeginMenu(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiendmenu": n(func(a []value.Value) (value.Value, error) {
			imgui.EndMenu()
			return z()
		}),
		"guimenuitem": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.MenuItemBool(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guibegintabbar": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			id := argS(a, 0)
			if id == "" {
				id = "tabs"
			}
			if imgui.BeginTabBar(id) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiendtabbar": n(func(a []value.Value) (value.Value, error) {
			imgui.EndTabBar()
			return z()
		}),
		"guitabitem": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.BeginTabItem(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiendtabitem": n(func(a []value.Value) (value.Value, error) {
			imgui.EndTabItem()
			return z()
		}),
		"guitree": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.TreeNodeStr(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guitreepop": n(func(a []value.Value) (value.Value, error) {
			imgui.TreePop()
			return z()
		}),
		"guibeginchild": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			id := argS(a, 0)
			if id == "" {
				id = "child"
			}
			if imgui.BeginChildStr(id) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiendchild": n(func(a []value.Value) (value.Value, error) {
			imgui.EndChild()
			return z()
		}),
		"guiopenpopup": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			imgui.OpenPopupStr(argS(a, 0))
			return z()
		}),
		"guibeginpopup": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.BeginPopup(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiendpopup": n(func(a []value.Value) (value.Value, error) {
			imgui.EndPopup()
			return z()
		}),
		"guiselectable": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.SelectableBool(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guidrag": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			label := argS(a, 0)
			lo := float32(argN(a, 1, 0))
			hi := float32(argN(a, 2, 1))
			if hi < lo {
				lo, hi = hi, lo
			}
			p := w.guiDrag[label]
			if p == nil {
				v := float32(argN(a, 3, float64(lo)))
				p = &v
				w.guiDrag[label] = p
			} else if len(a) > 3 {
				*p = float32(argN(a, 3, float64(*p)))
			}
			speed := (hi - lo) / 100
			if speed < 0.001 {
				speed = 0.001
			}
			imgui.DragFloatV(label, p, speed, lo, hi, "%.3f", 0)
			return value.Num(float64(*p)), nil
		}),
		"guiprogress": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			frac := float32(argN(a, 0, 0))
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			imgui.ProgressBarV(frac, imgui.NewVec2(-1, 0), argS(a, 1))
			return value.Num(float64(frac)), nil
		}),
		"guicombo": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			label := argS(a, 0)
			items := strings.Split(argS(a, 1), "|")
			if len(items) == 0 {
				items = []string{""}
			}
			p := w.guiCombo[label]
			if p == nil {
				v := int32(argI(a, 2, 0))
				p = &v
				w.guiCombo[label] = p
			} else if len(a) > 2 {
				*p = int32(argI(a, 2, int(*p)))
			}
			if *p < 0 {
				*p = 0
			}
			if int(*p) >= len(items) {
				*p = int32(len(items) - 1)
			}
			imgui.ComboStrarr(label, p, items, int32(len(items)))
			return value.Num(float64(*p)), nil
		}),
		"guiheader": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.CollapsingHeaderTreeNodeFlags(argS(a, 0)) {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guiradio": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			label := argS(a, 0)
			group := argS(a, 1)
			if group == "" {
				group = label
			}
			id := int32(argI(a, 2, 0))
			p := w.guiRadio[group]
			if p == nil {
				v := id
				if len(a) > 3 {
					v = int32(argI(a, 3, int(id)))
				}
				p = &v
				w.guiRadio[group] = p
			} else if len(a) > 3 {
				*p = int32(argI(a, 3, int(*p)))
			}
			if imgui.RadioButtonBool(label, *p == id) {
				*p = id
			}
			if *p == id {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"guicolor": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			label := argS(a, 0)
			col := w.guiColor[label]
			if col == nil {
				c := [3]float32{1, 1, 1}
				col = &c
				w.guiColor[label] = col
			}
			if len(a) > 1 {
				(*col)[0] = guiByteUnit(argN(a, 1, 255))
			}
			if len(a) > 2 {
				(*col)[1] = guiByteUnit(argN(a, 2, 255))
			}
			if len(a) > 3 {
				(*col)[2] = guiByteUnit(argN(a, 3, 255))
			}
			imgui.ColorEdit3(label, col)
			return value.Num(guiUnitByte((*col)[0])), nil
		}),
		"guicolorg": n(func(a []value.Value) (value.Value, error) {
			col := w.guiColor[argS(a, 0)]
			if col == nil {
				return value.Num(0), nil
			}
			return value.Num(guiUnitByte((*col)[1])), nil
		}),
		"guicolorb": n(func(a []value.Value) (value.Value, error) {
			col := w.guiColor[argS(a, 0)]
			if col == nil {
				return value.Num(0), nil
			}
			return value.Num(guiUnitByte((*col)[2])), nil
		}),
		"guitooltip": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			if imgui.BeginItemTooltip() {
				imgui.TextUnformatted(argS(a, 0))
				imgui.EndTooltip()
			}
			return z()
		}),
		"guibullet": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			imgui.Bullet()
			imgui.TextUnformatted(argS(a, 0))
			return z()
		}),
		"guispacing": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			imgui.Spacing()
			return z()
		}),
		"guisetwidth": n(func(a []value.Value) (value.Value, error) {
			if err := w.guiEnsureFrame(); err != nil {
				return value.Value{}, err
			}
			imgui.SetNextItemWidth(float32(argN(a, 0, -1)))
			return z()
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

func guiByteUnit(n float64) float32 {
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	return float32(n / 255)
}

func guiUnitByte(v float32) float64 {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return float64(int(v*255 + 0.5))
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
	if io := imgui.CurrentIO(); io != nil {
		io.SetConfigFlags(io.ConfigFlags() &^ imgui.ConfigFlagsViewportsEnable)
	}
	// G3N requests a 3.3 core context. The default imgui backend picks
	// "#version 130" below GL 4.1, which does not compile in core profile.
	if !implgl.InitV("#version 330") {
		return fmt.Errorf("Gui*: ImGui OpenGL3 init failed")
	}
	if io := imgui.CurrentIO(); io != nil {
		io.SetConfigFlags(io.ConfigFlags() &^ imgui.ConfigFlagsViewportsEnable)
		io.SetBackendFlags(io.BackendFlags() &^ imgui.BackendFlagsRendererHasViewports)
	}
	w.guiReady = true
	w.guiWarm = 24
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
	sx, sy := float32(1), float32(1)
	if gw := w.glfwWin(); gw != nil {
		if sw, sh := gw.GetSize(); sw > 0 && sh > 0 {
			ww, hh = sw, sh
		}
		if fbW, fbH := gw.GetFramebufferSize(); ww > 0 && hh > 0 && fbW > 0 && fbH > 0 {
			sx = float32(fbW) / float32(ww)
			sy = float32(fbH) / float32(hh)
		}
	}
	io.SetDisplaySize(imgui.NewVec2(float32(ww), float32(hh)))
	io.SetDisplayFramebufferScale(imgui.NewVec2(sx, sy))
	dt := float32(w.delta)
	if dt <= 0 {
		dt = 1.0 / 60.0
	}
	io.SetDeltaTime(dt)
	io.AddMousePosEvent(w.mx, w.my)
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
	if gw := w.glfwWin(); gw != nil {
		gw.MakeContextCurrent()
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	if w.guiWinOpen {
		imgui.End()
		w.guiWinOpen = false
	}
	implgl.NewFrame()
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
	w.guiRestoreSceneGL()
}

// guiRestoreSceneGL puts the context back the way G3N's cached GL state
// expects. ImGui (and its NewFrame device-object upload) enable scissor,
// bind samplers/VAOs, and disable depth without going through GLS, so the
// following RenderWorld pass can draw nothing while the overlay still appears.
func (w *World) guiRestoreSceneGL() {
	if w.app == nil {
		return
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.DrawBuffer(gl.BACK)
	gl.ReadBuffer(gl.BACK)
	gl.Disable(gl.SCISSOR_TEST)
	gl.Disable(gl.STENCIL_TEST)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.BLEND)
	gl.DepthMask(true)
	gl.ColorMask(true, true, true, true)
	gl.DepthFunc(gl.LEQUAL)
	gl.FrontFace(gl.CCW)
	gl.CullFace(gl.BACK)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindSampler(0, 0)
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
	gl.UseProgram(0)
	gs := w.app.Gls()
	if gs == nil {
		return
	}
	gs.Disable(gls.DEPTH_TEST)
	gs.Enable(gls.DEPTH_TEST)
	gs.Disable(gls.CULL_FACE)
	gs.Enable(gls.CULL_FACE)
	gs.Disable(gls.BLEND)
	gs.Enable(gls.BLEND)
	gs.DepthMask(false)
	gs.DepthMask(true)
	gs.DepthFunc(gls.LEQUAL)
	gs.BlendEquation(gls.FUNC_ADD)
	gs.BlendFunc(gls.SRC_ALPHA, gls.ONE_MINUS_SRC_ALPHA)
	gs.FrontFace(gls.CCW)
	gs.CullFace(gls.BACK)
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
	if w.guiPrevKeys == nil {
		w.guiPrevKeys = map[window.Key]bool{}
	}
	w.guiPrevKeys[k] = down
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
	io.AddMousePosEvent(w.mx, w.my)
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
