// Package value is the dynamic BASIC value (number, string, array).
package value

import (
	"fmt"
	"strconv"
	"strings"
)

type Kind int

const (
	KindNum Kind = iota
	KindStr
	KindArray
	KindStruct
	KindVec
)

type Value struct {
	Kind     Kind
	Num      float64
	Str      string
	Elems    []Value
	Dims     []int // Dim arguments (each is inclusive 0..n)
	TypeName string
	Fields   map[string]Value
}

func Num(n float64) Value { return Value{Kind: KindNum, Num: n} }
func Str(s string) Value  { return Value{Kind: KindStr, Str: s} }

// Vec3 is a lightweight [x, y, z] value used by Position / Color literals.
func Vec3(x, y, z float64) Value {
	return Value{Kind: KindVec, Elems: []Value{Num(x), Num(y), Num(z)}}
}

func Vec(elems []Value) Value {
	return Value{Kind: KindVec, Elems: append([]Value(nil), elems...)}
}

// XYZ unpacks a vec literal (or a 3+ element array).
func (v Value) XYZ() (x, y, z float64, ok bool) {
	if v.Kind != KindVec && v.Kind != KindArray {
		return 0, 0, 0, false
	}
	if len(v.Elems) < 2 {
		return 0, 0, 0, false
	}
	x = v.Elems[0].Number()
	y = v.Elems[1].Number()
	if len(v.Elems) >= 3 {
		z = v.Elems[2].Number()
	}
	return x, y, z, true
}

func StructOf(typeName string, fields []string) Value {
	m := make(map[string]Value, len(fields))
	for _, f := range fields {
		m[strings.ToLower(strings.TrimRight(f, "$%#"))] = Num(0)
	}
	return Value{Kind: KindStruct, TypeName: strings.ToLower(typeName), Fields: m}
}

func (v Value) Clone() Value {
	if v.Kind != KindStruct {
		return v
	}
	f := make(map[string]Value, len(v.Fields))
	for k, x := range v.Fields {
		f[k] = x.Clone()
	}
	return Value{Kind: KindStruct, TypeName: v.TypeName, Fields: f}
}

func (v Value) Field(name string) (Value, bool) {
	if v.Kind != KindStruct || v.Fields == nil {
		return Value{}, false
	}
	x, ok := v.Fields[strings.ToLower(strings.TrimRight(name, "$%#"))]
	return x, ok
}

func (v Value) SetField(name string, x Value) bool {
	if v.Kind != KindStruct || v.Fields == nil {
		return false
	}
	v.Fields[strings.ToLower(strings.TrimRight(name, "$%#"))] = x
	return true
}

func Array(sizes []int) Value {
	n := 1
	for _, s := range sizes {
		if s < 0 {
			s = 0
		}
		n *= s + 1 // Blitz Dim x(10) is 0..10
	}
	dims := append([]int(nil), sizes...)
	return Value{Kind: KindArray, Elems: make([]Value, n), Num: float64(len(sizes)), Dims: dims}
}

// Offset is the flat index for Dim-style coordinates (0..size inclusive per dim).
func (v Value) Offset(idx []int) (int, bool) {
	if v.Kind != KindArray {
		return 0, false
	}
	dims := v.Dims
	if len(dims) == 0 {
		if len(idx) < 1 {
			return 0, false
		}
		i := idx[0]
		if i < 0 || i >= len(v.Elems) {
			return 0, false
		}
		return i, true
	}
	off, stride := 0, 1
	for i, d := range dims {
		n := d + 1
		x := 0
		if i < len(idx) {
			x = idx[i]
		}
		if x < 0 || x >= n {
			return 0, false
		}
		off += x * stride
		stride *= n
	}
	if off < 0 || off >= len(v.Elems) {
		return 0, false
	}
	return off, true
}

func (v Value) DimSize(dim int) int {
	if v.Kind != KindArray {
		return 0
	}
	if dim < 0 {
		return len(v.Elems)
	}
	if dim >= len(v.Dims) {
		if dim == 0 {
			return len(v.Elems) - 1
		}
		return 0
	}
	return v.Dims[dim]
}

func (v Value) IsTrue() bool {
	switch v.Kind {
	case KindStr:
		return v.Str != "" && v.Str != "0"
	case KindArray:
		return len(v.Elems) > 0
	case KindStruct:
		return true
	case KindVec:
		return len(v.Elems) > 0
	default:
		return v.Num != 0
	}
}

func (v Value) Number() float64 {
	if v.Kind == KindStr {
		n, _ := strconv.ParseFloat(strings.TrimSpace(v.Str), 64)
		return n
	}
	return v.Num
}

func (v Value) Int() int { return int(v.Number()) }

func (v Value) String() string {
	switch v.Kind {
	case KindStr:
		return v.Str
	case KindArray:
		return fmt.Sprintf("Array(%d)", len(v.Elems))
	case KindStruct:
		return v.TypeName
	case KindVec:
		parts := make([]string, len(v.Elems))
		for i, e := range v.Elems {
			parts[i] = e.String()
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		if v.Num == float64(int64(v.Num)) {
			return strconv.FormatInt(int64(v.Num), 10)
		}
		return strconv.FormatFloat(v.Num, 'f', -1, 64)
	}
}

func (v Value) Repr() string { return v.String() }

func Add(a, b Value) Value {
	if a.Kind == KindStr || b.Kind == KindStr {
		return Str(a.String() + b.String())
	}
	return Num(a.Number() + b.Number())
}

func Cmp(a, b Value) int {
	if a.Kind == KindStruct || b.Kind == KindStruct {
		if a.Kind != b.Kind || a.TypeName != b.TypeName {
			return strings.Compare(a.String(), b.String())
		}
		return 0
	}
	if a.Kind == KindStr || b.Kind == KindStr {
		return strings.Compare(a.String(), b.String())
	}
	d := a.Number() - b.Number()
	if d < 0 {
		return -1
	}
	if d > 0 {
		return 1
	}
	return 0
}
