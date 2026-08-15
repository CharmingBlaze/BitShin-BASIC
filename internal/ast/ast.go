// Package ast is the BitShin BASIC syntax tree.
package ast

type Program struct {
	Stmts []Stmt
}

type Node interface {
	Pos() (line, col int)
}

type Stmt interface {
	Node
	stmt()
}

type Expr interface {
	Node
	expr()
}

type Src struct{ Line, Col int }

func (p Src) Pos() (int, int) { return p.Line, p.Col }

type AssignStmt struct {
	Src
	Name   string
	Fields []string // p.x = 1
	Index  []Expr
	Value  Expr
}

type CallStmt struct {
	Src
	Name string
	Args []Expr
}

type IfStmt struct {
	Src
	Cond   Expr
	Then   []Stmt
	ElseIf []ElseIf
	Else   []Stmt
}

type ElseIf struct {
	Cond Expr
	Body []Stmt
}

type WhileStmt struct {
	Src
	Cond Expr
	Body []Stmt
}

type ForStmt struct {
	Src
	Var        string
	Start, End Expr
	Step       Expr
	Body       []Stmt
}

type RepeatStmt struct {
	Src
	Body  []Stmt
	Until Expr
}

type ReturnStmt struct {
	Src
	Value Expr
}

type FuncDecl struct {
	Src
	Name   string
	Params []string
	Body   []Stmt
}

type DimStmt struct {
	Src
	Name  string
	Sizes []Expr
	Redim bool
}

type LocalStmt struct {
	Src
	Names  []string
	Values []Expr // nil entry means 0
}

type GlobalStmt struct {
	Src
	Names  []string
	Values []Expr
}

type ConstStmt struct {
	Src
	Names  []string
	Values []Expr
}

type IncludeStmt struct {
	Src
	Path string
}

type ImportStmt struct {
	Src
	Path string
	As   string
}

type TypeField struct {
	Name string
}

type TypeDecl struct {
	Src
	Name    string
	Fields  []TypeField
	Methods []*FuncDecl
}

type MethodDecl struct {
	Src
	Recv   string
	Name   string
	Params []string
	Body   []Stmt
}

type NamespaceDecl struct {
	Src
	Name  string
	Stmts []Stmt
}

type MethodStmt struct {
	Src
	Object string
	Path   []string
	Method string
	Args   []Expr
	Paren  bool // true only for recv.Name( — not recv.Name
}

type ExprStmt struct {
	Src
	Value Expr
}

type EndStmt struct{ Src }

type ExitStmt struct {
	Src
	Function bool
}

type SelectStmt struct {
	Src
	Value   Expr
	Cases   []CaseClause
	Default []Stmt
}

type CaseClause struct {
	Items []CaseItem
	Body  []Stmt
}

type CaseItem struct {
	Low  Expr
	High Expr // set for Case a To b
}

type DataStmt struct {
	Src
	Values []Expr
}

type ReadStmt struct {
	Src
	Names []string
}

type RestoreStmt struct{ Src }

type EnumStmt struct {
	Src
	Name   string
	Names  []string
	Values []Expr // nil = auto
}

func (AssignStmt) stmt()    {}
func (CallStmt) stmt()      {}
func (IfStmt) stmt()        {}
func (WhileStmt) stmt()     {}
func (ForStmt) stmt()       {}
func (RepeatStmt) stmt()    {}
func (ReturnStmt) stmt()    {}
func (FuncDecl) stmt()      {}
func (DimStmt) stmt()       {}
func (GlobalStmt) stmt()    {}
func (LocalStmt) stmt()     {}
func (ConstStmt) stmt()     {}
func (IncludeStmt) stmt()   {}
func (ImportStmt) stmt()    {}
func (TypeDecl) stmt()      {}
func (MethodDecl) stmt()    {}
func (NamespaceDecl) stmt() {}
func (MethodStmt) stmt()    {}
func (ExprStmt) stmt()      {}
func (EndStmt) stmt()       {}
func (ExitStmt) stmt()      {}
func (SelectStmt) stmt()    {}
func (DataStmt) stmt()      {}
func (ReadStmt) stmt()      {}
func (RestoreStmt) stmt()   {}
func (EnumStmt) stmt()      {}

type BinaryExpr struct {
	Src
	Op    string
	Left  Expr
	Right Expr
}

type UnaryExpr struct {
	Src
	Op string
	X  Expr
}

type CallExpr struct {
	Src
	Name string
	Args []Expr
}

type IdentExpr struct {
	Src
	Name string
}

type FieldExpr struct {
	Src
	X     Expr
	Field string
}

type MethodExpr struct {
	Src
	X    Expr
	Name string
	Args []Expr
}

type NewExpr struct {
	Src
	Type string
	Args []Expr
}

type IndexExpr struct {
	Src
	Name  string
	Index []Expr
}

type NumberExpr struct {
	Src
	Value float64
}

type VecExpr struct {
	Src
	Elems []Expr
}

type StringExpr struct {
	Src
	Value string
}

func (BinaryExpr) expr() {}
func (UnaryExpr) expr()  {}
func (CallExpr) expr()   {}
func (IdentExpr) expr()  {}
func (FieldExpr) expr()  {}
func (MethodExpr) expr() {}
func (NewExpr) expr()    {}
func (IndexExpr) expr()  {}
func (NumberExpr) expr() {}
func (StringExpr) expr() {}
func (VecExpr) expr()    {}
