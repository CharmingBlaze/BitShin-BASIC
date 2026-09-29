package interp

import (
	"bitshinbasic/internal/ast"
	"bitshinbasic/internal/lex"
	"bitshinbasic/internal/value"
)

type stateMachine struct {
	cur string
	fn  map[string]string
}

type opCode byte

const (
	opPush opCode = iota
	opLoad
	opStore
	opAdd
	opSub
	opMul
	opDiv
	opNeg
	opRet
)

type instr struct {
	op  opCode
	imm float64
	ix  int
}

type fastCode struct {
	ok    bool
	ins   []instr
	slots int
}

func (in *Interp) compiled(fn *ast.FuncDecl) (fastCode, bool) {
	key := lex.IdentKey(fn.Name)
	if c, ok := in.fastCache[key]; ok {
		return c, c.ok
	}
	c := compileFast(fn)
	if in.fastCache == nil {
		in.fastCache = map[string]fastCode{}
	}
	in.fastCache[key] = c
	return c, c.ok
}

func (in *Interp) runFast(fn *ast.FuncDecl, code fastCode, args []value.Value) value.Value {
	in.fastHits++
	slots := make([]float64, code.slots)
	for i := range fn.Params {
		if i < len(args) && i < len(slots) {
			slots[i] = args[i].Number()
		}
	}
	stack := make([]float64, 0, 8)
	push := func(n float64) { stack = append(stack, n) }
	pop := func() float64 {
		if len(stack) == 0 {
			return 0
		}
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return n
	}
	for _, ins := range code.ins {
		switch ins.op {
		case opPush:
			push(ins.imm)
		case opLoad:
			if ins.ix >= 0 && ins.ix < len(slots) {
				push(slots[ins.ix])
			} else {
				push(0)
			}
		case opStore:
			if ins.ix >= 0 && ins.ix < len(slots) {
				slots[ins.ix] = pop()
			}
		case opAdd:
			b, a := pop(), pop()
			push(a + b)
		case opSub:
			b, a := pop(), pop()
			push(a - b)
		case opMul:
			b, a := pop(), pop()
			push(a * b)
		case opDiv:
			b, a := pop(), pop()
			if b == 0 {
				push(0)
			} else {
				push(a / b)
			}
		case opNeg:
			push(-pop())
		case opRet:
			return value.Num(pop())
		}
	}
	return value.Num(0)
}

func compileFast(fn *ast.FuncDecl) fastCode {
	slots := map[string]int{}
	for _, p := range fn.Params {
		slots[lex.IdentKey(p)] = len(slots)
	}
	var ins []instr
	var compileExpr func(ast.Expr) bool
	compileExpr = func(e ast.Expr) bool {
		switch t := e.(type) {
		case *ast.NumberExpr:
			ins = append(ins, instr{op: opPush, imm: t.Value})
			return true
		case *ast.IdentExpr:
			ix, ok := slots[lex.IdentKey(t.Name)]
			if !ok {
				return false
			}
			ins = append(ins, instr{op: opLoad, ix: ix})
			return true
		case *ast.UnaryExpr:
			if t.Op != "-" {
				return false
			}
			if !compileExpr(t.X) {
				return false
			}
			ins = append(ins, instr{op: opNeg})
			return true
		case *ast.BinaryExpr:
			if !compileExpr(t.Left) || !compileExpr(t.Right) {
				return false
			}
			var op opCode
			switch t.Op {
			case "+":
				op = opAdd
			case "-":
				op = opSub
			case "*":
				op = opMul
			case "/":
				op = opDiv
			default:
				return false
			}
			ins = append(ins, instr{op: op})
			return true
		default:
			return false
		}
	}
	slotOf := func(name string) int {
		k := lex.IdentKey(name)
		if ix, ok := slots[k]; ok {
			return ix
		}
		ix := len(slots)
		slots[k] = ix
		return ix
	}
	for _, s := range fn.Body {
		switch t := s.(type) {
		case *ast.LocalStmt:
			for i, name := range t.Names {
				ix := slotOf(name)
				if i < len(t.Values) && t.Values[i] != nil {
					if !compileExpr(t.Values[i]) {
						return fastCode{}
					}
					ins = append(ins, instr{op: opStore, ix: ix})
				}
			}
		case *ast.AssignStmt:
			if len(t.Fields) > 0 || len(t.Index) > 0 || len(t.Names) > 1 || t.Value == nil {
				return fastCode{}
			}
			if !compileExpr(t.Value) {
				return fastCode{}
			}
			ins = append(ins, instr{op: opStore, ix: slotOf(t.Name)})
		case *ast.ReturnStmt:
			if t.Value == nil {
				ins = append(ins, instr{op: opPush, imm: 0}, instr{op: opRet})
				continue
			}
			if !compileExpr(t.Value) {
				return fastCode{}
			}
			ins = append(ins, instr{op: opRet})
		default:
			return fastCode{}
		}
	}
	if len(ins) == 0 {
		return fastCode{}
	}
	return fastCode{ok: true, ins: ins, slots: len(slots)}
}
