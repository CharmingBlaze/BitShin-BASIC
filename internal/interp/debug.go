package interp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"moonbasic/internal/ast"
	"moonbasic/internal/lex"
	"moonbasic/internal/parse"
	"moonbasic/internal/value"
)

type stepMode int

const (
	stepNone stepMode = iota
	stepInto
	stepOver
	stepOut
)

// DebugFrame is one call on the script call stack, innermost first.
type DebugFrame struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// DebugVar is one variable shown by the debugger.
type DebugVar struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// SetDebug pauses on the given source lines. in and out carry the debugger commands
// (c, s, n, o, bt, locals, globals, p expr, b line, d line, q).
func (in *Interp) SetDebug(lines []int, inR io.Reader, out io.Writer) {
	in.debug = true
	in.DebugLogs = true
	in.breaks = map[int]bool{}
	for _, ln := range lines {
		if ln > 0 {
			in.breaks[ln] = true
		}
	}
	if inR != nil {
		in.debugIn = bufio.NewReader(inR)
	}
	if out == nil {
		out = io.Discard
	}
	in.debugOut = out
}

// SetDebugJSON makes every pause print one "@debug {json}" line (stack, locals, globals)
// and p/eval print "@eval {json}". The IDE reads these; people read the plain form.
func (in *Interp) SetDebugJSON(on bool) { in.debugJSON = on }

// SetScript names the source file used in DebugLog, Assert, and stack frames.
func (in *Interp) SetScript(name string) { in.File = name }

// debugEnvOn reports whether MOON_DEBUG asks for DebugLog output.
func debugEnvOn() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MOON_DEBUG"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func (in *Interp) fileName() string {
	if in.File == "" {
		return "main"
	}
	return in.File
}

func (in *Interp) where() string {
	return fmt.Sprintf("%s:%d", in.fileName(), in.curLine)
}

// assertError carries its own file:line, so cmdFail does not wrap it again.
type assertError struct{ msg string }

func (e *assertError) Error() string { return e.msg }

func (in *Interp) installDebugBuiltins() {
	in.Register("debuglog", func(args []value.Value) (value.Value, error) {
		if !in.DebugLogs {
			return value.Num(0), nil
		}
		msg := ""
		if len(args) > 0 {
			msg = args[0].String()
		}
		level := "info"
		if len(args) > 1 {
			level = strings.ToLower(strings.TrimSpace(args[1].String()))
		}
		w := in.Out
		tag := "[debug]"
		switch level {
		case "warn", "warning":
			tag = "[debug:warn]"
			w = in.errOut()
		case "error", "err":
			tag = "[debug:error]"
			w = in.errOut()
		}
		fmt.Fprintf(w, "%s %s: %s\n", tag, in.where(), msg)
		return value.Num(0), nil
	})
	in.Register("assert", func(args []value.Value) (value.Value, error) {
		if len(args) > 0 && args[0].IsTrue() {
			return value.Num(1), nil
		}
		msg := "assertion failed"
		if len(args) > 1 && args[1].String() != "" {
			msg = args[1].String()
		}
		text := fmt.Sprintf("%s: Assert failed: %s", in.where(), msg)
		if in.debugOut != nil {
			in.debugPause(in.curLine, "assert", text)
		}
		return value.Num(0), &assertError{msg: text}
	})
	in.Register("debugbreak", func(args []value.Value) (value.Value, error) {
		if in.debugOut != nil {
			in.debugPause(in.curLine, "debugbreak", "")
		}
		return value.Num(0), nil
	})
}

// hasTry reports whether a Try block encloses the running statement, in this call or a caller.
func (in *Interp) hasTry() bool {
	has := func(fs []frame) bool {
		for i := range fs {
			if fs[i].kind == frameTry {
				return true
			}
		}
		return false
	}
	for _, fs := range in.outer {
		if has(fs) {
			return true
		}
	}
	return has(in.stack)
}

func (in *Interp) errOut() io.Writer {
	if in.ErrOut != nil {
		return in.ErrOut
	}
	return os.Stderr
}

// enterCall saves the caller's frames before a Function or Method runs on a fresh stack,
// so the call stack and step depth still see them.
func (in *Interp) enterCall(saved []frame) int {
	in.outer = append(in.outer, saved)
	return in.curLine
}

func (in *Interp) leaveCall(line int) {
	if n := len(in.outer); n > 0 {
		in.outer = in.outer[:n-1]
	}
	in.curLine = line
}

// callDepth counts Function and Method calls on the logical stack (1 = main program).
func (in *Interp) callDepth() int {
	d := 1
	count := func(fs []frame) {
		for i := range fs {
			if fs[i].kind == frameFunc {
				d++
			}
		}
	}
	for _, fs := range in.outer {
		count(fs)
	}
	count(in.stack)
	return d
}

// CallStack lists the active calls, innermost first, each with the line it is on.
func (in *Interp) CallStack() []DebugFrame {
	recs := []DebugFrame{{Name: "(main)", File: in.fileName()}}
	walk := func(fs []frame) {
		for i := range fs {
			f := &fs[i]
			if f.kind == frameFunc {
				recs = append(recs, DebugFrame{Name: in.displayFn(f.fn), File: in.fileName()})
			}
			if f.pc > 0 && f.pc <= len(f.stmts) {
				if ln, _ := f.stmts[f.pc-1].Pos(); ln > 0 {
					recs[len(recs)-1].Line = ln
				}
			}
		}
	}
	for _, fs := range in.outer {
		walk(fs)
	}
	walk(in.stack)
	if in.curLine > 0 {
		recs[len(recs)-1].Line = in.curLine
	}
	if len(recs) > 1 && recs[0].Line == 0 {
		recs = recs[1:]
	}
	out := make([]DebugFrame, len(recs))
	for i := range recs {
		out[i] = recs[len(recs)-1-i]
	}
	return out
}

// displayFn turns a call key back into the name as written (Outer, Pt.Bump).
func (in *Interp) displayFn(key string) string {
	if key == "" {
		return "(function)"
	}
	if fn := in.funcs[key]; fn != nil {
		return fn.Name
	}
	for k, fn := range in.methods {
		if strings.HasSuffix(k, "."+key) {
			if td := in.types[strings.TrimSuffix(k, "."+key)]; td != nil {
				return td.Name + "." + fn.Name
			}
			return fn.Name
		}
	}
	return key
}

var hiddenGlobals = map[string]bool{"true": true, "false": true, "yes": true, "no": true, "null": true}

// Locals lists the current Function's variables (empty at the top level).
func (in *Interp) Locals() []DebugVar {
	seen := map[string]bool{}
	var out []DebugVar
	for e := in.env(); e != nil && e != in.global; e = e.parent {
		for k, v := range e.vars {
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, debugVar(k, v))
		}
	}
	sortVars(out)
	return out
}

// Globals lists the main program's variables.
func (in *Interp) Globals() []DebugVar {
	var out []DebugVar
	for k, v := range in.global.vars {
		if hiddenGlobals[k] {
			continue
		}
		out = append(out, debugVar(k, v))
	}
	sortVars(out)
	return out
}

func sortVars(vs []DebugVar) {
	sort.Slice(vs, func(i, j int) bool { return vs[i].Name < vs[j].Name })
}

func debugVar(name string, v value.Value) DebugVar {
	return DebugVar{Name: name, Kind: kindName(v), Value: DebugValue(v)}
}

func kindName(v value.Value) string {
	switch v.Kind {
	case value.KindStr:
		return "string"
	case value.KindArray:
		return "array"
	case value.KindStruct:
		return "type"
	case value.KindVec:
		return "vec"
	case value.KindMap:
		return "map"
	case value.KindFunc:
		return "function"
	default:
		return "number"
	}
}

const debugPreview = 8

// DebugValue shows a value on one line: strings quoted, arrays and Types summarized.
func DebugValue(v value.Value) string { return debugValue(v, 0) }

func debugValue(v value.Value, depth int) string {
	switch v.Kind {
	case value.KindStr:
		return strconv.Quote(v.Str)
	case value.KindArray:
		dims := make([]string, len(v.Dims))
		for i, d := range v.Dims {
			dims[i] = strconv.Itoa(d)
		}
		head := fmt.Sprintf("Array(%s)", strings.Join(dims, ","))
		if len(v.Dims) == 0 {
			head = fmt.Sprintf("Array[%d]", len(v.Elems))
		}
		if depth > 0 {
			return head
		}
		parts := make([]string, 0, debugPreview)
		for i, e := range v.Elems {
			if i == debugPreview {
				parts = append(parts, fmt.Sprintf("… %d more", len(v.Elems)-debugPreview))
				break
			}
			parts = append(parts, debugValue(e, depth+1))
		}
		return head + " [" + strings.Join(parts, ", ") + "]"
	case value.KindStruct, value.KindMap:
		head := v.TypeName
		if v.Kind == value.KindMap {
			head = "Map"
		}
		if depth > 0 {
			return head + " {…}"
		}
		keys := make([]string, 0, len(v.Fields))
		for k := range v.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for i, k := range keys {
			if i == debugPreview {
				parts = append(parts, fmt.Sprintf("… %d more", len(keys)-debugPreview))
				break
			}
			parts = append(parts, k+": "+debugValue(v.Fields[k], depth+1))
		}
		return head + " {" + strings.Join(parts, ", ") + "}"
	case value.KindFunc:
		return "Function " + v.String()
	default:
		return v.String()
	}
}

// Eval evaluates a watch expression in the paused frame.
func (in *Interp) Eval(expr string) (value.Value, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return value.Value{}, fmt.Errorf("empty expression")
	}
	prog, err := parse.Parse("dbgwatch_value = " + expr + "\n")
	if err != nil {
		return value.Value{}, err
	}
	if len(prog.Stmts) != 1 {
		return value.Value{}, fmt.Errorf("not an expression: %s", expr)
	}
	as, ok := prog.Stmts[0].(*ast.AssignStmt)
	if !ok || as.Value == nil || lex.IdentKey(as.Name) != "dbgwatch_value" || len(as.Index) > 0 || len(as.Fields) > 0 {
		return value.Value{}, fmt.Errorf("not an expression: %s", expr)
	}
	savedErr, savedStatus, savedRet, savedLine := in.err, in.status, in.ret, in.curLine
	savedMode := in.stepMode
	in.stepMode = stepNone
	in.evaluating++
	v, err := in.eval(as.Value)
	in.evaluating--
	in.err, in.status, in.ret, in.curLine = savedErr, savedStatus, savedRet, savedLine
	in.stepMode = savedMode
	return v, err
}

func (in *Interp) maybeBreak(s ast.Stmt, line int) {
	if in.debugOut == nil || line < 1 || in.evaluating > 0 {
		return
	}
	if in.waitStmt != nil {
		if s == in.waitStmt {
			return
		}
		in.waitStmt = nil
	}
	hit := in.breaks[line]
	reason := "breakpoint"
	switch in.stepMode {
	case stepInto:
		hit, reason = true, "step"
	case stepOver:
		if in.callDepth() <= in.stepDepth {
			hit, reason = true, "step"
		}
	case stepOut:
		if in.callDepth() < in.stepDepth {
			hit, reason = true, "step"
		}
	}
	if hit {
		in.debugPause(line, reason, "")
	}
}

type debugState struct {
	Reason  string       `json:"reason"`
	Message string       `json:"message,omitempty"`
	File    string       `json:"file"`
	Line    int          `json:"line"`
	Depth   int          `json:"depth"`
	Stack   []DebugFrame `json:"stack"`
	Locals  []DebugVar   `json:"locals"`
	Globals []DebugVar   `json:"globals"`
}

func (in *Interp) emitJSON(tag string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(in.debugOut, "%s %s\n", tag, b)
}

func printVars(w io.Writer, title string, vs []DebugVar) {
	fmt.Fprintf(w, "%s:\n", title)
	if len(vs) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, v := range vs {
		fmt.Fprintf(w, "  %s = %s\n", v.Name, v.Value)
	}
}

func (in *Interp) printStack(w io.Writer) {
	for i, f := range in.CallStack() {
		fmt.Fprintf(w, "#%d %s %s:%d\n", i, f.Name, f.File, f.Line)
	}
}

func (in *Interp) debugPause(line int, reason, msg string) {
	in.stepMode = stepNone
	in.curLine = line
	out := in.debugOut
	if in.debugJSON {
		locals := in.Locals()
		if locals == nil {
			locals = []DebugVar{}
		}
		globals := in.Globals()
		if globals == nil {
			globals = []DebugVar{}
		}
		in.emitJSON("@debug", debugState{
			Reason: reason, Message: msg, File: in.fileName(), Line: line,
			Depth: in.callDepth(), Stack: in.CallStack(), Locals: locals, Globals: globals,
		})
	} else {
		fmt.Fprintf(out, "break %d\n", line)
		if msg != "" {
			fmt.Fprintln(out, msg)
		}
		vars := in.Locals()
		if in.env() == in.global {
			vars = in.Globals()
		}
		for _, v := range vars {
			fmt.Fprintf(out, "  %s = %s\n", v.Name, v.Value)
		}
	}
	if in.debugIn == nil {
		return
	}
	for {
		if !in.debugJSON {
			fmt.Fprint(out, "debug> ")
		}
		text, err := in.debugIn.ReadString('\n')
		if err != nil && strings.TrimSpace(text) == "" {
			return
		}
		fields := strings.Fields(text)
		if len(fields) == 0 {
			continue
		}
		rest := strings.TrimSpace(strings.TrimSpace(text)[len(fields[0]):])
		switch strings.ToLower(fields[0]) {
		case "c", "continue", "g", "go":
			return
		case "s", "step", "si", "into":
			in.stepMode = stepInto
			return
		case "n", "next", "over":
			in.stepMode, in.stepDepth = stepOver, in.callDepth()
			return
		case "o", "out", "finish":
			in.stepMode, in.stepDepth = stepOut, in.callDepth()
			return
		case "bt", "stack", "where":
			if in.debugJSON {
				in.emitJSON("@stack", in.CallStack())
			} else {
				in.printStack(out)
			}
		case "locals":
			printVars(out, "locals", in.Locals())
		case "globals":
			printVars(out, "globals", in.Globals())
		case "vars", "v":
			printVars(out, "locals", in.Locals())
			printVars(out, "globals", in.Globals())
		case "p", "print", "w", "watch", "eval", "e":
			if rest == "" {
				continue
			}
			v, err := in.Eval(rest)
			if in.debugJSON {
				res := map[string]string{"expr": rest}
				if err != nil {
					res["error"] = err.Error()
				} else {
					res["value"] = DebugValue(v)
					res["kind"] = kindName(v)
				}
				in.emitJSON("@eval", res)
			} else if err != nil {
				fmt.Fprintf(out, "%s: %v\n", rest, err)
			} else {
				fmt.Fprintf(out, "%s = %s\n", rest, DebugValue(v))
			}
		case "b", "break":
			for _, n := range parseLines(rest) {
				in.breaks[n] = true
			}
		case "d", "delete", "clear":
			if rest == "" || rest == "*" {
				in.breaks = map[int]bool{}
			}
			for _, n := range parseLines(rest) {
				delete(in.breaks, n)
			}
		case "q", "quit":
			in.err = fmt.Errorf("stopped at line %d", line)
			in.status = StatusEnd
			in.stack = nil
			in.debugIn = nil
			in.debugOut = nil
			return
		default:
			fmt.Fprintln(out, "commands: c continue, s step into, n step over, o step out, bt call stack, locals, globals, p expr, b line, d line, q quit")
		}
	}
}

func parseLines(s string) []int {
	var out []int
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		if n, err := strconv.Atoi(part); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	return out
}
