// Package interp walks the AST. Flip yields so the graphics backend can present a frame.
package interp

import (
	"bufio"
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"bitshinbasic/internal/ast"
	"bitshinbasic/internal/lex"
	"bitshinbasic/internal/syntax"
	"bitshinbasic/internal/value"
)

type Status int

const (
	StatusOK Status = iota
	StatusYield
	StatusWaitYield
	StatusEnd
	StatusReturn
	StatusExit
)

type Builtin func(args []value.Value) (value.Value, error)

type Host interface {
	Call(name string, args []value.Value) (value.Value, error)
	Yields(name string) bool
	Holds() bool
}

type frameKind int

const (
	frameBlock frameKind = iota
	frameWhile
	frameFor
	frameRepeat
	frameFunc
	frameTry
)

type frame struct {
	kind    frameKind
	stmts   []ast.Stmt
	pc      int
	cond    ast.Expr
	forVar  string
	end     ast.Expr
	step    ast.Expr
	forEach []value.Value
	forAt   int
	until   ast.Expr
	env     *env
	ret     value.Value
	catch   []ast.Stmt
	errName string
}

type env struct {
	parent      *env
	vars        map[string]value.Value
	declared    map[string]bool
	kinds       map[string]value.Kind
	ints        map[string]bool
	funcScope   bool
	globalAlias map[string]bool // Global name inside a function: assign the global
}

func newEnv(parent *env) *env {
	return &env{parent: parent, vars: map[string]value.Value{}}
}

func (e *env) get(name string) (value.Value, bool) {
	k := lex.IdentKey(name)
	if v, ok := e.vars[k]; ok {
		return v, true
	}
	if e.parent != nil {
		return e.parent.get(k)
	}
	return value.Value{}, false
}

func (e *env) set(name string, v value.Value) {
	k := lex.IdentKey(name)
	v = e.coerce(k, v)
	if _, ok := e.vars[k]; ok {
		e.vars[k] = v
		return
	}
	if e.globalAlias != nil && e.globalAlias[k] {
		g := e
		for g.parent != nil {
			g = g.parent
		}
		g.set(name, v)
		return
	}
	if e.funcScope {
		e.define(name, v)
		return
	}
	if e.parent != nil {
		if _, ok := e.parent.get(k); ok {
			e.parent.set(name, v)
			return
		}
	}
	e.mark(name)
	e.vars[k] = v
}

// storeExisting writes an array or struct root back without creating a function local.
func (e *env) storeExisting(name string, v value.Value) {
	k := lex.IdentKey(name)
	v = e.coerce(k, v)
	if _, ok := e.vars[k]; ok {
		e.vars[k] = v
		return
	}
	if e.parent != nil {
		if _, ok := e.parent.get(k); ok {
			e.parent.storeExisting(name, v)
			return
		}
	}
	e.mark(name)
	e.vars[k] = v
}

func (e *env) markGlobal(name string) {
	if e.globalAlias == nil {
		e.globalAlias = map[string]bool{}
	}
	e.globalAlias[lex.IdentKey(name)] = true
}

func (e *env) writableHere(name string) bool {
	k := lex.IdentKey(name)
	if _, ok := e.vars[k]; ok {
		return true
	}
	return e.globalAlias != nil && e.globalAlias[k]
}

func (e *env) define(name string, v value.Value) {
	e.mark(name)
	k := lex.IdentKey(name)
	v = e.coerce(k, v)
	e.vars[k] = v
}

func (e *env) mark(name string) {
	k := lex.IdentKey(name)
	if e.declared == nil {
		e.declared = map[string]bool{}
	}
	e.declared[k] = true
	if kind, asInt, ok := suffixKind(name); ok {
		if e.kinds == nil {
			e.kinds = map[string]value.Kind{}
		}
		e.kinds[k] = kind
		if asInt {
			if e.ints == nil {
				e.ints = map[string]bool{}
			}
			e.ints[k] = true
		}
	}
}

func suffixKind(name string) (value.Kind, bool, bool) {
	switch {
	case strings.HasSuffix(name, "$"):
		return value.KindStr, false, true
	case strings.HasSuffix(name, "%"):
		return value.KindNum, true, true
	case strings.HasSuffix(name, "#"):
		return value.KindNum, false, true
	default:
		return 0, false, false
	}
}

func (e *env) coerce(k string, v value.Value) value.Value {
	for cur := e; cur != nil; cur = cur.parent {
		kind, ok := cur.kinds[k]
		if !ok {
			continue
		}
		switch kind {
		case value.KindStr:
			if v.Kind != value.KindStr {
				v = value.Str(v.String())
			}
		case value.KindNum:
			n := v.Number()
			if cur.ints != nil && cur.ints[k] {
				n = float64(int(n))
			}
			v = value.Num(n)
		}
		return v
	}
	return v
}

func (e *env) known(name string) bool {
	k := lex.IdentKey(name)
	for cur := e; cur != nil; cur = cur.parent {
		if cur.declared != nil && cur.declared[k] {
			return true
		}
		if _, ok := cur.vars[k]; ok {
			return true
		}
	}
	return false
}

type Interp struct {
	global     *env
	stack      []frame
	funcs      map[string]*ast.FuncDecl
	host       Host
	builtin    map[string]Builtin
	Out        io.Writer
	rng        *rand.Rand
	rngSeed    int64
	start      time.Time
	err        error
	status     Status
	live       bool // true after the first Flip; in-game command errors must not End
	ret        value.Value
	dataSrc    []ast.Expr
	data       []value.Value
	dataPC     int
	lists      map[int][]value.Value
	nextList   int
	machines   map[int]*stateMachine
	nextMach   int
	strict     bool
	fastHits   int
	fastCache  map[string]fastCode
	debug      bool
	breaks     map[int]bool
	stepNext   bool
	debugIn    *bufio.Reader
	debugOut   io.Writer
	types      map[string]*ast.TypeDecl
	methods    map[string]*ast.FuncDecl
	namespaces map[string]bool
	bindEnv    *env // parameter defaults evaluate here
}

func New(prog *ast.Program, host Host) *Interp {
	in := &Interp{
		global:     newEnv(nil),
		funcs:      map[string]*ast.FuncDecl{},
		host:       host,
		builtin:    map[string]Builtin{},
		Out:        os.Stdout,
		start:      time.Now(),
		lists:      map[int][]value.Value{},
		nextList:   1,
		machines:   map[int]*stateMachine{},
		nextMach:   1,
		fastCache:  map[string]fastCode{},
		types:      map[string]*ast.TypeDecl{},
		methods:    map[string]*ast.FuncDecl{},
		namespaces: map[string]bool{},
	}
	s0, s1 := entropySeed()
	in.rngSeed = int64(s0)
	in.rng = newRNG(s0, s1)
	in.installBuiltins()
	in.global.define("true", value.Num(1))
	in.global.define("false", value.Num(0))
	in.global.define("yes", value.Num(1))
	in.global.define("no", value.Num(0))
	in.global.define("null", value.Num(0))
	in.collectFuncs(prog.Stmts)
	in.collectData(prog.Stmts)
	in.stack = []frame{{kind: frameBlock, stmts: filterTop(prog.Stmts), env: in.global}}
	return in
}

func filterTop(stmts []ast.Stmt) []ast.Stmt {
	var out []ast.Stmt
	for _, s := range stmts {
		switch s.(type) {
		case *ast.FuncDecl, *ast.TypeDecl, *ast.MethodDecl, *ast.NamespaceDecl, *ast.ImportStmt, *ast.IncludeStmt:
			continue
		}
		out = append(out, s)
	}
	return out
}

func (in *Interp) collectFuncs(stmts []ast.Stmt) {
	in.collectFuncsPrefixed(stmts, "")
}

func (in *Interp) collectFuncsPrefixed(stmts []ast.Stmt, prefix string) {
	qual := func(name string) string {
		if prefix == "" {
			return name
		}
		return prefix + "." + name
	}
	for _, s := range stmts {
		switch t := s.(type) {
		case *ast.FuncDecl:
			in.funcs[lex.IdentKey(qual(t.Name))] = t
		case *ast.TypeDecl:
			tname := lex.IdentKey(qual(t.Name))
			in.types[tname] = t
			for _, m := range t.Methods {
				in.methods[tname+"."+lex.IdentKey(m.Name)] = m
			}
		case *ast.MethodDecl:
			in.methods[lex.IdentKey(qual(t.Recv))+"."+lex.IdentKey(t.Name)] = &ast.FuncDecl{
				Src: t.Src, Name: t.Name, Params: t.Params, Defaults: t.Defaults, Body: t.Body,
			}
		case *ast.NamespaceDecl:
			p := qual(t.Name)
			in.namespaces[lex.IdentKey(p)] = true
			in.collectFuncsPrefixed(t.Stmts, p)
		case *ast.IfStmt:
			in.collectFuncs(t.Then)
			for _, e := range t.ElseIf {
				in.collectFuncs(e.Body)
			}
			in.collectFuncs(t.Else)
		case *ast.WhileStmt:
			in.collectFuncs(t.Body)
		case *ast.ForStmt:
			in.collectFuncs(t.Body)
		case *ast.RepeatStmt:
			in.collectFuncs(t.Body)
		case *ast.SelectStmt:
			for _, c := range t.Cases {
				in.collectFuncs(c.Body)
			}
			in.collectFuncs(t.Default)
		}
	}
}

func (in *Interp) collectData(stmts []ast.Stmt) {
	for _, s := range stmts {
		switch t := s.(type) {
		case *ast.DataStmt:
			in.dataSrc = append(in.dataSrc, t.Values...)
		case *ast.IfStmt:
			in.collectData(t.Then)
			for _, e := range t.ElseIf {
				in.collectData(e.Body)
			}
			in.collectData(t.Else)
		case *ast.WhileStmt:
			in.collectData(t.Body)
		case *ast.ForStmt:
			in.collectData(t.Body)
		case *ast.RepeatStmt:
			in.collectData(t.Body)
		case *ast.SelectStmt:
			for _, c := range t.Cases {
				in.collectData(c.Body)
			}
			in.collectData(t.Default)
		case *ast.FuncDecl:
			in.collectData(t.Body)
		}
	}
}

func (in *Interp) Register(name string, fn Builtin) {
	in.builtin[lex.IdentKey(name)] = fn
}

func (in *Interp) MarkLive() {
	if in == nil {
		return
	}
	in.live = true
}

func (in *Interp) Done() bool {
	return in.status == StatusEnd || len(in.stack) == 0
}

func (in *Interp) Finished() bool {
	return in.Done() && in.err == nil
}

func (in *Interp) ClearLiveError() {
	if !in.live || in.err == nil {
		return
	}
	in.err = nil
	if in.status == StatusEnd && len(in.stack) > 0 {
		in.status = StatusOK
	}
}

func (in *Interp) Err() error { return in.err }

func (in *Interp) Run() error {
	for {
		st := in.Step()
		if st == StatusYield {
			return nil
		}
		if st == StatusEnd {
			return in.err
		}
	}
}

// ExecProgram runs another parsed program on this interpreter (shared variables).
// Scene files must not Flip.
func (in *Interp) ExecProgram(prog *ast.Program) error {
	in.collectFuncs(prog.Stmts)
	in.collectData(prog.Stmts)
	depth := len(in.stack)
	in.push(frame{kind: frameBlock, stmts: filterTop(prog.Stmts), env: in.global})
	for len(in.stack) > depth {
		st := in.Step()
		if st == StatusYield {
			return fmt.Errorf("LoadScene: Flip / WaitTimer / WaitKey / Delay are not allowed in a scene file")
		}
		if st == StatusEnd {
			break
		}
	}
	return in.err
}

func (in *Interp) Step() Status {
	if in.err != nil {
		if in.live {
			fmt.Fprintln(in.Out, in.err)
			in.err = nil
			in.status = StatusOK
		} else {
			in.status = StatusEnd
			return StatusEnd
		}
	}
	for len(in.stack) > 0 {
		f := &in.stack[len(in.stack)-1]
		if f.pc >= len(f.stmts) {
			if in.finishFrame(f) {
				continue
			}
			if in.status == StatusYield {
				return StatusYield
			}
			continue
		}
		s := f.stmts[f.pc]
		if in.debug {
			in.maybeBreak(s)
		}
		f.pc++
		st := in.exec(s)
		if st == StatusWaitYield {
			f.pc--
			in.status = StatusYield
			return StatusYield
		}
		if st == StatusYield {
			in.status = StatusYield
			return StatusYield
		}
		if st == StatusReturn {
			in.unwindReturn()
			continue
		}
		if st == StatusExit {
			in.unwindExit()
			continue
		}
		if st == StatusEnd {
			in.status = StatusEnd
			in.stack = nil
			return StatusEnd
		}
	}
	in.status = StatusEnd
	return StatusEnd
}

func (in *Interp) finishFrame(f *frame) bool {
	loopErr := func(err error) bool {
		if in.live {
			fmt.Fprintln(in.Out, err)
			if f.kind == frameWhile || f.kind == frameFor || f.kind == frameRepeat {
				f.pc = 0
				return true
			}
			return len(in.stack) > 0
		}
		in.err = err
		in.stack = nil
		return false
	}
	switch f.kind {
	case frameWhile:
		ok, err := in.whileContinues(f.cond)
		if err != nil {
			return loopErr(err)
		}
		if ok {
			f.pc = 0
			return true
		}
	case frameFor:
		if f.forEach != nil {
			f.forAt++
			if f.forAt < len(f.forEach) {
				in.env().set(f.forVar, f.forEach[f.forAt])
				f.pc = 0
				return true
			}
			break
		}
		cur, _ := in.env().get(f.forVar)
		step := 1.0
		if f.step != nil {
			sv, err := in.eval(f.step)
			if err != nil {
				return loopErr(err)
			}
			step = sv.Number()
		}
		next := cur.Number() + step
		in.env().set(f.forVar, value.Num(next))
		endv, err := in.eval(f.end)
		if err != nil {
			return loopErr(err)
		}
		end := endv.Number()
		cont := (step >= 0 && next <= end) || (step < 0 && next >= end)
		if cont {
			f.pc = 0
			return true
		}
	case frameRepeat:
		ok, err := in.evalBool(f.until)
		if err != nil {
			return loopErr(err)
		}
		if !ok {
			f.pc = 0
			return true
		}
	case frameFunc:
		in.ret = value.Num(0)
	case frameTry:
		// body finished without error; skip Catch
	}
	in.stack = in.stack[:len(in.stack)-1]
	return len(in.stack) > 0
}

func (in *Interp) unwindReturn() {
	for len(in.stack) > 0 {
		f := in.stack[len(in.stack)-1]
		in.stack = in.stack[:len(in.stack)-1]
		if f.kind == frameFunc {
			return
		}
	}
}

func (in *Interp) unwindExit() {
	for len(in.stack) > 0 {
		f := in.stack[len(in.stack)-1]
		in.stack = in.stack[:len(in.stack)-1]
		if f.kind == frameWhile || f.kind == frameFor || f.kind == frameRepeat {
			return
		}
		if f.kind == frameFunc {
			in.ret = value.Num(0)
			return
		}
	}
}

func (in *Interp) env() *env {
	if in.bindEnv != nil {
		for i := len(in.stack) - 1; i >= 0; i-- {
			if in.stack[i].env != nil {
				return in.stack[i].env
			}
		}
		return in.bindEnv
	}
	for i := len(in.stack) - 1; i >= 0; i-- {
		if in.stack[i].env != nil {
			return in.stack[i].env
		}
	}
	return in.global
}

func (in *Interp) exec(s ast.Stmt) Status {
	switch t := s.(type) {
	case *ast.AssignStmt:
		v, err := in.eval(t.Value)
		if err != nil {
			return in.cmdFail(t, err)
		}
		if len(t.Names) > 1 {
			if err := in.assignUnpack(t.Names, v); err != nil {
				return in.cmdFail(t, err)
			}
			return StatusOK
		}
		bare := len(t.Index) == 0 && len(t.Fields) == 0
		if err := in.assignAllowed(t.Name, bare); err != nil {
			return in.cmdFail(t, err)
		}
		if len(t.Index) > 0 {
			if err := in.setIndex(t.Name, t.Index, t.Fields, v); err != nil {
				return in.cmdFail(t, err)
			}
			return StatusOK
		}
		if len(t.Fields) > 0 {
			if err := in.setFields(t.Name, t.Fields, v); err != nil {
				return in.cmdFail(t, err)
			}
			return StatusOK
		}
		in.env().set(t.Name, v)
	case *ast.ExprStmt:
		if _, err := in.eval(t.Value); err != nil {
			return in.cmdFail(t, err)
		}
	case *ast.MethodStmt:
		args, err := in.evalArgs(t.Args)
		if err != nil {
			return in.cmdFail(t, err)
		}
		if _, err := in.callDotted(t.Object, t.Path, t.Method, args, t.Paren); err != nil {
			return in.cmdFail(t, err)
		}
	case *ast.CallStmt:
		name := lex.IdentKey(t.Name)
		args, err := in.evalArgs(t.Args)
		if err != nil {
			return in.cmdFail(t, err)
		}
		if fn, ok := in.funcs[name]; ok {
			local, args, err := in.prepareCall(fn, args)
			if err != nil {
				return in.cmdFail(t, err)
			}
			if code, ok := in.compiled(fn); ok {
				in.ret = in.runFast(fn, code, args)
				return StatusOK
			}
			in.push(frame{kind: frameFunc, stmts: filterTop(fn.Body), env: local})
			return StatusOK
		}
		if in.host != nil && in.host.Yields(name) {
			if _, err := in.host.Call(name, args); err != nil {
				return in.cmdFail(t, err)
			}
			if name == "flip" {
				in.live = true
				return StatusYield
			}
			if in.host.Holds() {
				return StatusWaitYield
			}
			return StatusOK
		}
		if _, err := in.call(name, args); err != nil {
			return in.cmdFail(t, err)
		}
	case *ast.IfStmt:
		ok, err := in.evalBool(t.Cond)
		if err != nil {
			return in.cmdFail(t, err)
		}
		if ok {
			in.push(frame{kind: frameBlock, stmts: t.Then, env: in.env()})
			return StatusOK
		}
		for _, e := range t.ElseIf {
			ok, err := in.evalBool(e.Cond)
			if err != nil {
				return in.cmdFail(t, err)
			}
			if ok {
				in.push(frame{kind: frameBlock, stmts: e.Body, env: in.env()})
				return StatusOK
			}
		}
		if len(t.Else) > 0 {
			in.push(frame{kind: frameBlock, stmts: t.Else, env: in.env()})
		}
	case *ast.WhileStmt:
		ok, err := in.whileContinues(t.Cond)
		if err != nil {
			return in.cmdFail(t, err)
		}
		if ok {
			in.push(frame{kind: frameWhile, stmts: t.Body, cond: t.Cond, env: in.env()})
		}
	case *ast.ForStmt:
		if t.In != nil {
			seq, err := in.eval(t.In)
			if err != nil {
				return in.cmdFail(t, err)
			}
			elems, err := in.iterElems(seq)
			if err != nil {
				return in.cmdFail(t, err)
			}
			if len(elems) == 0 {
				break
			}
			in.env().define(t.Var, elems[0])
			in.push(frame{kind: frameFor, stmts: t.Body, forVar: t.Var, forEach: elems, env: in.env()})
			break
		}
		start, err := in.eval(t.Start)
		if err != nil {
			return in.cmdFail(t, err)
		}
		in.env().define(t.Var, start)
		endv, err := in.eval(t.End)
		if err != nil {
			return in.cmdFail(t, err)
		}
		step := 1.0
		if t.Step != nil {
			sv, err := in.eval(t.Step)
			if err != nil {
				return in.cmdFail(t, err)
			}
			step = sv.Number()
		}
		n := start.Number()
		end := endv.Number()
		if (step >= 0 && n <= end) || (step < 0 && n >= end) {
			in.push(frame{kind: frameFor, stmts: t.Body, forVar: t.Var, end: t.End, step: t.Step, env: in.env()})
		}
	case *ast.RepeatStmt:
		in.push(frame{kind: frameRepeat, stmts: t.Body, until: t.Until, env: in.env()})
	case *ast.ReturnStmt:
		if t.Value != nil {
			v, err := in.eval(t.Value)
			if err != nil {
				return in.cmdFail(t, err)
			}
			in.ret = v
		} else {
			in.ret = value.Num(0)
		}
		return StatusReturn
	case *ast.DimStmt:
		sizes := make([]int, len(t.Sizes))
		for i, e := range t.Sizes {
			v, err := in.eval(e)
			if err != nil {
				return in.cmdFail(t, err)
			}
			sizes[i] = v.Int()
		}
		arr := value.Array(sizes)
		if t.TypeName != "" {
			if _, ok := in.types[lex.IdentKey(t.TypeName)]; !ok {
				return in.cmdFail(t, fmt.Errorf("unknown type %s", t.TypeName))
			}
			arr.TypeName = lex.IdentKey(t.TypeName)
		}
		if t.Redim {
			if old, ok := in.env().get(t.Name); ok && old.Kind == value.KindArray {
				n := len(old.Elems)
				if n > len(arr.Elems) {
					n = len(arr.Elems)
				}
				copy(arr.Elems, old.Elems[:n])
				if arr.TypeName == "" {
					arr.TypeName = old.TypeName
				}
			}
		}
		in.env().define(t.Name, arr)
	case *ast.GlobalStmt:
		e := in.env()
		for i, n := range t.Names {
			hasVal := i < len(t.Values) && t.Values[i] != nil
			v := value.Num(0)
			if hasVal {
				ev, err := in.eval(t.Values[i])
				if err != nil {
					return in.cmdFail(t, err)
				}
				v = ev
			}
			if e.funcScope {
				e.markGlobal(n)
				if hasVal {
					in.global.define(n, v)
				} else if _, ok := in.global.get(n); !ok {
					in.global.define(n, value.Num(0))
				}
				continue
			}
			in.global.define(n, v)
		}
	case *ast.LocalStmt:
		e := in.env()
		for i, n := range t.Names {
			v := value.Num(0)
			if i < len(t.Values) && t.Values[i] != nil {
				ev, err := in.eval(t.Values[i])
				if err != nil {
					return in.cmdFail(t, err)
				}
				v = ev
			}
			e.define(n, v)
		}
	case *ast.EnumStmt:
		n := 0.0
		for i, name := range t.Names {
			if i < len(t.Values) && t.Values[i] != nil {
				ev, err := in.eval(t.Values[i])
				if err != nil {
					return in.cmdFail(t, err)
				}
				n = ev.Number()
			}
			in.global.define(name, value.Num(n))
			if t.Name != "" {
				in.global.define(t.Name+"_"+name, value.Num(n))
			}
			n++
		}
	case *ast.ConstStmt:
		e := in.env()
		for i, n := range t.Names {
			v := value.Num(0)
			if i < len(t.Values) && t.Values[i] != nil {
				ev, err := in.eval(t.Values[i])
				if err != nil {
					return in.cmdFail(t, err)
				}
				v = ev
			}
			e.define(n, v)
		}
	case *ast.SelectStmt:
		cur, err := in.eval(t.Value)
		if err != nil {
			return in.cmdFail(t, err)
		}
		for _, c := range t.Cases {
			ok, err := in.caseMatch(cur, c)
			if err != nil {
				return in.cmdFail(t, err)
			}
			if ok {
				in.push(frame{kind: frameBlock, stmts: c.Body, env: in.env()})
				return StatusOK
			}
		}
		if len(t.Default) > 0 {
			in.push(frame{kind: frameBlock, stmts: t.Default, env: in.env()})
		}
	case *ast.DataStmt:
		// collected at load
	case *ast.ReadStmt:
		if err := in.readNames(t.Names, t.Quals); err != nil {
			return in.cmdFail(t, err)
		}
	case *ast.RestoreStmt:
		in.dataPC = 0
	case *ast.StrictStmt:
		in.strict = true
	case *ast.TryStmt:
		in.push(frame{kind: frameTry, stmts: t.Body, catch: t.Catch, errName: t.ErrVar, env: in.env()})
	case *ast.EndStmt:
		return StatusEnd
	case *ast.ExitStmt:
		if t.Function {
			in.ret = value.Num(0)
			return StatusReturn
		}
		return StatusExit
	case *ast.FuncDecl, *ast.TypeDecl, *ast.MethodDecl, *ast.NamespaceDecl, *ast.ImportStmt, *ast.IncludeStmt:
		// already collected / expanded
	}
	return StatusOK
}

func (in *Interp) push(f frame) { in.stack = append(in.stack, f) }

func (in *Interp) wrap(n ast.Node, err error) error {
	line, col := n.Pos()
	return fmt.Errorf("line %d:%d: %w", line, col, err)
}

func (in *Interp) cmdFail(n ast.Node, err error) Status {
	werr := in.wrap(n, err)
	if in.raise(werr) {
		return StatusOK
	}
	if in.live {
		fmt.Fprintln(in.Out, werr)
		return StatusOK
	}
	in.err = werr
	return StatusEnd
}

func (in *Interp) raise(err error) bool {
	for i := len(in.stack) - 1; i >= 0; i-- {
		f := in.stack[i]
		if f.kind != frameTry {
			continue
		}
		env := f.env
		if env == nil {
			env = in.global
		}
		if f.errName != "" {
			env.define(f.errName, value.Str(err.Error()))
		}
		in.stack = in.stack[:i]
		in.err = nil
		in.status = StatusOK
		in.push(frame{kind: frameBlock, stmts: f.catch, env: env})
		return true
	}
	return false
}

func (in *Interp) evalBool(e ast.Expr) (bool, error) {
	if in.live {
		if e2 := captureQuitAsEsc(e); e2 != nil {
			e = e2
		}
	}
	v, err := in.eval(e)
	if err != nil {
		return false, err
	}
	return v.IsTrue(), nil
}

// captureQuitAsEsc turns `If KeyHit(1) Or frames > 8 Then End` into
// `frames > 8 And KeyHit(1)` after the first Flip. The Or form was a
// batch-capture loop that closes the window on frame 9.
func captureQuitAsEsc(e ast.Expr) ast.Expr {
	b, ok := e.(*ast.BinaryExpr)
	if !ok || !strings.EqualFold(b.Op, "or") {
		return nil
	}
	leftHit, leftFrames := isEscHitCall(b.Left), isFramesPast(b.Left)
	rightHit, rightFrames := isEscHitCall(b.Right), isFramesPast(b.Right)
	if leftHit && rightFrames {
		return &ast.BinaryExpr{Src: b.Src, Op: "and", Left: b.Right, Right: b.Left}
	}
	if rightHit && leftFrames {
		return &ast.BinaryExpr{Src: b.Src, Op: "and", Left: b.Left, Right: b.Right}
	}
	return nil
}

func isEscHitCall(e ast.Expr) bool {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	switch lex.IdentKey(c.Name) {
	case "keyhit", "keydown":
		return true
	}
	return false
}

func isFramesPast(e ast.Expr) bool {
	b, ok := e.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	if b.Op != ">" && b.Op != ">=" {
		return false
	}
	id, ok := b.Left.(*ast.IdentExpr)
	if !ok || !strings.EqualFold(id.Name, "frames") {
		return false
	}
	n, ok := b.Right.(*ast.NumberExpr)
	if !ok {
		return false
	}
	return n.Value <= 60
}

// preFlipQuitInput makes KeyDown(1)/KeyHit(Escape)/WindowShouldClose
// report false until the first Flip so `While Not KeyDown(1)` and
// `If KeyHit(KEY_ESCAPE) Then End` cannot End before a frame presents.
func (in *Interp) preFlipQuitInput(name string, args []value.Value) (value.Value, bool) {
	if in.live {
		return value.Value{}, false
	}
	switch lex.IdentKey(name) {
	case "windowshouldclose":
		return value.Num(0), true
	case "keydown", "keyhit":
		if isEscapeKeyArg(args) {
			return value.Num(0), true
		}
	}
	return value.Value{}, false
}

func isEscapeKeyArg(args []value.Value) bool {
	if len(args) == 0 {
		return false
	}
	if args[0].Kind == value.KindNum {
		n := args[0].Int()
		return n == 1 || n == 0x01
	}
	s := strings.ToLower(strings.TrimSpace(args[0].String()))
	return s == "1" || s == "escape" || s == "esc" || s == "key_escape"
}

// whileContinues keeps KeyDown / WindowShouldClose loops running until
// the first Flip. Combined with preFlipQuitInput, `While Not KeyDown(1)`
// is true at loop entry even if the host already reports a phantom Escape.
func (in *Interp) whileContinues(cond ast.Expr) (bool, error) {
	if !in.live && isPreFlipQuitCond(cond) {
		return true, nil
	}
	return in.evalBool(cond)
}

func isPreFlipQuitCond(e ast.Expr) bool {
	if e == nil {
		return false
	}
	switch t := e.(type) {
	case *ast.CallExpr:
		switch lex.IdentKey(t.Name) {
		case "keydown", "keyhit", "windowshouldclose":
			return true
		}
		for _, a := range t.Args {
			if isPreFlipQuitCond(a) {
				return true
			}
		}
	case *ast.UnaryExpr:
		return isPreFlipQuitCond(t.X)
	case *ast.BinaryExpr:
		return isPreFlipQuitCond(t.Left) || isPreFlipQuitCond(t.Right)
	}
	return false
}

func (in *Interp) evalArgs(args []ast.Expr) ([]value.Value, error) {
	out := make([]value.Value, len(args))
	for i, a := range args {
		v, err := in.eval(a)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func (in *Interp) eval(e ast.Expr) (value.Value, error) {
	switch t := e.(type) {
	case *ast.NumberExpr:
		return value.Num(t.Value), nil
	case *ast.VecExpr:
		elems := make([]value.Value, len(t.Elems))
		for i, e := range t.Elems {
			v, err := in.eval(e)
			if err != nil {
				return value.Value{}, err
			}
			elems[i] = v
		}
		return value.Vec(elems), nil
	case *ast.StringExpr:
		return value.Str(t.Value), nil
	case *ast.IdentExpr:
		if v, ok := in.env().get(t.Name); ok {
			return v, nil
		}
		if fn, ok := in.funcs[lex.IdentKey(t.Name)]; ok && len(fn.Params) > 0 {
			return value.Func(fn.Name), nil
		}
		// zero-arg builtin / function
		if v, err, ok := in.tryCall(lex.IdentKey(t.Name), nil); ok {
			return v, err
		}
		if in.strict {
			return value.Value{}, fmt.Errorf("undefined %s", t.Name)
		}
		return value.Num(0), nil
	case *ast.CallExpr:
		args, err := in.evalArgs(t.Args)
		if err != nil {
			return value.Value{}, err
		}
		// array index via ()
		if v, ok := in.env().get(t.Name); ok && v.Kind == value.KindArray {
			return in.indexOf(v, args)
		}
		if v, ok := in.env().get(t.Name); ok && v.Kind == value.KindMap {
			if len(args) == 0 {
				return value.Num(0), nil
			}
			got, ok := v.Field(args[0].String())
			if !ok {
				return value.Num(0), nil
			}
			return got, nil
		}
		if td, ok := in.types[lex.IdentKey(t.Name)]; ok {
			return in.construct(td, args), nil
		}
		return in.call(lex.IdentKey(t.Name), args)
	case *ast.FieldExpr:
		obj, err := in.eval(t.X)
		if err != nil {
			return value.Value{}, err
		}
		f, ok := obj.Field(t.Field)
		if !ok {
			return value.Num(0), nil
		}
		return f, nil
	case *ast.MethodExpr:
		args, err := in.evalArgs(t.Args)
		if err != nil {
			return value.Value{}, err
		}
		if id, ok := t.X.(*ast.IdentExpr); ok {
			return in.callDotted(id.Name, nil, t.Name, args, true)
		}
		recv, err := in.eval(t.X)
		if err != nil {
			return value.Value{}, err
		}
		return in.callMethod(recv, t.Name, args, "", nil, true)
	case *ast.NewExpr:
		td, ok := in.types[lex.IdentKey(t.Type)]
		if !ok {
			return value.Value{}, fmt.Errorf("unknown type %s", t.Type)
		}
		args, err := in.evalArgs(t.Args)
		if err != nil {
			return value.Value{}, err
		}
		return in.construct(td, args), nil
	case *ast.IndexExpr:
		args, err := in.evalArgs(t.Index)
		if err != nil {
			return value.Value{}, err
		}
		v, ok := in.env().get(t.Name)
		if !ok {
			return value.Value{}, fmt.Errorf("unknown array %s", t.Name)
		}
		return in.indexOf(v, args)
	case *ast.UnaryExpr:
		x, err := in.eval(t.X)
		if err != nil {
			return value.Value{}, err
		}
		switch strings.ToLower(t.Op) {
		case "not":
			if x.IsTrue() {
				return value.Num(0), nil
			}
			return value.Num(1), nil
		case "-":
			return value.Num(-x.Number()), nil
		case "+":
			return value.Num(x.Number()), nil
		}
	case *ast.BinaryExpr:
		if t.Op == "and" {
			l, err := in.eval(t.Left)
			if err != nil {
				return value.Value{}, err
			}
			if !l.IsTrue() {
				return value.Num(0), nil
			}
			r, err := in.eval(t.Right)
			if err != nil {
				return value.Value{}, err
			}
			if r.IsTrue() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}
		if t.Op == "or" {
			l, err := in.eval(t.Left)
			if err != nil {
				return value.Value{}, err
			}
			if l.IsTrue() {
				return value.Num(1), nil
			}
			r, err := in.eval(t.Right)
			if err != nil {
				return value.Value{}, err
			}
			if r.IsTrue() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}
		l, err := in.eval(t.Left)
		if err != nil {
			return value.Value{}, err
		}
		r, err := in.eval(t.Right)
		if err != nil {
			return value.Value{}, err
		}
		return in.binary(t.Op, l, r)
	}
	return value.Value{}, fmt.Errorf("bad expression")
}

func (in *Interp) binary(op string, l, r value.Value) (value.Value, error) {
	switch op {
	case "+":
		return value.Add(l, r), nil
	case "-":
		return value.Num(l.Number() - r.Number()), nil
	case "*":
		return value.Num(l.Number() * r.Number()), nil
	case "/":
		if r.Number() == 0 {
			return value.Num(0), nil
		}
		return value.Num(l.Number() / r.Number()), nil
	case "^":
		return value.Num(math.Pow(l.Number(), r.Number())), nil
	case "mod":
		if r.Number() == 0 {
			return value.Num(0), nil
		}
		return value.Num(math.Mod(l.Number(), r.Number())), nil
	case "xor":
		if l.IsTrue() != r.IsTrue() {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	case "=":
		if value.Cmp(l, r) == 0 {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	case "<>":
		if value.Cmp(l, r) != 0 {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	case "<":
		if value.Cmp(l, r) < 0 {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	case ">":
		if value.Cmp(l, r) > 0 {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	case "<=":
		if value.Cmp(l, r) <= 0 {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	case ">=":
		if value.Cmp(l, r) >= 0 {
			return value.Num(1), nil
		}
		return value.Num(0), nil
	}
	return value.Value{}, fmt.Errorf("unknown operator %s", op)
}

func (in *Interp) indexInts(idx []value.Value) []int {
	out := make([]int, len(idx))
	for i, v := range idx {
		out[i] = v.Int()
	}
	return out
}

func (in *Interp) indexOf(arr value.Value, idx []value.Value) (value.Value, error) {
	if arr.Kind != value.KindArray || len(arr.Elems) == 0 {
		return value.Value{}, fmt.Errorf("not an array")
	}
	i, ok := arr.Offset(in.indexInts(idx))
	if !ok {
		return value.Num(0), nil
	}
	return arr.Elems[i], nil
}

func (in *Interp) setIndex(name string, idx []ast.Expr, fields []string, v value.Value) error {
	arr, ok := in.env().get(name)
	if !ok {
		return fmt.Errorf("not an array: %s", name)
	}
	if arr.Kind == value.KindMap {
		args, err := in.evalArgs(idx)
		if err != nil {
			return err
		}
		key := ""
		if len(args) > 0 {
			key = args[0].String()
		}
		if len(fields) == 0 {
			if !arr.SetField(key, v) {
				return fmt.Errorf("not a map: %s", name)
			}
			in.env().storeExisting(name, arr)
			return nil
		}
		el, ok := arr.Field(key)
		if !ok || (el.Kind != value.KindStruct && el.Kind != value.KindMap) {
			el = value.Map()
		}
		if err := assignFields(&el, fields, v); err != nil {
			return err
		}
		if !arr.SetField(key, el) {
			return fmt.Errorf("not a map: %s", name)
		}
		in.env().storeExisting(name, arr)
		return nil
	}
	if arr.Kind != value.KindArray {
		return fmt.Errorf("not an array: %s", name)
	}
	args, err := in.evalArgs(idx)
	if err != nil {
		return err
	}
	i, ok := arr.Offset(in.indexInts(args))
	if !ok {
		return fmt.Errorf("array index out of range")
	}
	if len(fields) == 0 {
		arr.Elems[i] = v
		in.env().storeExisting(name, arr)
		return nil
	}
	el, err := in.promoteFieldTarget(arr, arr.Elems[i])
	if err != nil {
		return err
	}
	if err := assignFields(&el, fields, v); err != nil {
		return err
	}
	arr.Elems[i] = el
	in.env().storeExisting(name, arr)
	return nil
}

func (in *Interp) promoteFieldTarget(arr, el value.Value) (value.Value, error) {
	if el.Kind == value.KindStruct || el.Kind == value.KindMap {
		return el, nil
	}
	if el.Kind != value.KindNum || el.Num != 0 {
		return value.Value{}, fmt.Errorf("not a struct")
	}
	if arr.TypeName != "" {
		td, ok := in.types[lex.IdentKey(arr.TypeName)]
		if !ok {
			return value.Value{}, fmt.Errorf("unknown type %s", arr.TypeName)
		}
		names := make([]string, len(td.Fields))
		for i, f := range td.Fields {
			names[i] = f.Name
		}
		return value.StructOf(td.Name, names), nil
	}
	return value.Map(), nil
}

func assignFields(root *value.Value, fields []string, val value.Value) error {
	if root.Kind != value.KindStruct && root.Kind != value.KindMap {
		return fmt.Errorf("not a struct")
	}
	cur := *root
	for i, f := range fields {
		if i == len(fields)-1 {
			if !cur.SetField(f, val) {
				return fmt.Errorf("cannot set .%s", f)
			}
			if i == 0 {
				*root = cur
			}
			return nil
		}
		next, ok := cur.Field(f)
		if !ok || (next.Kind != value.KindStruct && next.Kind != value.KindMap) {
			return fmt.Errorf(".%s is not a struct", f)
		}
		cur = next
	}
	return nil
}

func (in *Interp) caseMatch(cur value.Value, c ast.CaseClause) (bool, error) {
	for _, it := range c.Items {
		low, err := in.eval(it.Low)
		if err != nil {
			return false, err
		}
		if it.High != nil {
			hi, err := in.eval(it.High)
			if err != nil {
				return false, err
			}
			if value.Cmp(cur, low) >= 0 && value.Cmp(cur, hi) <= 0 {
				return true, nil
			}
			continue
		}
		if value.Cmp(cur, low) == 0 {
			return true, nil
		}
	}
	return false, nil
}

func (in *Interp) ensureData() {
	if in.data != nil {
		return
	}
	in.data = make([]value.Value, 0, len(in.dataSrc))
	for _, e := range in.dataSrc {
		v, err := in.eval(e)
		if err != nil {
			in.data = append(in.data, value.Num(0))
			continue
		}
		in.data = append(in.data, v)
	}
}

func (in *Interp) readNames(names []string, quals [][]string) error {
	in.ensureData()
	for i, n := range names {
		var v value.Value
		if in.dataPC < len(in.data) {
			v = in.data[in.dataPC]
			in.dataPC++
		}
		if i < len(quals) && len(quals[i]) > 0 {
			if err := in.setFields(n, quals[i], v); err != nil {
				return err
			}
			continue
		}
		in.env().set(n, v)
	}
	return nil
}

func (in *Interp) assignAllowed(name string, bare bool) error {
	if !in.strict {
		return nil
	}
	e := in.env()
	if bare && e.funcScope {
		if !e.writableHere(name) {
			return fmt.Errorf("undefined %s", name)
		}
		return nil
	}
	if !e.known(name) {
		return fmt.Errorf("undefined %s", name)
	}
	return nil
}

func (in *Interp) assignUnpack(names []string, v value.Value) error {
	elems := unpackList(v, len(names))
	for i, n := range names {
		if err := in.assignAllowed(n, true); err != nil {
			return err
		}
		in.env().set(n, elems[i])
	}
	return nil
}

func unpackList(v value.Value, n int) []value.Value {
	out := make([]value.Value, n)
	if (v.Kind == value.KindVec || v.Kind == value.KindArray) && len(v.Elems) > 0 {
		for i := 0; i < n; i++ {
			if i < len(v.Elems) {
				out[i] = v.Elems[i]
			} else {
				out[i] = value.Num(0)
			}
		}
		return out
	}
	if n > 0 {
		out[0] = v
	}
	for i := 1; i < n; i++ {
		out[i] = value.Num(0)
	}
	return out
}

func (in *Interp) iterElems(v value.Value) ([]value.Value, error) {
	switch {
	case v.Kind == value.KindNum && v.TypeName == "list":
		return append([]value.Value(nil), in.lists[v.Int()]...), nil
	case v.Kind == value.KindVec:
		return append([]value.Value(nil), v.Elems...), nil
	case v.Kind == value.KindArray:
		if v.TypeName == "" {
			return append([]value.Value(nil), v.Elems...), nil
		}
		out := make([]value.Value, 0, len(v.Elems))
		for _, el := range v.Elems {
			if el.Kind == value.KindNum && el.Num == 0 && el.TypeName == "" {
				continue
			}
			out = append(out, el)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("For In expects an array, list, or vector")
	}
}

func (in *Interp) tryCall(name string, args []value.Value) (value.Value, error, bool) {
	if v, ok := in.preFlipQuitInput(name, args); ok {
		return v, nil, true
	}
	if td, ok := in.types[name]; ok {
		return in.construct(td, args), nil, true
	}
	if fn, ok := in.funcs[name]; ok {
		v, err := in.callUser(fn, args)
		return v, err, true
	}
	if fn, ok := in.builtin[name]; ok {
		v, err := fn(args)
		return v, err, true
	}
	if in.host != nil {
		v, err := in.host.Call(name, args)
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			return v, err, true
		}
	}
	return value.Value{}, nil, false
}

func (in *Interp) call(name string, args []value.Value) (value.Value, error) {
	args = syntax.ExpandCommandArgs(name, args)
	if v, err, ok := in.tryCall(name, args); ok {
		if err != nil {
			return v, err
		}
		if syntax.ReturnsEntity(name) && len(args) > 0 && v.Kind == value.KindNum && v.Num == 0 {
			return args[0], nil
		}
		return v, nil
	}
	return value.Value{}, fmt.Errorf("unknown command %s", name)
}

func (in *Interp) prepareCall(fn *ast.FuncDecl, args []value.Value) (*env, []value.Value, error) {
	local := newEnv(in.global)
	local.funcScope = true
	savedStack := in.stack
	savedBind := in.bindEnv
	in.stack = nil
	in.bindEnv = local
	defer func() {
		in.stack = savedStack
		in.bindEnv = savedBind
	}()
	full := make([]value.Value, len(fn.Params))
	for i, p := range fn.Params {
		var v value.Value
		switch {
		case i < len(args):
			v = args[i]
		case i < len(fn.Defaults) && fn.Defaults[i] != nil:
			ev, err := in.eval(fn.Defaults[i])
			if err != nil {
				return nil, nil, err
			}
			v = ev
		}
		local.define(p, v)
		full[i] = v
	}
	return local, full, nil
}

func (in *Interp) callUser(fn *ast.FuncDecl, args []value.Value) (value.Value, error) {
	local, args, err := in.prepareCall(fn, args)
	if err != nil {
		return value.Value{}, err
	}
	if code, ok := in.compiled(fn); ok {
		return in.runFast(fn, code, args), nil
	}
	saved := in.stack
	savedStatus := in.status
	in.stack = []frame{{kind: frameFunc, stmts: filterTop(fn.Body), env: local}}
	in.ret = value.Num(0)
	var yieldErr error
	for {
		st := in.Step()
		if st == StatusYield {
			yieldErr = fmt.Errorf("Flip, WaitTimer, WaitKey, and Delay are not allowed inside a Function used as an expression")
			break
		}
		if st == StatusEnd || len(in.stack) == 0 {
			break
		}
	}
	ret := in.ret
	err = in.err
	in.stack = saved
	in.status = savedStatus
	in.err = nil
	if yieldErr != nil {
		return value.Value{}, yieldErr
	}
	return ret, err
}

func (in *Interp) construct(td *ast.TypeDecl, args []value.Value) value.Value {
	names := make([]string, len(td.Fields))
	for i, f := range td.Fields {
		names[i] = f.Name
	}
	v := value.StructOf(td.Name, names)
	for i, a := range args {
		if i < len(td.Fields) {
			v.SetField(td.Fields[i].Name, a.Clone())
		}
	}
	return v
}

func (in *Interp) setFields(name string, fields []string, val value.Value) error {
	root, ok := in.env().get(name)
	if !ok || (root.Kind != value.KindStruct && root.Kind != value.KindMap) {
		return fmt.Errorf("%s is not a struct", name)
	}
	cur := root
	for i, f := range fields {
		if i == len(fields)-1 {
			if !cur.SetField(f, val) {
				return fmt.Errorf("cannot set %s.%s", name, f)
			}
			in.env().storeExisting(name, root)
			return nil
		}
		next, ok := cur.Field(f)
		if !ok || next.Kind != value.KindStruct {
			return fmt.Errorf("%s.%s is not a struct", name, f)
		}
		cur = next
	}
	return nil
}

func (in *Interp) callDotted(object string, path []string, method string, args []value.Value, paren bool) (value.Value, error) {
	key := lex.IdentKey(object) + "." + lex.IdentKey(method)
	if len(path) == 0 {
		if fn, ok := in.funcs[key]; ok {
			return in.callUser(fn, args)
		}
		if td, ok := in.types[key]; ok {
			return in.construct(td, args), nil
		}
		if in.namespaces[lex.IdentKey(object)] {
			return in.call(key, args)
		}
	}
	recv, ok := in.env().get(object)
	if !ok {
		if td, ok := in.types[lex.IdentKey(object)]; ok && len(path) == 0 {
			inst := in.construct(td, nil)
			return in.callMethod(inst, method, args, "", nil, paren)
		}
		return value.Value{}, fmt.Errorf("unknown %s.%s", object, method)
	}
	for _, f := range path {
		next, ok := recv.Field(f)
		if !ok {
			return value.Value{}, fmt.Errorf("%s has no field %s", object, f)
		}
		recv = next
	}
	return in.callMethod(recv, method, args, object, path, paren)
}

func (in *Interp) callMethod(recv value.Value, method string, args []value.Value, root string, path []string, paren bool) (value.Value, error) {
	if recv.Kind == value.KindStruct {
		return in.callStructMethod(recv, method, args, root, path)
	}
	if paren {
		if cmd, ok := syntax.ResolveMethod(method); ok {
			full := append([]value.Value{recv}, args...)
			return in.call(cmd, full)
		}
	}
	return value.Value{}, fmt.Errorf("method %s on non-struct", method)
}

func (in *Interp) callStructMethod(recv value.Value, method string, args []value.Value, root string, path []string) (value.Value, error) {
	fn := in.methods[recv.TypeName+"."+lex.IdentKey(method)]
	if fn == nil {
		return value.Value{}, fmt.Errorf("unknown method %s.%s", recv.TypeName, method)
	}
	local := newEnv(in.global)
	local.define("self", recv)
	local.define("this", recv)
	for i, p := range fn.Params {
		var v value.Value
		if i < len(args) {
			v = args[i]
		}
		local.define(p, v)
	}
	saved := in.stack
	in.stack = []frame{{kind: frameFunc, stmts: filterTop(fn.Body), env: local}}
	in.ret = value.Num(0)
	for {
		st := in.Step()
		if st == StatusYield {
			in.err = fmt.Errorf("Flip, WaitTimer, WaitKey, and Delay are not allowed inside a Method")
			in.stack = saved
			return value.Value{}, in.err
		}
		if st == StatusEnd || len(in.stack) == 0 {
			break
		}
	}
	ret := in.ret
	self, _ := local.get("self")
	in.stack = saved
	in.status = StatusOK
	in.err = nil
	if root != "" {
		if len(path) == 0 {
			in.env().set(root, self)
		} else {
			_ = in.setFields(root, path, self)
		}
	}
	return ret, nil
}

func (in *Interp) installBuiltins() {
	n := func(name string, fn func([]value.Value) value.Value) {
		in.Register(name, func(args []value.Value) (value.Value, error) { return fn(args), nil })
	}
	n("print", func(args []value.Value) value.Value {
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = a.String()
		}
		line := strings.Join(parts, " ")
		fmt.Fprintln(in.Out, line)
		if in.host != nil {
			_, _ = in.host.Call("hudprint", []value.Value{value.Str(line)})
		}
		return value.Num(0)
	})
	n("sin", func(a []value.Value) value.Value { return value.Num(math.Sin(deg(a, 0))) })
	n("cos", func(a []value.Value) value.Value { return value.Num(math.Cos(deg(a, 0))) })
	n("tan", func(a []value.Value) value.Value { return value.Num(math.Tan(deg(a, 0))) })
	n("asin", func(a []value.Value) value.Value { return value.Num(radToDeg(math.Asin(num(a, 0)))) })
	n("acos", func(a []value.Value) value.Value { return value.Num(radToDeg(math.Acos(num(a, 0)))) })
	n("atan", func(a []value.Value) value.Value { return value.Num(radToDeg(math.Atan(num(a, 0)))) })
	n("atan2", func(a []value.Value) value.Value { return value.Num(radToDeg(math.Atan2(num(a, 0), num(a, 1)))) })
	n("sqr", func(a []value.Value) value.Value { return value.Num(math.Sqrt(num(a, 0))) })
	n("abs", func(a []value.Value) value.Value { return value.Num(math.Abs(num(a, 0))) })
	n("int", func(a []value.Value) value.Value { return value.Num(math.Trunc(num(a, 0))) })
	n("floor", func(a []value.Value) value.Value { return value.Num(math.Floor(num(a, 0))) })
	n("ceil", func(a []value.Value) value.Value { return value.Num(math.Ceil(num(a, 0))) })
	n("float", func(a []value.Value) value.Value { return value.Num(num(a, 0)) })
	n("sgn", func(a []value.Value) value.Value {
		x := num(a, 0)
		if x < 0 {
			return value.Num(-1)
		}
		if x > 0 {
			return value.Num(1)
		}
		return value.Num(0)
	})
	n("min", func(a []value.Value) value.Value { return value.Num(math.Min(num(a, 0), num(a, 1))) })
	n("max", func(a []value.Value) value.Value { return value.Num(math.Max(num(a, 0), num(a, 1))) })
	in.installMath(n)
	n("rnd", func(a []value.Value) value.Value {
		x := num(a, 0)
		if x == 0 {
			return value.Num(in.rng.Float64())
		}
		return value.Num(in.rng.Float64() * x)
	})
	n("rand", func(a []value.Value) value.Value {
		lo, hi := int(num(a, 0)), int(num(a, 1))
		if len(a) < 2 {
			hi = lo
			lo = 1
		}
		if hi < lo {
			lo, hi = hi, lo
		}
		return value.Num(float64(lo + in.rng.IntN(hi-lo+1)))
	})
	n("seedrnd", func(a []value.Value) value.Value {
		s := uint64(int64(num(a, 0)))
		in.rngSeed = int64(s)
		in.rng = newRNG(s, s^0xA0761D6478BD642F)
		return value.Num(0)
	})
	n("rndseed", func(a []value.Value) value.Value { return value.Num(float64(in.rngSeed)) })
	n("millisecs", func(a []value.Value) value.Value {
		return value.Num(float64(time.Since(in.start).Milliseconds()))
	})
	n("len", func(a []value.Value) value.Value { return value.Num(float64(len(str(a, 0)))) })
	n("left", func(a []value.Value) value.Value {
		s := str(a, 0)
		n := int(num(a, 1))
		if n < 0 {
			n = 0
		}
		if n > len(s) {
			n = len(s)
		}
		return value.Str(s[:n])
	})
	n("right", func(a []value.Value) value.Value {
		s := str(a, 0)
		n := int(num(a, 1))
		if n < 0 {
			n = 0
		}
		if n > len(s) {
			n = len(s)
		}
		return value.Str(s[len(s)-n:])
	})
	n("mid", func(a []value.Value) value.Value {
		s := str(a, 0)
		start := int(num(a, 1)) - 1
		n := int(num(a, 2))
		if start < 0 {
			start = 0
		}
		if start > len(s) {
			return value.Str("")
		}
		end := start + n
		if n == 0 || end > len(s) {
			end = len(s)
		}
		return value.Str(s[start:end])
	})
	n("chr", func(a []value.Value) value.Value { return value.Str(string(rune(int(num(a, 0))))) })
	n("asc", func(a []value.Value) value.Value {
		s := str(a, 0)
		if s == "" {
			return value.Num(0)
		}
		return value.Num(float64(s[0]))
	})
	n("str", func(a []value.Value) value.Value { return value.Str(a[0].String()) })
	n("instr", func(a []value.Value) value.Value {
		return value.Num(float64(strings.Index(str(a, 0), str(a, 1)) + 1))
	})
	n("lower", func(a []value.Value) value.Value { return value.Str(strings.ToLower(str(a, 0))) })
	n("upper", func(a []value.Value) value.Value { return value.Str(strings.ToUpper(str(a, 0))) })
	n("trim", func(a []value.Value) value.Value { return value.Str(strings.TrimSpace(str(a, 0))) })
	n("true", func(a []value.Value) value.Value { return value.Num(1) })
	n("false", func(a []value.Value) value.Value { return value.Num(0) })
	n("yes", func(a []value.Value) value.Value { return value.Num(1) })
	n("no", func(a []value.Value) value.Value { return value.Num(0) })
	n("null", func(a []value.Value) value.Value { return value.Num(0) })
	n("pi", func(a []value.Value) value.Value { return value.Num(math.Pi) })
	n("hex", func(a []value.Value) value.Value {
		if len(a) == 0 {
			return value.Num(0)
		}
		if a[0].Kind == value.KindNum {
			return a[0]
		}
		n, err := syntax.ParseHexInt(a[0].String())
		if err != nil {
			return value.Num(0)
		}
		return value.Num(float64(n))
	})
	for name, mode := range syntax.WeatherConstants {
		mode := mode
		n(name, func(a []value.Value) value.Value { return value.Str(mode) })
	}
	for name, kind := range syntax.NetConstants {
		kind := kind
		n(name, func(a []value.Value) value.Value { return value.Num(float64(kind)) })
	}
	for name, nconst := range syntax.PhysicsConstants {
		nconst := nconst
		n(name, func(a []value.Value) value.Value { return value.Num(nconst) })
	}
	n("createlist", func(a []value.Value) value.Value {
		id := in.nextList
		in.nextList++
		in.lists[id] = []value.Value{}
		v := value.Num(float64(id))
		v.TypeName = "list"
		return v
	})
	n("listadd", func(a []value.Value) value.Value {
		id := int(num(a, 0))
		in.lists[id] = append(in.lists[id], aValue(a, 1))
		return value.Num(float64(len(in.lists[id])))
	})
	n("listget", func(a []value.Value) value.Value {
		lst := in.lists[int(num(a, 0))]
		i := int(num(a, 1))
		if i < 0 || i >= len(lst) {
			return value.Num(0)
		}
		return lst[i]
	})
	n("listset", func(a []value.Value) value.Value {
		id := int(num(a, 0))
		lst := in.lists[id]
		i := int(num(a, 1))
		if i < 0 || i >= len(lst) {
			return value.Num(0)
		}
		lst[i] = aValue(a, 2)
		in.lists[id] = lst
		return value.Num(1)
	})
	n("listcount", func(a []value.Value) value.Value {
		return value.Num(float64(len(in.lists[int(num(a, 0))])))
	})
	n("listremove", func(a []value.Value) value.Value {
		id := int(num(a, 0))
		lst := in.lists[id]
		i := int(num(a, 1))
		if i < 0 || i >= len(lst) {
			return value.Num(0)
		}
		in.lists[id] = append(lst[:i], lst[i+1:]...)
		return value.Num(1)
	})
	n("createmap", func(a []value.Value) value.Value { return value.Map() })
	n("mapset", func(a []value.Value) value.Value {
		m := aValue(a, 0)
		if m.Kind != value.KindMap || m.Fields == nil {
			return value.Num(0)
		}
		m.SetField(str(a, 1), aValue(a, 2))
		return value.Num(1)
	})
	n("mapget", func(a []value.Value) value.Value {
		m := aValue(a, 0)
		if m.Kind != value.KindMap {
			return value.Num(0)
		}
		if v, ok := m.Field(str(a, 1)); ok {
			return v
		}
		return value.Num(0)
	})
	n("maphas", func(a []value.Value) value.Value {
		m := aValue(a, 0)
		if m.Kind != value.KindMap {
			return value.Num(0)
		}
		if _, ok := m.Field(str(a, 1)); ok {
			return value.Num(1)
		}
		return value.Num(0)
	})
	n("mapdelete", func(a []value.Value) value.Value {
		m := aValue(a, 0)
		if m.Kind != value.KindMap || m.Fields == nil {
			return value.Num(0)
		}
		delete(m.Fields, strings.ToLower(strings.TrimRight(str(a, 1), "$%#")))
		return value.Num(1)
	})
	n("mapcount", func(a []value.Value) value.Value {
		m := aValue(a, 0)
		if m.Kind != value.KindMap {
			return value.Num(0)
		}
		return value.Num(float64(len(m.Fields)))
	})
	n("copy", func(a []value.Value) value.Value { return aValue(a, 0).DeepCopy() })
	n("callback", func(a []value.Value) value.Value { return value.Func(str(a, 0)) })
	n("createstatemachine", func(a []value.Value) value.Value {
		id := in.nextMach
		in.nextMach++
		in.machines[id] = &stateMachine{fn: map[string]string{}}
		return value.Num(float64(id))
	})
	n("addstate", func(a []value.Value) value.Value {
		m := in.machines[int(num(a, 0))]
		if m == nil {
			return value.Num(0)
		}
		name := strings.ToLower(str(a, 1))
		m.fn[name] = lex.IdentKey(str(a, 2))
		if m.cur == "" {
			m.cur = name
		}
		return value.Num(1)
	})
	n("gostate", func(a []value.Value) value.Value {
		m := in.machines[int(num(a, 0))]
		if m == nil {
			return value.Num(0)
		}
		m.cur = strings.ToLower(str(a, 1))
		return value.Num(1)
	})
	n("statename", func(a []value.Value) value.Value {
		m := in.machines[int(num(a, 0))]
		if m == nil {
			return value.Str("")
		}
		return value.Str(m.cur)
	})
	n("updatestate", func(a []value.Value) value.Value {
		m := in.machines[int(num(a, 0))]
		if m == nil {
			return value.Num(0)
		}
		fn := m.fn[m.cur]
		if fn == "" {
			return value.Num(0)
		}
		if _, err := in.call(fn, nil); err != nil {
			return value.Num(0)
		}
		return value.Num(1)
	})
	n("arraysize", func(a []value.Value) value.Value {
		if len(a) < 1 || a[0].Kind != value.KindArray {
			return value.Num(0)
		}
		dim := 0
		if len(a) >= 2 {
			dim = int(a[1].Number())
			if dim > 0 {
				dim--
			}
		}
		return value.Num(float64(a[0].DimSize(dim)))
	})
	n("backbuffer", func(a []value.Value) value.Value { return value.Num(0) })
	n("frontbuffer", func(a []value.Value) value.Value { return value.Num(1) })
	n("setbuffer", func(a []value.Value) value.Value { return value.Num(0) })
}

func aValue(a []value.Value, i int) value.Value {
	if i >= len(a) {
		return value.Num(0)
	}
	return a[i]
}

func (in *Interp) CallNamed(name string, args []value.Value) (value.Value, error) {
	return in.call(lex.IdentKey(name), args)
}

// Reload replaces function and type declarations from a reparsed program.
func (in *Interp) Reload(prog *ast.Program) {
	if prog == nil {
		return
	}
	in.collectFuncs(prog.Stmts)
	in.fastCache = map[string]fastCode{}
}

func num(a []value.Value, i int) float64 {
	if i >= len(a) {
		return 0
	}
	return a[i].Number()
}

func str(a []value.Value, i int) string {
	if i >= len(a) {
		return ""
	}
	return a[i].String()
}

func deg(a []value.Value, i int) float64 {
	return num(a, i) * math.Pi / 180
}

func radToDeg(r float64) float64 { return r * 180 / math.Pi }

func entropySeed() (uint64, uint64) {
	var b [16]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		n := uint64(time.Now().UnixNano())
		return n, n ^ 0x9E3779B97F4A7C15
	}
	return binary.LittleEndian.Uint64(b[0:8]), binary.LittleEndian.Uint64(b[8:16])
}

func newRNG(s0, s1 uint64) *rand.Rand {
	if s0 == 0 && s1 == 0 {
		s0, s1 = 1, 1
	}
	return rand.New(rand.NewPCG(s0, s1))
}
