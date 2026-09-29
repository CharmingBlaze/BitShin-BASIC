package lsp

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"bitshinbasic/internal/ast"
	"bitshinbasic/internal/lex"
	"bitshinbasic/internal/parse"
	"bitshinbasic/internal/syntax"
)

type document struct {
	uri     string
	text    string
	version int
}

func parseDiagnostics(text string) []diagnostic {
	_, err := parse.Parse(text)
	if err == nil {
		return []diagnostic{}
	}
	line, col, msg := splitParseErr(err.Error())
	if line < 1 {
		line = 1
	}
	if col < 1 {
		col = 1
	}
	start := position{Line: line - 1, Character: col - 1}
	end := position{Line: line - 1, Character: col}
	lines := strings.Split(text, "\n")
	if start.Line >= 0 && start.Line < len(lines) {
		ln := lines[start.Line]
		if start.Character < len(ln) {
			end.Character = start.Character + identLen(ln[start.Character:])
			if end.Character <= start.Character {
				end.Character = start.Character + 1
			}
		}
	}
	return []diagnostic{{
		Range:    lspRange{Start: start, End: end},
		Severity: severityError,
		Source:   "bitshin",
		Message:  msg,
	}}
}

func splitParseErr(s string) (line, col int, msg string) {
	msg = s
	n, _ := fmt.Sscanf(s, "line %d:%d:", &line, &col)
	if n != 2 {
		return 1, 1, s
	}
	prefix := fmt.Sprintf("line %d:%d: ", line, col)
	if strings.HasPrefix(s, prefix) {
		msg = strings.TrimPrefix(s, prefix)
	}
	return line, col, msg
}

func identLen(s string) int {
	n := 0
	for _, r := range s {
		if n == 0 {
			if r != '_' && !isLetter(r) {
				return 1
			}
			n++
			continue
		}
		if r == '_' || isLetter(r) || (r >= '0' && r <= '9') || r == '$' || r == '#' || r == '%' {
			n++
			continue
		}
		break
	}
	if n == 0 {
		return 1
	}
	return n
}

func isLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func tokenAt(text string, pos position) (lex.Token, bool) {
	lx := lex.New(text)
	wantLine := pos.Line + 1
	wantCol := pos.Character + 1
	for {
		t := lx.Next()
		if t.Kind == lex.EOF {
			return t, false
		}
		if t.Line != wantLine {
			continue
		}
		end := t.Col + utf8.RuneCountInString(t.Lit)
		if t.Kind == lex.Ident || t.Kind == lex.Number || t.Kind == lex.Hex {
			end = t.Col + len(t.Lit)
		}
		if wantCol >= t.Col && wantCol <= end {
			return t, true
		}
	}
}

func hoverFor(c *catalog, doc *document, pos position) *hoverResult {
	t, ok := tokenAt(doc.text, pos)
	if !ok || t.Kind != lex.Ident {
		return nil
	}
	key := lex.IdentKey(t.Lit)
	prog, err := parse.Parse(doc.text)
	if err == nil {
		if fn, ok := findFunc(prog.Stmts, key); ok {
			return &hoverResult{Contents: markup{
				Kind:  "markdown",
				Value: "**Function** `" + fn.Name + "(" + strings.Join(fn.Params, ", ") + ")`",
			}}
		}
	}
	if text, ok := c.hover[key]; ok {
		return &hoverResult{Contents: markup{
			Kind:  "markdown",
			Value: "**" + t.Lit + "**\n\n" + text,
		}}
	}
	return &hoverResult{Contents: markup{
		Kind:  "markdown",
		Value: "`" + t.Lit + "`",
	}}
}

func completions(c *catalog, doc *document, pos position) completionList {
	items := []completionItem{}
	seen := map[string]bool{}
	add := func(label string, kind int, detail string) {
		key := strings.ToLower(label)
		if seen[key] {
			return
		}
		seen[key] = true
		docu := c.hover[lex.IdentKey(label)]
		items = append(items, completionItem{Label: label, Kind: kind, Detail: detail, Documentation: docu})
	}
	t, tokOK := tokenAt(doc.text, pos)
	dot := tokOK && t.Kind == lex.Dot
	if !dot {
		line := lineAt(doc.text, pos.Line)
		if pos.Character > 0 && pos.Character <= len(line) && line[pos.Character-1] == '.' {
			dot = true
		}
	}
	if dot {
		for k, cmd := range syntax.EntityMethods {
			add(k, kindMethod, cmd)
		}
		return completionList{Items: items}
	}
	for _, k := range keywordList() {
		add(k, kindKeyword, "keyword")
	}
	for _, k := range builtinList() {
		add(k, kindFunction, "builtin")
	}
	for _, n := range c.names {
		kind := kindFunction
		key := lex.IdentKey(n)
		if strings.HasPrefix(key, "key_") || strings.HasPrefix(key, "weather_") || strings.HasPrefix(key, "net_") {
			kind = kindConstant
		}
		add(n, kind, "command")
	}
	prog, err := parse.Parse(doc.text)
	if err == nil {
		syms := []docSymbol{}
		collectSymbols(prog.Stmts, &syms)
		for _, s := range syms {
			add(s.Name, kindFunction, "function")
		}
	}
	return completionList{Items: items}
}

func definitionAt(doc *document, pos position) []location {
	t, ok := tokenAt(doc.text, pos)
	if !ok || t.Kind != lex.Ident {
		return nil
	}
	prog, err := parse.Parse(doc.text)
	if err != nil {
		return nil
	}
	fn, ok := findFunc(prog.Stmts, lex.IdentKey(t.Lit))
	if !ok {
		return nil
	}
	r := rangeFromSrc(fn.Src, fn.Name)
	return []location{{URI: doc.uri, Range: r}}
}

func definitionAcross(docs []*document, doc *document, pos position) []location {
	if loc := definitionAt(doc, pos); len(loc) > 0 {
		return loc
	}
	t, ok := tokenAt(doc.text, pos)
	if !ok || t.Kind != lex.Ident {
		return nil
	}
	key := lex.IdentKey(t.Lit)
	for _, other := range docs {
		if other == nil || other.uri == doc.uri {
			continue
		}
		prog, err := parse.Parse(other.text)
		if err != nil {
			continue
		}
		fn, ok := findFunc(prog.Stmts, key)
		if !ok {
			continue
		}
		r := rangeFromSrc(fn.Src, fn.Name)
		return []location{{URI: other.uri, Range: r}}
	}
	return nil
}

func semanticDiags(cat *catalog, doc *document) []diagnostic {
	prog, err := parse.Parse(doc.text)
	if err != nil {
		return nil
	}
	funcs := map[string]int{}
	var out []diagnostic
	var walk func([]ast.Stmt)
	walk = func(stmts []ast.Stmt) {
		for _, s := range stmts {
			switch t := s.(type) {
			case *ast.FuncDecl:
				k := lex.IdentKey(t.Name)
				if prev, ok := funcs[k]; ok {
					out = append(out, diagnostic{
						Range:    rangeFromSrc(t.Src, t.Name),
						Severity: severityWarning,
						Source:   "bitshin",
						Message:  fmt.Sprintf("function %s is already declared (line %d)", t.Name, prev),
					})
				} else {
					funcs[k] = t.Src.Line
				}
				walk(t.Body)
			case *ast.IfStmt:
				walk(t.Then)
				for _, e := range t.ElseIf {
					walk(e.Body)
				}
				walk(t.Else)
			case *ast.WhileStmt:
				walk(t.Body)
			case *ast.ForStmt:
				walk(t.Body)
			case *ast.RepeatStmt:
				walk(t.Body)
			case *ast.SelectStmt:
				for _, c := range t.Cases {
					walk(c.Body)
				}
				walk(t.Default)
			case *ast.TryStmt:
				walk(t.Body)
				walk(t.Catch)
			case *ast.NamespaceDecl:
				walk(t.Stmts)
			case *ast.TypeDecl:
				for _, m := range t.Methods {
					walk(m.Body)
				}
			}
		}
	}
	walk(prog.Stmts)
	var calls func(ast.Expr)
	calls = func(e ast.Expr) {
		if e == nil {
			return
		}
		switch t := e.(type) {
		case *ast.CallExpr:
			k := lex.IdentKey(t.Name)
			if _, ok := funcs[k]; ok {
				return
			}
			if cat != nil {
				if _, ok := cat.hover[k]; ok {
					return
				}
			}
			if k == "" {
				return
			}
			out = append(out, diagnostic{
				Range:    rangeFromSrc(t.Src, t.Name),
				Severity: severityWarning,
				Source:   "bitshin",
				Message:  "unknown function " + t.Name,
			})
		case *ast.BinaryExpr:
			calls(t.Left)
			calls(t.Right)
		case *ast.UnaryExpr:
			calls(t.X)
		}
	}
	var walkExprStmt func([]ast.Stmt)
	walkExprStmt = func(stmts []ast.Stmt) {
		for _, s := range stmts {
			switch t := s.(type) {
			case *ast.CallStmt:
				calls(&ast.CallExpr{Src: t.Src, Name: t.Name, Args: t.Args})
			case *ast.AssignStmt:
				calls(t.Value)
			case *ast.ExprStmt:
				calls(t.Value)
			case *ast.IfStmt:
				calls(t.Cond)
				walkExprStmt(t.Then)
				walkExprStmt(t.Else)
			case *ast.WhileStmt:
				calls(t.Cond)
				walkExprStmt(t.Body)
			case *ast.ForStmt:
				walkExprStmt(t.Body)
			case *ast.FuncDecl:
				walkExprStmt(t.Body)
			case *ast.TryStmt:
				walkExprStmt(t.Body)
				walkExprStmt(t.Catch)
			}
		}
	}
	walkExprStmt(prog.Stmts)
	return out
}

func renameFunc(docs []*document, doc *document, pos position, newName string) *workspaceEdit {
	t, ok := tokenAt(doc.text, pos)
	if !ok || t.Kind != lex.Ident || newName == "" {
		return nil
	}
	key := lex.IdentKey(t.Lit)
	found := false
	for _, d := range docs {
		if d == nil {
			continue
		}
		prog, err := parse.Parse(d.text)
		if err != nil {
			continue
		}
		if _, ok := findFunc(prog.Stmts, key); ok {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	changes := map[string][]textEdit{}
	for _, d := range docs {
		if d == nil {
			continue
		}
		edits := renameIdents(d.text, key, newName)
		if len(edits) > 0 {
			changes[d.uri] = edits
		}
	}
	if len(changes) == 0 {
		return nil
	}
	return &workspaceEdit{Changes: changes}
}

func renameIdents(text, key, newName string) []textEdit {
	lx := lex.New(text)
	var out []textEdit
	for {
		t := lx.Next()
		if t.Kind == lex.EOF {
			break
		}
		if t.Kind == lex.Ident && lex.IdentKey(t.Lit) == key {
			start := position{Line: t.Line - 1, Character: t.Col - 1}
			out = append(out, textEdit{
				Range:   lspRange{Start: start, End: position{Line: start.Line, Character: start.Character + len(t.Lit)}},
				NewText: newName,
			})
		}
	}
	return out
}

func documentSymbols(doc *document) []docSymbol {
	prog, err := parse.Parse(doc.text)
	if err != nil {
		return []docSymbol{}
	}
	out := []docSymbol{}
	collectSymbols(prog.Stmts, &out)
	return out
}

func collectSymbols(stmts []ast.Stmt, out *[]docSymbol) {
	for _, s := range stmts {
		switch t := s.(type) {
		case *ast.FuncDecl:
			r := rangeFromSrc(t.Src, t.Name)
			*out = append(*out, docSymbol{Name: t.Name, Kind: symFunction, Range: r, SelectionRange: r})
		case *ast.TypeDecl:
			r := rangeFromSrc(t.Src, t.Name)
			kids := []docSymbol{}
			for _, m := range t.Methods {
				mr := rangeFromSrc(m.Src, m.Name)
				kids = append(kids, docSymbol{Name: m.Name, Kind: symMethod, Range: mr, SelectionRange: mr})
			}
			*out = append(*out, docSymbol{Name: t.Name, Kind: symClass, Range: r, SelectionRange: r, Children: kids})
		case *ast.MethodDecl:
			r := rangeFromSrc(t.Src, t.Name)
			*out = append(*out, docSymbol{Name: t.Recv + "." + t.Name, Kind: symMethod, Range: r, SelectionRange: r})
		case *ast.NamespaceDecl:
			r := rangeFromSrc(t.Src, t.Name)
			kids := []docSymbol{}
			collectSymbols(t.Stmts, &kids)
			*out = append(*out, docSymbol{Name: t.Name, Kind: symNamespace, Range: r, SelectionRange: r, Children: kids})
		case *ast.ConstStmt:
			for _, n := range t.Names {
				r := rangeFromSrc(t.Src, n)
				*out = append(*out, docSymbol{Name: n, Kind: symConstant, Range: r, SelectionRange: r})
			}
		case *ast.EnumStmt:
			r := rangeFromSrc(t.Src, t.Name)
			*out = append(*out, docSymbol{Name: t.Name, Kind: symEnum, Range: r, SelectionRange: r})
		case *ast.IfStmt:
			collectSymbols(t.Then, out)
			for _, e := range t.ElseIf {
				collectSymbols(e.Body, out)
			}
			collectSymbols(t.Else, out)
		case *ast.WhileStmt:
			collectSymbols(t.Body, out)
		case *ast.ForStmt:
			collectSymbols(t.Body, out)
		case *ast.RepeatStmt:
			collectSymbols(t.Body, out)
		case *ast.SelectStmt:
			for _, c := range t.Cases {
				collectSymbols(c.Body, out)
			}
			collectSymbols(t.Default, out)
		}
	}
}

func findFunc(stmts []ast.Stmt, key string) (*ast.FuncDecl, bool) {
	walk := append([]ast.Stmt{}, stmts...)
	for i := 0; i < len(walk); i++ {
		switch t := walk[i].(type) {
		case *ast.FuncDecl:
			if lex.IdentKey(t.Name) == key {
				return t, true
			}
			walk = append(walk, t.Body...)
		case *ast.TypeDecl:
			for _, m := range t.Methods {
				if lex.IdentKey(m.Name) == key {
					return m, true
				}
			}
		case *ast.MethodDecl:
			if lex.IdentKey(t.Name) == key {
				return &ast.FuncDecl{Src: t.Src, Name: t.Name, Params: t.Params, Body: t.Body}, true
			}
		case *ast.NamespaceDecl:
			walk = append(walk, t.Stmts...)
		case *ast.IfStmt:
			walk = append(walk, t.Then...)
			walk = append(walk, t.Else...)
			for _, e := range t.ElseIf {
				walk = append(walk, e.Body...)
			}
		case *ast.WhileStmt:
			walk = append(walk, t.Body...)
		case *ast.ForStmt:
			walk = append(walk, t.Body...)
		case *ast.RepeatStmt:
			walk = append(walk, t.Body...)
		case *ast.SelectStmt:
			walk = append(walk, t.Default...)
			for _, c := range t.Cases {
				walk = append(walk, c.Body...)
			}
		}
	}
	return nil, false
}

func rangeFromSrc(src ast.Src, name string) lspRange {
	line := src.Line - 1
	if line < 0 {
		line = 0
	}
	col := src.Col - 1
	if col < 0 {
		col = 0
	}
	start := position{Line: line, Character: col}
	end := position{Line: line, Character: col + len(name)}
	if end.Character < start.Character+1 {
		end.Character = start.Character + 1
	}
	return lspRange{Start: start, End: end}
}

func lineAt(text string, line int) string {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return lines[line]
}
