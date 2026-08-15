// Package lex tokenizes BitShin BASIC source (case-insensitive BASIC).
package lex

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Kind int

const (
	EOF Kind = iota
	Newline
	Ident
	Number
	String
	Comma
	Colon
	LParen
	RParen
	LBracket
	RBracket
	Plus
	Minus
	Star
	Slash
	Caret
	Eq
	Neq
	Lt
	Gt
	Le
	Ge
	Dot
	Hex
)

type Token struct {
	Kind Kind
	Lit  string
	Line int
	Col  int
}

func (k Kind) String() string {
	names := map[Kind]string{
		EOF: "EOF", Newline: "newline", Ident: "ident", Number: "number",
		String: "string", Comma: ",", Colon: ":", LParen: "(", RParen: ")",
		LBracket: "[", RBracket: "]", Plus: "+", Minus: "-", Star: "*",
		Slash: "/", Caret: "^", Eq: "=", Neq: "<>", Lt: "<", Gt: ">",
		Le: "<=", Ge: ">=", Dot: ".", Hex: "hex",
	}
	if s, ok := names[k]; ok {
		return s
	}
	return fmt.Sprintf("tok(%d)", k)
}

type Lexer struct {
	src  string
	pos  int
	line int
	col  int
	peek *Token
}

func New(src string) *Lexer {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	return &Lexer{src: src, line: 1, col: 1}
}

func (l *Lexer) Peek() Token {
	if l.peek == nil {
		t := l.next()
		l.peek = &t
	}
	return *l.peek
}

func (l *Lexer) Next() Token {
	if l.peek != nil {
		t := *l.peek
		l.peek = nil
		return t
	}
	return l.next()
}

func (l *Lexer) next() Token {
	for {
		l.skipSpace()
		if l.pos >= len(l.src) {
			return Token{Kind: EOF, Line: l.line, Col: l.col}
		}
		line, col := l.line, l.col
		r, _ := l.rune()
		switch r {
		case '\n':
			l.adv()
			return Token{Kind: Newline, Lit: "\n", Line: line, Col: col}
		case ';', '\'':
			l.skipLine()
			continue
		case '/':
			if l.lookahead() == '/' {
				l.skipLine()
				continue
			}
			l.adv()
			return Token{Kind: Slash, Lit: "/", Line: line, Col: col}
		case ',':
			l.adv()
			return Token{Kind: Comma, Lit: ",", Line: line, Col: col}
		case ':':
			l.adv()
			return Token{Kind: Colon, Lit: ":", Line: line, Col: col}
		case '(':
			l.adv()
			return Token{Kind: LParen, Lit: "(", Line: line, Col: col}
		case ')':
			l.adv()
			return Token{Kind: RParen, Lit: ")", Line: line, Col: col}
		case '[':
			l.adv()
			return Token{Kind: LBracket, Lit: "[", Line: line, Col: col}
		case ']':
			l.adv()
			return Token{Kind: RBracket, Lit: "]", Line: line, Col: col}
		case '+':
			l.adv()
			return Token{Kind: Plus, Lit: "+", Line: line, Col: col}
		case '-':
			l.adv()
			return Token{Kind: Minus, Lit: "-", Line: line, Col: col}
		case '*':
			l.adv()
			return Token{Kind: Star, Lit: "*", Line: line, Col: col}
		case '^':
			l.adv()
			return Token{Kind: Caret, Lit: "^", Line: line, Col: col}
		case '=':
			l.adv()
			if l.current() == '<' {
				l.adv()
				return Token{Kind: Le, Lit: "=<", Line: line, Col: col}
			}
			if l.current() == '>' {
				l.adv()
				return Token{Kind: Ge, Lit: "=>", Line: line, Col: col}
			}
			return Token{Kind: Eq, Lit: "=", Line: line, Col: col}
		case '<':
			l.adv()
			if l.current() == '>' {
				l.adv()
				return Token{Kind: Neq, Lit: "<>", Line: line, Col: col}
			}
			if l.current() == '=' {
				l.adv()
				return Token{Kind: Le, Lit: "<=", Line: line, Col: col}
			}
			return Token{Kind: Lt, Lit: "<", Line: line, Col: col}
		case '>':
			l.adv()
			if l.current() == '=' {
				l.adv()
				return Token{Kind: Ge, Lit: ">=", Line: line, Col: col}
			}
			if l.current() == '<' {
				l.adv()
				return Token{Kind: Neq, Lit: "><", Line: line, Col: col}
			}
			return Token{Kind: Gt, Lit: ">", Line: line, Col: col}
		case '.':
			if unicode.IsDigit(l.lookahead()) {
				return l.number(line, col)
			}
			l.adv()
			return Token{Kind: Dot, Lit: ".", Line: line, Col: col}
		case '$':
			if isHexDigit(l.lookahead()) {
				return l.hex(line, col)
			}
			l.adv()
			return Token{Kind: Ident, Lit: "$", Line: line, Col: col}
		case '"':
			return l.string(line, col)
		default:
			if unicode.IsDigit(r) {
				return l.number(line, col)
			}
			if isIdentStart(r) {
				return l.ident(line, col)
			}
			l.adv()
			return Token{Kind: Ident, Lit: string(r), Line: line, Col: col}
		}
	}
}

func (l *Lexer) string(line, col int) Token {
	l.adv() // "
	var b strings.Builder
	for l.pos < len(l.src) {
		r, _ := l.rune()
		if r == '"' {
			l.adv()
			if l.current() == '"' {
				b.WriteByte('"')
				l.adv()
				continue
			}
			return Token{Kind: String, Lit: b.String(), Line: line, Col: col}
		}
		if r == '\n' {
			break
		}
		b.WriteRune(r)
		l.adv()
	}
	return Token{Kind: String, Lit: b.String(), Line: line, Col: col}
}

func (l *Lexer) number(line, col int) Token {
	start := l.pos
	sawDot := false
	for l.pos < len(l.src) {
		r, n := l.rune()
		if r == '.' {
			if sawDot {
				break
			}
			sawDot = true
			l.pos += n
			l.col += n
			continue
		}
		if !unicode.IsDigit(r) {
			break
		}
		l.pos += n
		l.col += n
	}
	lit := l.src[start:l.pos]
	if lit == "." {
		return Token{Kind: Ident, Lit: ".", Line: line, Col: col}
	}
	return Token{Kind: Number, Lit: lit, Line: line, Col: col}
}

func (l *Lexer) hex(line, col int) Token {
	l.adv() // $
	start := l.pos
	for l.pos < len(l.src) {
		r, n := l.rune()
		if !isHexDigit(r) {
			break
		}
		l.pos += n
		l.col += n
	}
	return Token{Kind: Hex, Lit: l.src[start:l.pos], Line: line, Col: col}
}

func (l *Lexer) ident(line, col int) Token {
	start := l.pos
	for l.pos < len(l.src) {
		r, n := l.rune()
		if !isIdentPart(r) {
			break
		}
		l.pos += n
		l.col += n
	}
	// Blitz type suffixes
	if l.pos < len(l.src) {
		r, n := l.rune()
		if r == '$' || r == '#' || r == '%' {
			l.pos += n
			l.col += n
		}
	}
	return Token{Kind: Ident, Lit: l.src[start:l.pos], Line: line, Col: col}
}

func (l *Lexer) skipSpace() {
	for l.pos < len(l.src) {
		r, n := l.rune()
		if r == '\n' || !unicode.IsSpace(r) {
			return
		}
		l.pos += n
		l.col += n
	}
}

func (l *Lexer) skipLine() {
	for l.pos < len(l.src) {
		r, _ := l.rune()
		if r == '\n' {
			return
		}
		l.adv()
	}
}

func (l *Lexer) rune() (rune, int) {
	return utf8.DecodeRuneInString(l.src[l.pos:])
}

func (l *Lexer) current() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.pos:])
	return r
}

func (l *Lexer) lookahead() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	_, n := utf8.DecodeRuneInString(l.src[l.pos:])
	if l.pos+n >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.pos+n:])
	return r
}

func (l *Lexer) adv() {
	r, n := l.rune()
	l.pos += n
	if r == '\n' {
		l.line++
		l.col = 1
		return
	}
	l.col += n
}

func isHexDigit(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func IdentKey(s string) string {
	s = strings.TrimRight(s, "$%#")
	return strings.ToLower(s)
}
