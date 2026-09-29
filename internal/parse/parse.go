// Package parse turns tokens into an AST (statements, expressions, functions).
package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bitshinbasic/internal/ast"
	"bitshinbasic/internal/lex"
)

type Parser struct {
	lx   *lex.Lexer
	tok  lex.Token
	base string
}

func ParseFile(path string) (*ast.Program, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p := New(string(b), filepath.Dir(path))
	return p.Parse()
}

func Parse(src string) (*ast.Program, error) {
	return New(src, ".").Parse()
}

func New(src, base string) *Parser {
	p := &Parser{lx: lex.New(src), base: base}
	p.next()
	return p
}

func (p *Parser) Parse() (*ast.Program, error) {
	p.skipNL()
	stmts, err := p.statements(endProgram)
	if err != nil {
		return nil, err
	}
	return &ast.Program{Stmts: stmts}, nil
}

func (p *Parser) next() { p.tok = p.lx.Next() }

func (p *Parser) skipNL() {
	for p.tok.Kind == lex.Newline {
		p.next()
	}
}

func (p *Parser) acceptNL() {
	for p.tok.Kind == lex.Newline || p.tok.Kind == lex.Colon {
		p.next()
	}
}

func (p *Parser) errorf(format string, args ...any) error {
	return fmt.Errorf("line %d:%d: "+format, append([]any{p.tok.Line, p.tok.Col}, args...)...)
}

func identKey(t lex.Token) string {
	if t.Kind != lex.Ident {
		return ""
	}
	return lex.IdentKey(t.Lit)
}

func (p *Parser) is(k string) bool { return identKey(p.tok) == k }

func (p *Parser) eatIdent(k string) bool {
	if p.is(k) {
		p.next()
		return true
	}
	return false
}

type stopper func(*Parser) bool

func endProgram(p *Parser) bool { return p.tok.Kind == lex.EOF }

func endIf(p *Parser) bool {
	k := identKey(p.tok)
	return k == "endif" || k == "else" || k == "elseif" || (k == "end" && isEndIf(p))
}

func isEndIf(p *Parser) bool {
	n := p.lx.Peek()
	return lex.IdentKey(n.Lit) == "if"
}

func endWhile(p *Parser) bool { return p.is("wend") || p.is("endwhile") }

func endFor(p *Parser) bool { return p.is("next") }

func endRepeat(p *Parser) bool { return p.is("until") }

func endFunc(p *Parser) bool {
	if p.is("endfunction") {
		return true
	}
	return p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "function"
}

func endSelect(p *Parser) bool {
	k := identKey(p.tok)
	if k == "case" || k == "default" || k == "endselect" {
		return true
	}
	return k == "end" && lex.IdentKey(p.lx.Peek().Lit) == "select"
}

func (p *Parser) statements(stop stopper) ([]ast.Stmt, error) {
	var out []ast.Stmt
	for {
		p.acceptNL()
		if stop(p) || p.tok.Kind == lex.EOF {
			return out, nil
		}
		s, err := p.statement()
		if err != nil {
			return nil, err
		}
		if s != nil {
			out = append(out, s)
		}
	}
}

func (p *Parser) statement() (ast.Stmt, error) {
	p.acceptNL()
	if p.tok.Kind == lex.EOF {
		return nil, nil
	}
	if p.tok.Kind != lex.Ident {
		return nil, p.errorf("expected statement, got %s", p.tok.Kind)
	}
	pos := astPos(p.tok)
	key := identKey(p.tok)

	switch key {
	case "if":
		return p.parseIf()
	case "while":
		return p.parseWhile()
	case "for":
		return p.parseFor()
	case "repeat":
		return p.parseRepeat()
	case "function":
		return p.parseFunc()
	case "select":
		return p.parseSelect()
	case "const":
		return p.parseConsts()
	case "enum":
		return p.parseEnum()
	case "strict":
		p.next()
		return &ast.StrictStmt{Src: pos}, nil
	case "try":
		return p.parseTry()
	case "redim":
		return p.parseRedim()
	case "data":
		return p.parseData()
	case "read":
		return p.parseRead()
	case "restore":
		p.next()
		return &ast.RestoreStmt{Src: pos}, nil
	case "return":
		p.next()
		if startsExpr(p.tok) {
			e, err := p.exprList()
			if err != nil {
				return nil, err
			}
			return &ast.ReturnStmt{Src: pos, Value: e}, nil
		}
		return &ast.ReturnStmt{Src: pos}, nil
	case "dim":
		return p.parseDim()
	case "global":
		return p.parseDeclNames(true)
	case "local":
		return p.parseDeclNames(false)
	case "include":
		p.next()
		if p.tok.Kind != lex.String {
			return nil, p.errorf("Include expects a filename string")
		}
		path := p.tok.Lit
		p.next()
		return &ast.IncludeStmt{Src: pos, Path: path}, nil
	case "import":
		return p.parseImport()
	case "type", "struct":
		return p.parseType()
	case "method":
		return p.parseMethod("")
	case "namespace":
		return p.parseNamespace()
	case "end":
		n := p.lx.Peek()
		nk := lex.IdentKey(n.Lit)
		if nk == "if" || nk == "function" || nk == "while" || nk == "select" || nk == "type" || nk == "struct" || nk == "method" || nk == "namespace" || nk == "enum" || nk == "try" {
			return nil, p.errorf("unexpected End %s", n.Lit)
		}
		p.next()
		return &ast.EndStmt{Src: pos}, nil
	case "exit":
		p.next()
		fn := p.eatIdent("function")
		return &ast.ExitStmt{Src: pos, Function: fn}, nil
	}

	name := p.tok.Lit
	p.next()

	if p.tok.Kind == lex.Comma {
		names := []string{name}
		for p.tok.Kind == lex.Comma {
			p.next()
			if p.tok.Kind != lex.Ident {
				return nil, p.errorf("expected name")
			}
			names = append(names, p.tok.Lit)
			p.next()
		}
		if p.tok.Kind != lex.Eq {
			return nil, p.errorf("expected =")
		}
		p.next()
		e, err := p.exprList()
		if err != nil {
			return nil, err
		}
		return &ast.AssignStmt{Src: pos, Name: names[0], Names: names, Value: e}, nil
	}

	if p.tok.Kind == lex.Dot {
		return p.parseDottedStmt(pos, name)
	}

	// assignment: name = expr  OR name(i) = expr OR name[i] = expr
	if p.tok.Kind == lex.Eq {
		p.next()
		e, err := p.exprList()
		if err != nil {
			return nil, err
		}
		return &ast.AssignStmt{Src: pos, Name: name, Value: e}, nil
	}

	var index []ast.Expr
	if p.tok.Kind == lex.LParen || p.tok.Kind == lex.LBracket {
		open := p.tok.Kind
		closeK := lex.RParen
		if open == lex.LBracket {
			closeK = lex.RBracket
		}
		p.next()
		args, err := p.argList(closeK)
		if err != nil {
			return nil, err
		}
		var fields []string
		for p.tok.Kind == lex.Dot {
			p.next()
			if p.tok.Kind != lex.Ident {
				return nil, p.errorf("expected field name")
			}
			fields = append(fields, p.tok.Lit)
			p.next()
		}
		if p.tok.Kind == lex.Eq {
			p.next()
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			return &ast.AssignStmt{Src: pos, Name: name, Index: args, Fields: fields, Value: e}, nil
		}
		if len(fields) > 0 {
			return nil, p.errorf("expected =")
		}
		// function call with parens — allow CreateCube().Scale().Position()
		call := &ast.CallExpr{Src: pos, Name: name, Args: args}
		if p.tok.Kind == lex.Dot {
			e, err := p.postfix(call)
			if err != nil {
				return nil, err
			}
			return &ast.ExprStmt{Src: pos, Value: e}, nil
		}
		return &ast.CallStmt{Src: pos, Name: name, Args: args}, nil
	}

	// command-style call: Name arg, arg
	if startsExpr(p.tok) && p.tok.Kind != lex.Newline && p.tok.Kind != lex.Colon && p.tok.Kind != lex.EOF {
		args, err := p.bareArgs()
		if err != nil {
			return nil, err
		}
		_ = index
		return &ast.CallStmt{Src: pos, Name: name, Args: args}, nil
	}
	return &ast.CallStmt{Src: pos, Name: name}, nil
}

func (p *Parser) parseIf() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	cond, err := p.expr()
	if err != nil {
		return nil, err
	}
	p.eatIdent("then")

	// single-line THEN
	if p.tok.Kind != lex.Newline && p.tok.Kind != lex.EOF && !p.is("endif") && !(p.is("end") && isEndIf(p)) {
		s, err := p.statement()
		if err != nil {
			return nil, err
		}
		var then []ast.Stmt
		if s != nil {
			then = []ast.Stmt{s}
		}
		st := &ast.IfStmt{Src: pos, Cond: cond, Then: then}
		if p.is("else") {
			p.next()
			e, err := p.statement()
			if err != nil {
				return nil, err
			}
			if e != nil {
				st.Else = []ast.Stmt{e}
			}
		}
		return st, nil
	}

	then, err := p.statements(endIf)
	if err != nil {
		return nil, err
	}
	st := &ast.IfStmt{Src: pos, Cond: cond, Then: then}
	for p.is("elseif") || (p.is("else") && lex.IdentKey(p.lx.Peek().Lit) == "if") {
		p.next()
		if p.is("if") {
			p.next()
		}
		c, err := p.expr()
		if err != nil {
			return nil, err
		}
		p.eatIdent("then")
		body, err := p.statements(endIf)
		if err != nil {
			return nil, err
		}
		st.ElseIf = append(st.ElseIf, ast.ElseIf{Cond: c, Body: body})
	}
	if p.is("else") {
		p.next()
		els, err := p.statements(endIf)
		if err != nil {
			return nil, err
		}
		st.Else = els
	}
	if p.is("endif") {
		p.next()
		return st, nil
	}
	if p.is("end") && isEndIf(p) {
		p.next()
		p.next()
		return st, nil
	}
	return nil, p.errorf("expected EndIf")
}

func (p *Parser) parseWhile() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	p.eatIdent("then") // tolerate While Then
	cond, err := p.expr()
	if err != nil {
		return nil, err
	}
	body, err := p.statements(endWhile)
	if err != nil {
		return nil, err
	}
	if !p.eatIdent("wend") && !p.eatIdent("endwhile") {
		if p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "while" {
			p.next()
			p.next()
		} else {
			return nil, p.errorf("expected Wend")
		}
	}
	return &ast.WhileStmt{Src: pos, Cond: cond, Body: body}, nil
}

func (p *Parser) parseFor() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	p.eatIdent("each")
	if p.tok.Kind != lex.Ident {
		return nil, p.errorf("For expects a variable")
	}
	name := p.tok.Lit
	p.next()
	if p.eatIdent("in") {
		seq, err := p.expr()
		if err != nil {
			return nil, err
		}
		body, err := p.statements(endFor)
		if err != nil {
			return nil, err
		}
		if !p.eatIdent("next") {
			return nil, p.errorf("expected Next")
		}
		if p.tok.Kind == lex.Ident && !isKeyword(identKey(p.tok)) {
			p.next()
		}
		return &ast.ForStmt{Src: pos, Var: name, In: seq, Body: body}, nil
	}
	if p.tok.Kind != lex.Eq {
		return nil, p.errorf("For expects = or In")
	}
	p.next()
	start, err := p.expr()
	if err != nil {
		return nil, err
	}
	if !p.eatIdent("to") {
		return nil, p.errorf("For expects To")
	}
	end, err := p.expr()
	if err != nil {
		return nil, err
	}
	var step ast.Expr
	if p.eatIdent("step") {
		step, err = p.expr()
		if err != nil {
			return nil, err
		}
	}
	body, err := p.statements(endFor)
	if err != nil {
		return nil, err
	}
	if !p.eatIdent("next") {
		return nil, p.errorf("expected Next")
	}
	if p.tok.Kind == lex.Ident && !isKeyword(identKey(p.tok)) {
		p.next()
	}
	return &ast.ForStmt{Src: pos, Var: name, Start: start, End: end, Step: step, Body: body}, nil
}

func (p *Parser) parseRepeat() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	body, err := p.statements(endRepeat)
	if err != nil {
		return nil, err
	}
	if !p.eatIdent("until") {
		return nil, p.errorf("expected Until")
	}
	cond, err := p.expr()
	if err != nil {
		return nil, err
	}
	return &ast.RepeatStmt{Src: pos, Body: body, Until: cond}, nil
}

func (p *Parser) parseFunc() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	if p.tok.Kind != lex.Ident {
		return nil, p.errorf("Function expects a name")
	}
	name := p.tok.Lit
	p.next()
	params, defaults, err := p.parseParamList()
	if err != nil {
		return nil, err
	}
	body, err := p.statements(endFunc)
	if err != nil {
		return nil, err
	}
	if p.eatIdent("endfunction") {
		return &ast.FuncDecl{Src: pos, Name: name, Params: params, Defaults: defaults, Body: body}, nil
	}
	if p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "function" {
		p.next()
		p.next()
		return &ast.FuncDecl{Src: pos, Name: name, Params: params, Defaults: defaults, Body: body}, nil
	}
	return nil, p.errorf("expected End Function")
}

func (p *Parser) parseDim() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	if p.tok.Kind != lex.Ident {
		return nil, p.errorf("Dim expects a name")
	}
	name := p.tok.Lit
	p.next()
	if p.tok.Kind != lex.LParen && p.tok.Kind != lex.LBracket {
		return nil, p.errorf("Dim expects (size)")
	}
	closeK := lex.RParen
	if p.tok.Kind == lex.LBracket {
		closeK = lex.RBracket
	}
	p.next()
	sizes, err := p.argList(closeK)
	if err != nil {
		return nil, err
	}
	typeName := ""
	if p.eatIdent("as") {
		if p.tok.Kind != lex.Ident {
			return nil, p.errorf("Dim As expects a type name")
		}
		typeName = p.tok.Lit
		p.next()
	}
	return &ast.DimStmt{Src: pos, Name: name, Sizes: sizes, TypeName: typeName}, nil
}

func (p *Parser) parseParamList() (params []string, defaults []ast.Expr, err error) {
	if p.tok.Kind != lex.LParen {
		return nil, nil, nil
	}
	p.next()
	seenDef := false
	for p.tok.Kind != lex.RParen && p.tok.Kind != lex.EOF {
		if p.tok.Kind != lex.Ident {
			return nil, nil, p.errorf("expected parameter name")
		}
		params = append(params, p.tok.Lit)
		p.next()
		if p.tok.Kind == lex.Eq {
			p.next()
			e, err := p.expr()
			if err != nil {
				return nil, nil, err
			}
			defaults = append(defaults, e)
			seenDef = true
		} else {
			if seenDef {
				return nil, nil, p.errorf("default parameters must come last")
			}
			defaults = append(defaults, nil)
		}
		if p.tok.Kind == lex.Comma {
			p.next()
		}
	}
	if p.tok.Kind != lex.RParen {
		return nil, nil, p.errorf("expected )")
	}
	p.next()
	return params, defaults, nil
}

func (p *Parser) parseDeclNames(global bool) (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	var names []string
	var values []ast.Expr
	for p.tok.Kind == lex.Ident {
		names = append(names, p.tok.Lit)
		p.next()
		var val ast.Expr
		if p.tok.Kind == lex.Eq {
			p.next()
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			val = e
		}
		values = append(values, val)
		if p.tok.Kind == lex.Comma {
			p.next()
			continue
		}
		break
	}
	if global {
		return &ast.GlobalStmt{Src: pos, Names: names, Values: values}, nil
	}
	return &ast.LocalStmt{Src: pos, Names: names, Values: values}, nil
}

func (p *Parser) parseTry() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	stopTry := func(p *Parser) bool {
		return p.is("catch") || p.is("endtry") || (p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "try")
	}
	body, err := p.statements(stopTry)
	if err != nil {
		return nil, err
	}
	st := &ast.TryStmt{Src: pos, Body: body}
	if p.is("catch") {
		p.next()
		if p.tok.Kind == lex.Ident {
			st.ErrVar = p.tok.Lit
			p.next()
		}
		catch, err := p.statements(func(p *Parser) bool {
			return p.is("endtry") || (p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "try")
		})
		if err != nil {
			return nil, err
		}
		st.Catch = catch
	}
	if p.eatIdent("endtry") {
		return st, nil
	}
	if p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "try" {
		p.next()
		p.next()
		return st, nil
	}
	return nil, p.errorf("expected End Try")
}

func (p *Parser) parseEnum() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	name := ""
	if p.tok.Kind == lex.Ident {
		name = p.tok.Lit
		p.next()
	}
	st := &ast.EnumStmt{Src: pos, Name: name}
	p.acceptNL()
	for {
		p.acceptNL()
		if p.is("endenum") || (p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "enum") || p.tok.Kind == lex.EOF {
			break
		}
		if p.tok.Kind != lex.Ident {
			return nil, p.errorf("Enum expects a name")
		}
		st.Names = append(st.Names, p.tok.Lit)
		p.next()
		var val ast.Expr
		if p.tok.Kind == lex.Eq {
			p.next()
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			val = e
		}
		st.Values = append(st.Values, val)
		p.acceptNL()
	}
	if p.eatIdent("endenum") {
		return st, nil
	}
	if p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "enum" {
		p.next()
		p.next()
		return st, nil
	}
	return nil, p.errorf("expected End Enum")
}

func (p *Parser) parseConsts() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	s := &ast.ConstStmt{Src: pos}
	for p.tok.Kind == lex.Ident {
		s.Names = append(s.Names, p.tok.Lit)
		p.next()
		if p.tok.Kind != lex.Eq {
			return nil, p.errorf("Const expects =")
		}
		p.next()
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		s.Values = append(s.Values, e)
		if p.tok.Kind == lex.Comma {
			p.next()
			continue
		}
		break
	}
	return s, nil
}

func (p *Parser) parseRedim() (ast.Stmt, error) {
	st, err := p.parseDim()
	if err != nil {
		return nil, err
	}
	d := st.(*ast.DimStmt)
	d.Redim = true
	return d, nil
}

func (p *Parser) parseSelect() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	val, err := p.expr()
	if err != nil {
		return nil, err
	}
	st := &ast.SelectStmt{Src: pos, Value: val}
	p.acceptNL()
	for {
		p.acceptNL()
		if p.is("endselect") || (p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "select") || p.tok.Kind == lex.EOF {
			break
		}
		if p.is("default") {
			p.next()
			body, err := p.statements(endSelect)
			if err != nil {
				return nil, err
			}
			st.Default = body
			continue
		}
		if !p.is("case") {
			return nil, p.errorf("expected Case, Default, or End Select")
		}
		p.next()
		cl := ast.CaseClause{}
		for {
			low, err := p.expr()
			if err != nil {
				return nil, err
			}
			item := ast.CaseItem{Low: low}
			if p.eatIdent("to") {
				hi, err := p.expr()
				if err != nil {
					return nil, err
				}
				item.High = hi
			}
			cl.Items = append(cl.Items, item)
			if p.tok.Kind == lex.Comma {
				p.next()
				continue
			}
			break
		}
		body, err := p.statements(endSelect)
		if err != nil {
			return nil, err
		}
		cl.Body = body
		st.Cases = append(st.Cases, cl)
	}
	if p.eatIdent("endselect") {
		return st, nil
	}
	if p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "select" {
		p.next()
		p.next()
		return st, nil
	}
	return nil, p.errorf("expected End Select")
}

func (p *Parser) parseData() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	vals, err := p.bareArgs()
	if err != nil {
		return nil, err
	}
	return &ast.DataStmt{Src: pos, Values: vals}, nil
}

func (p *Parser) parseRead() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	s := &ast.ReadStmt{Src: pos}
	for p.tok.Kind == lex.Ident {
		s.Names = append(s.Names, p.tok.Lit)
		p.next()
		var qual []string
		for p.tok.Kind == lex.Dot {
			p.next()
			if p.tok.Kind != lex.Ident {
				return nil, p.errorf("expected field name")
			}
			qual = append(qual, p.tok.Lit)
			p.next()
		}
		s.Quals = append(s.Quals, qual)
		if p.tok.Kind == lex.Comma {
			p.next()
			continue
		}
		break
	}
	return s, nil
}

func (p *Parser) exprList() (ast.Expr, error) {
	e, err := p.expr()
	if err != nil {
		return nil, err
	}
	if p.tok.Kind != lex.Comma {
		return e, nil
	}
	line, col := e.Pos()
	elems := []ast.Expr{e}
	for p.tok.Kind == lex.Comma {
		p.next()
		n, err := p.expr()
		if err != nil {
			return nil, err
		}
		elems = append(elems, n)
	}
	return &ast.VecExpr{Src: ast.Src{Line: line, Col: col}, Elems: elems}, nil
}

func endType(p *Parser) bool {
	k := identKey(p.tok)
	if k == "endtype" || k == "endstruct" {
		return true
	}
	if k == "end" {
		n := lex.IdentKey(p.lx.Peek().Lit)
		return n == "type" || n == "struct"
	}
	return false
}

func endMethod(p *Parser) bool {
	k := identKey(p.tok)
	if k == "endmethod" || k == "endfunction" {
		return true
	}
	if k == "end" {
		n := lex.IdentKey(p.lx.Peek().Lit)
		return n == "method" || n == "function"
	}
	return false
}

func endNamespace(p *Parser) bool {
	if p.is("endnamespace") {
		return true
	}
	return p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == "namespace"
}

func (p *Parser) eatEnd(word string) bool {
	if p.eatIdent("end" + word) {
		return true
	}
	if p.is("end") && lex.IdentKey(p.lx.Peek().Lit) == word {
		p.next()
		p.next()
		return true
	}
	return false
}

func (p *Parser) parseImport() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	if p.tok.Kind != lex.String {
		return nil, p.errorf("Import expects a filename string")
	}
	path := p.tok.Lit
	p.next()
	as := ""
	if p.eatIdent("as") {
		if p.tok.Kind != lex.Ident {
			return nil, p.errorf("Import As expects a name")
		}
		as = p.tok.Lit
		p.next()
	}
	return &ast.ImportStmt{Src: pos, Path: path, As: as}, nil
}

func (p *Parser) parseType() (ast.Stmt, error) {
	pos := astPos(p.tok)
	kind := identKey(p.tok)
	p.next()
	if p.tok.Kind != lex.Ident {
		return nil, p.errorf("%s expects a name", strings.Title(kind))
	}
	name := p.tok.Lit
	p.next()
	st := &ast.TypeDecl{Src: pos, Name: name}
	for {
		p.acceptNL()
		if endType(p) || p.tok.Kind == lex.EOF {
			break
		}
		if p.is("method") || p.is("function") {
			m, err := p.parseMethod(name)
			if err != nil {
				return nil, err
			}
			switch t := m.(type) {
			case *ast.FuncDecl:
				st.Methods = append(st.Methods, t)
			case *ast.MethodDecl:
				st.Methods = append(st.Methods, &ast.FuncDecl{Src: t.Src, Name: t.Name, Params: t.Params, Defaults: t.Defaults, Body: t.Body})
			}
			continue
		}
		if p.eatIdent("field") {
			for p.tok.Kind == lex.Ident {
				st.Fields = append(st.Fields, ast.TypeField{Name: p.tok.Lit})
				p.next()
				if p.tok.Kind == lex.Comma {
					p.next()
					continue
				}
				break
			}
			continue
		}
		if p.tok.Kind == lex.Ident {
			st.Fields = append(st.Fields, ast.TypeField{Name: p.tok.Lit})
			p.next()
			continue
		}
		return nil, p.errorf("expected Field, Method, or End %s", strings.Title(kind))
	}
	if kind == "struct" {
		if !p.eatEnd("struct") {
			return nil, p.errorf("expected End Struct")
		}
	} else if !p.eatEnd("type") {
		return nil, p.errorf("expected End Type")
	}
	return st, nil
}

func (p *Parser) parseMethod(recv string) (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	if p.tok.Kind != lex.Ident {
		return nil, p.errorf("Method expects a name")
	}
	name := p.tok.Lit
	p.next()
	if p.tok.Kind == lex.Dot {
		recv = name
		p.next()
		if p.tok.Kind != lex.Ident {
			return nil, p.errorf("Method expects Type.Name")
		}
		name = p.tok.Lit
		p.next()
	}
	params, defaults, err := p.parseParamList()
	if err != nil {
		return nil, err
	}
	body, err := p.statements(endMethod)
	if err != nil {
		return nil, err
	}
	if !p.eatEnd("method") && !p.eatEnd("function") {
		return nil, p.errorf("expected End Method")
	}
	if recv == "" {
		return nil, p.errorf("Method needs a type: Method Type.Name()")
	}
	return &ast.MethodDecl{Src: pos, Recv: recv, Name: name, Params: params, Defaults: defaults, Body: body}, nil
}

func (p *Parser) parseNamespace() (ast.Stmt, error) {
	pos := astPos(p.tok)
	p.next()
	if p.tok.Kind != lex.Ident {
		return nil, p.errorf("Namespace expects a name")
	}
	name := p.tok.Lit
	p.next()
	body, err := p.statements(endNamespace)
	if err != nil {
		return nil, err
	}
	if !p.eatEnd("namespace") {
		return nil, p.errorf("expected End Namespace")
	}
	return &ast.NamespaceDecl{Src: pos, Name: name, Stmts: body}, nil
}

func (p *Parser) parseDottedStmt(pos ast.Src, name string) (ast.Stmt, error) {
	var fields []string
	for p.tok.Kind == lex.Dot {
		p.next()
		if p.tok.Kind != lex.Ident {
			return nil, p.errorf("expected name after .")
		}
		fields = append(fields, p.tok.Lit)
		p.next()
	}
	if p.tok.Kind == lex.Eq {
		p.next()
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		return &ast.AssignStmt{Src: pos, Name: name, Fields: fields, Value: e}, nil
	}
	var args []ast.Expr
	paren := false
	if p.tok.Kind == lex.LParen {
		paren = true
		p.next()
		var err error
		args, err = p.argList(lex.RParen)
		if err != nil {
			return nil, err
		}
	} else if startsExpr(p.tok) && p.tok.Kind != lex.Newline && p.tok.Kind != lex.Colon && p.tok.Kind != lex.EOF {
		var err error
		args, err = p.bareArgs()
		if err != nil {
			return nil, err
		}
	}
	method := fields[len(fields)-1]
	path := fields[:len(fields)-1]
	return &ast.MethodStmt{Src: pos, Object: name, Path: path, Method: method, Args: args, Paren: paren}, nil
}

func startsExpr(t lex.Token) bool {
	switch t.Kind {
	case lex.Ident, lex.Number, lex.String, lex.Plus, lex.Minus, lex.LParen, lex.Dot, lex.Hex, lex.LBracket:
		return true
	}
	return false
}

func (p *Parser) bareArgs() ([]ast.Expr, error) {
	var args []ast.Expr
	for {
		if stopsBareArg(p) {
			return args, nil
		}
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		args = append(args, e)
		if p.tok.Kind == lex.Comma {
			p.next()
			p.skipNL()
			continue
		}
		return args, nil
	}
}

func (p *Parser) argList(closeK lex.Kind) ([]ast.Expr, error) {
	var args []ast.Expr
	p.skipNL()
	if p.tok.Kind == closeK {
		p.next()
		return args, nil
	}
	for {
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		args = append(args, e)
		p.skipNL()
		if p.tok.Kind == lex.Comma {
			p.next()
			p.skipNL()
			continue
		}
		if p.tok.Kind != closeK {
			return nil, p.errorf("expected %s", closeK)
		}
		p.next()
		return args, nil
	}
}

func (p *Parser) expr() (ast.Expr, error) { return p.or() }

func (p *Parser) or() (ast.Expr, error) {
	e, err := p.and()
	if err != nil {
		return nil, err
	}
	for p.is("or") || p.is("xor") {
		op := identKey(p.tok)
		pos := astPos(p.tok)
		p.next()
		r, err := p.and()
		if err != nil {
			return nil, err
		}
		e = &ast.BinaryExpr{Src: pos, Op: op, Left: e, Right: r}
	}
	return e, nil
}

func (p *Parser) and() (ast.Expr, error) {
	e, err := p.cmp()
	if err != nil {
		return nil, err
	}
	for p.is("and") {
		pos := astPos(p.tok)
		p.next()
		r, err := p.cmp()
		if err != nil {
			return nil, err
		}
		e = &ast.BinaryExpr{Src: pos, Op: "and", Left: e, Right: r}
	}
	return e, nil
}

func (p *Parser) cmp() (ast.Expr, error) {
	e, err := p.add()
	if err != nil {
		return nil, err
	}
	for {
		var op string
		switch p.tok.Kind {
		case lex.Eq:
			op = "="
		case lex.Neq:
			op = "<>"
		case lex.Lt:
			op = "<"
		case lex.Gt:
			op = ">"
		case lex.Le:
			op = "<="
		case lex.Ge:
			op = ">="
		default:
			return e, nil
		}
		pos := astPos(p.tok)
		p.next()
		r, err := p.add()
		if err != nil {
			return nil, err
		}
		e = &ast.BinaryExpr{Src: pos, Op: op, Left: e, Right: r}
	}
}

func (p *Parser) add() (ast.Expr, error) {
	e, err := p.mul()
	if err != nil {
		return nil, err
	}
	for p.tok.Kind == lex.Plus || p.tok.Kind == lex.Minus {
		op := p.tok.Lit
		pos := astPos(p.tok)
		p.next()
		r, err := p.mul()
		if err != nil {
			return nil, err
		}
		e = &ast.BinaryExpr{Src: pos, Op: op, Left: e, Right: r}
	}
	return e, nil
}

func (p *Parser) mul() (ast.Expr, error) {
	e, err := p.pow()
	if err != nil {
		return nil, err
	}
	for p.tok.Kind == lex.Star || p.tok.Kind == lex.Slash || p.is("mod") {
		op := p.tok.Lit
		if p.is("mod") {
			op = "mod"
		}
		pos := astPos(p.tok)
		p.next()
		r, err := p.pow()
		if err != nil {
			return nil, err
		}
		e = &ast.BinaryExpr{Src: pos, Op: op, Left: e, Right: r}
	}
	return e, nil
}

func (p *Parser) pow() (ast.Expr, error) {
	e, err := p.unary()
	if err != nil {
		return nil, err
	}
	if p.tok.Kind == lex.Caret {
		pos := astPos(p.tok)
		p.next()
		r, err := p.pow()
		if err != nil {
			return nil, err
		}
		e = &ast.BinaryExpr{Src: pos, Op: "^", Left: e, Right: r}
	}
	return e, nil
}

func (p *Parser) unary() (ast.Expr, error) {
	if p.is("not") || p.tok.Kind == lex.Minus || p.tok.Kind == lex.Plus {
		op := p.tok.Lit
		if p.is("not") {
			op = "not"
		}
		pos := astPos(p.tok)
		p.next()
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Src: pos, Op: op, X: x}, nil
	}
	return p.primary()
}

func (p *Parser) primary() (ast.Expr, error) {
	pos := astPos(p.tok)
	switch p.tok.Kind {
	case lex.Number:
		n, err := strconv.ParseFloat(p.tok.Lit, 64)
		if err != nil {
			return nil, p.errorf("bad number %q", p.tok.Lit)
		}
		p.next()
		return &ast.NumberExpr{Src: pos, Value: n}, nil
	case lex.Hex:
		n, err := strconv.ParseInt(p.tok.Lit, 16, 64)
		if err != nil {
			return nil, p.errorf("bad hex $%s", p.tok.Lit)
		}
		p.next()
		return &ast.NumberExpr{Src: pos, Value: float64(n)}, nil
	case lex.LBracket:
		p.next()
		elems, err := p.argList(lex.RBracket)
		if err != nil {
			return nil, err
		}
		return p.postfix(&ast.VecExpr{Src: pos, Elems: elems})
	case lex.String:
		s := p.tok.Lit
		p.next()
		return &ast.StringExpr{Src: pos, Value: s}, nil
	case lex.LParen:
		p.next()
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if p.tok.Kind != lex.RParen {
			return nil, p.errorf("expected )")
		}
		p.next()
		return e, nil
	case lex.Dot:
		p.next()
		if p.tok.Kind != lex.Ident {
			return nil, p.errorf("expected field name after .")
		}
		field := p.tok.Lit
		p.next()
		e := ast.Expr(&ast.FieldExpr{Src: pos, X: &ast.IdentExpr{Src: pos, Name: "Self"}, Field: field})
		return p.postfix(e)
	case lex.Ident:
		if identKey(p.tok) == "new" {
			p.next()
			if p.tok.Kind != lex.Ident {
				return nil, p.errorf("New expects a type name")
			}
			typ := p.tok.Lit
			p.next()
			var args []ast.Expr
			if p.tok.Kind == lex.LParen {
				p.next()
				var err error
				args, err = p.argList(lex.RParen)
				if err != nil {
					return nil, err
				}
			}
			return p.postfix(&ast.NewExpr{Src: pos, Type: typ, Args: args})
		}
		name := p.tok.Lit
		p.next()
		if p.tok.Kind == lex.LParen || p.tok.Kind == lex.LBracket {
			closeK := lex.RParen
			if p.tok.Kind == lex.LBracket {
				closeK = lex.RBracket
			}
			p.next()
			args, err := p.argList(closeK)
			if err != nil {
				return nil, err
			}
			// Could be array index or function call — interpreter decides.
			if closeK == lex.RBracket {
				return p.postfix(&ast.IndexExpr{Src: pos, Name: name, Index: args})
			}
			return p.postfix(&ast.CallExpr{Src: pos, Name: name, Args: args})
		}
		return p.postfix(&ast.IdentExpr{Src: pos, Name: name})
	default:
		return nil, p.errorf("expected expression, got %s %q", p.tok.Kind, p.tok.Lit)
	}
}

func (p *Parser) postfix(e ast.Expr) (ast.Expr, error) {
	for p.tok.Kind == lex.Dot {
		pos := astPos(p.tok)
		p.next()
		if p.tok.Kind != lex.Ident {
			return nil, p.errorf("expected name after .")
		}
		name := p.tok.Lit
		p.next()
		if p.tok.Kind == lex.LParen {
			p.next()
			args, err := p.argList(lex.RParen)
			if err != nil {
				return nil, err
			}
			e = &ast.MethodExpr{Src: pos, X: e, Name: name, Args: args}
			continue
		}
		e = &ast.FieldExpr{Src: pos, X: e, Field: name}
	}
	return e, nil
}

func astPos(t lex.Token) ast.Src {
	return ast.Src{Line: t.Line, Col: t.Col}
}

func stopsBareArg(p *Parser) bool {
	k := identKey(p.tok)
	switch k {
	case "else", "elseif", "endif", "wend", "next", "until", "case", "default",
		"end", "endselect", "endfunction", "endwhile", "endtype", "endstruct",
		"endmethod", "endnamespace":
		return true
	}
	return false
}

func isKeyword(k string) bool {
	switch k {
	case "if", "then", "else", "elseif", "endif", "while", "wend", "for", "to", "step",
		"next", "repeat", "until", "function", "end", "return", "dim", "redim", "global", "local",
		"const", "enum", "endenum", "select", "case", "default", "data", "read", "restore",
		"and", "or", "not", "xor", "mod", "include", "import", "exit",
		"type", "struct", "method", "namespace", "field", "new", "as":
		return true
	}
	return false
}

func ExpandIncludes(prog *ast.Program, base string) (*ast.Program, error) {
	var out []ast.Stmt
	for _, s := range prog.Stmts {
		path, as, ok := importPath(s)
		if !ok {
			out = append(out, s)
			continue
		}
		file := filepath.FromSlash(strings.ReplaceAll(path, "\\", "/"))
		if !filepath.IsAbs(file) {
			file = filepath.Join(base, file)
		}
		inner, err := ParseFile(file)
		if err != nil {
			return nil, err
		}
		inner, err = ExpandIncludes(inner, filepath.Dir(file))
		if err != nil {
			return nil, err
		}
		stmts := inner.Stmts
		if as != "" {
			prefixStmts(stmts, as)
		}
		out = append(out, stmts...)
	}
	return &ast.Program{Stmts: out}, nil
}

func importPath(s ast.Stmt) (path, as string, ok bool) {
	switch t := s.(type) {
	case *ast.IncludeStmt:
		return t.Path, "", true
	case *ast.ImportStmt:
		return t.Path, t.As, true
	default:
		return "", "", false
	}
}

func prefixStmts(stmts []ast.Stmt, prefix string) {
	for _, s := range stmts {
		switch t := s.(type) {
		case *ast.FuncDecl:
			t.Name = prefix + "." + t.Name
		case *ast.TypeDecl:
			t.Name = prefix + "." + t.Name
		case *ast.MethodDecl:
			t.Recv = prefix + "." + t.Recv
		case *ast.NamespaceDecl:
			t.Name = prefix + "." + t.Name
		}
	}
}

func JoinPos(line, col int) string {
	return fmt.Sprintf("%d:%d", line, col)
}

var _ = strings.ToLower
