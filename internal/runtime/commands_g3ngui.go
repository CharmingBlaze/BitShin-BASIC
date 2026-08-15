package runtime

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gui"

	"bitshinbasic/internal/value"
)

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
