package runtime

import (
	"fmt"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gui"

	"moonbasic/internal/value"
)

type uiTween struct {
	id       int
	sx, sy   float32
	x, y     float32
	t, dur   float64
}

// Scene-graph UI (G3N gui package). Names do not clash with Dear ImGui:
// ImGui keeps GuiBegin / GuiButton / GuiSlider; these are CreatePanel,
// CreateButton, CreateLabel, CreateSlider, CreateCheckbox, CreateEdit.

func (w *World) g3nGuiCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error), need func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"createpanel": need(func(a []value.Value) (value.Value, error) {
			p := gui.NewPanel(float32(argN(a, 0, 200)), float32(argN(a, 1, 120)))
			id := w.addEntity(&Entity{node: p}, argI(a, 2, 0))
			return value.Num(float64(id)), nil
		}),
		"createbutton": need(func(a []value.Value) (value.Value, error) {
			b := gui.NewButton(argS(a, 0))
			id := w.addEntity(&Entity{node: b}, argI(a, 1, 0))
			return value.Num(float64(id)), nil
		}),
		"createlabel": need(func(a []value.Value) (value.Value, error) {
			l := gui.NewLabel(argS(a, 0))
			id := w.addEntity(&Entity{node: l}, argI(a, 1, 0))
			return value.Num(float64(id)), nil
		}),
		"createslider": need(func(a []value.Value) (value.Value, error) {
			s := gui.NewHSlider(float32(argN(a, 0, 160)), float32(argN(a, 1, 24)))
			s.SetScaleFactor(1)
			id := w.addEntity(&Entity{node: s}, argI(a, 2, 0))
			return value.Num(float64(id)), nil
		}),
		"createcheckbox": need(func(a []value.Value) (value.Value, error) {
			c := gui.NewCheckBox(argS(a, 0))
			id := w.addEntity(&Entity{node: c}, argI(a, 1, 0))
			return value.Num(float64(id)), nil
		}),
		"createedit": need(func(a []value.Value) (value.Value, error) {
			ed := gui.NewEdit(argI(a, 0, 160), "")
			id := w.addEntity(&Entity{node: ed}, argI(a, 1, 0))
			return value.Num(float64(id)), nil
		}),
		"setonclick": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			fn := argS(a, 1)
			e.onClick = fn
			id := argI(a, 0, 0)
			w.subscribeWidget(e.node, gui.OnClick, func() {
				w.fireHook(fn, value.Num(float64(id)))
			})
			return z()
		}),
		"setonchange": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			fn := argS(a, 1)
			e.onChange = fn
			id := argI(a, 0, 0)
			w.subscribeWidget(e.node, gui.OnChange, func() {
				w.fireHook(fn, value.Num(float64(id)), w.widgetValue(e))
			})
			return z()
		}),
		"setwidgettext": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			s := argS(a, 1)
			switch t := e.node.(type) {
			case *gui.Button:
				t.Label.SetText(s)
			case *gui.Label:
				t.SetText(s)
			case *gui.Edit:
				t.SetText(s)
			case *gui.CheckRadio:
				t.Label.SetText(s)
			}
			return z()
		}),
		"widgettext": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			switch t := e.node.(type) {
			case *gui.Button:
				return value.Str(t.Label.Text()), nil
			case *gui.Label:
				return value.Str(t.Text()), nil
			case *gui.Edit:
				return value.Str(t.Text()), nil
			case *gui.CheckRadio:
				return value.Str(t.Label.Text()), nil
			}
			return value.Str(""), nil
		}),
		"setwidgetvalue": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			v := argN(a, 1, 0)
			switch t := e.node.(type) {
			case *gui.Slider:
				t.SetValue(float32(v))
			case *gui.CheckRadio:
				t.SetValue(v != 0)
			}
			return z()
		}),
		"widgetvalue": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			return w.widgetValue(e), nil
		}),
		"tweenwidget": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			e, err := w.ent(id)
			if err != nil {
				return value.Value{}, err
			}
			s, ok := e.node.(sizedWidget)
			if !ok {
				return value.Num(0), fmt.Errorf("TweenWidget: not a widget")
			}
			tw := uiTween{
				id: id, x: float32(argN(a, 1, 0)), y: float32(argN(a, 2, 0)),
				dur: argN(a, 3, 0),
			}
			p := e.node.GetNode().Position()
			tw.sx, tw.sy = p.X, p.Y
			if tw.dur <= 0 {
				s.SetPosition(tw.x, tw.y)
				return value.Num(1), nil
			}
			w.widgetTweens = append(w.widgetTweens, tw)
			return value.Num(1), nil
		}),
		"setwidgetanchor": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if _, err := w.ent(id); err != nil {
				return value.Value{}, err
			}
			if w.anchors == nil {
				w.anchors = map[int][2]float64{}
			}
			w.anchors[id] = [2]float64{argN(a, 1, 0), argN(a, 2, 0)}
			w.applyWidgetAnchors()
			return z()
		}),
		"setwidgetpos": need(func(a []value.Value) (value.Value, error) {
			e, err := w.ent(argI(a, 0, 0))
			if err != nil {
				return value.Value{}, err
			}
			if p, ok := e.node.(interface{ SetPosition(x, y float32) }); ok {
				p.SetPosition(float32(argN(a, 1, 0)), float32(argN(a, 2, 0)))
			}
			return z()
		}),
		"g3nbutton": need(func(a []value.Value) (value.Value, error) {
			return w.g3nGuiCommands(n, z, need)["createbutton"](a)
		}),
	}
}

type sizedWidget interface {
	SetPosition(x, y float32)
	Width() float32
	Height() float32
}

func (w *World) tickUITweens() {
	if len(w.widgetTweens) == 0 {
		return
	}
	dt := w.delta
	if dt <= 0 {
		dt = 1.0 / 60
	}
	cur := append([]uiTween(nil), w.widgetTweens...)
	left := make([]uiTween, 0, len(cur))
	for _, tw := range cur {
		e := w.ents[tw.id]
		if e == nil || e.node == nil {
			continue
		}
		widget, ok := e.node.(sizedWidget)
		if !ok {
			continue
		}
		tw.t += dt
		u := tw.t / tw.dur
		if u > 1 {
			u = 1
		}
		widget.SetPosition(tw.sx+float32(u)*(tw.x-tw.sx), tw.sy+float32(u)*(tw.y-tw.sy))
		if u < 1 {
			left = append(left, tw)
		}
	}
	w.widgetTweens = left
}

func (w *World) applyWidgetAnchors() {
	pw, ph := float32(800), float32(600)
	if w.app != nil {
		ww, hh := w.app.GetSize()
		if ww > 0 && hh > 0 {
			pw, ph = float32(ww), float32(hh)
		}
	}
	for id, a := range w.anchors {
		e := w.ents[id]
		if e == nil || e.node == nil {
			continue
		}
		s, ok := e.node.(sizedWidget)
		if !ok {
			continue
		}
		ax, ay := float32(a[0]), float32(a[1])
		s.SetPosition(ax*(pw-s.Width()), ay*(ph-s.Height()))
	}
}

func (w *World) widgetValue(e *Entity) value.Value {
	switch t := e.node.(type) {
	case *gui.Slider:
		return value.Num(float64(t.Value()))
	case *gui.CheckRadio:
		if t.Value() {
			return value.Num(1)
		}
		return value.Num(0)
	}
	return value.Num(0)
}

func (w *World) subscribeWidget(node core.INode, ev string, fn func()) {
	type subber interface {
		Subscribe(string, core.Callback)
	}
	if s, ok := node.(subber); ok {
		s.Subscribe(ev, func(string, interface{}) { fn() })
	}
}
