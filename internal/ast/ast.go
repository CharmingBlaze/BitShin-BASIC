// Package ast is the moonBASIC syntax tree.
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
	Names  []string // len > 1: a, b = expr
	Fields []string // p.x = 1, or pads(i).x = 1 together with Index
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
	Tag  string // "do" when this is a Do loop
}

type ForStmt struct {
	Src
	Var        string
	Start, End Expr
	Step       Expr
	In         Expr   // For v In xs  (Each is optional)
	EachType   string // For v = Each(Type)
	Body       []Stmt
}

type RepeatStmt struct {
	Src
	Body  []Stmt
	Until Expr
	Tag   string // "do" when this is Do ... Loop Until
}

type ReturnStmt struct {
	Src
	Value Expr
}

type FuncDecl struct {
	Src
	Name       string
	Params     []string
	Defaults   []Expr // same length as Params; nil means required
	ParamTypes []string
	Ret        string
	Body       []Stmt
	Coroutine  bool
}

type DimStmt struct {
	Src
	Name     string
	Sizes    []Expr
	TypeName string // Dim pads(8) As Pad
	Redim    bool
}

type LocalStmt struct {
	Src
	Names  []string
	Values []Expr // nil entry means 0
	Static bool
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
	Recv     string
	Name     string
	Params   []string
	Defaults []Expr
	Body     []Stmt
}

type NamespaceDecl struct {
	Src
	Name  string
	Stmts []Stmt
}

type MethodStmt struct {
	Src
	Object string
	Index  []Expr // foe(i).Hide — empty for foe.Hide
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

type EraseStmt struct {
	Src
	Name string
	All  bool
}

type DeleteStmt struct {
	Src
	Name string
}

type YieldStmt struct{ Src }

type ExitStmt struct {
	Src
	Function bool
	Loop     string // for, while, repeat, do; empty exits the nearest loop
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
	Quals [][]string // parallel to Names; coin.x stores fields on coin
}

type RestoreStmt struct{ Src }

type EnumStmt struct {
	Src
	Name   string
	Names  []string
	Values []Expr // nil = auto
}

type StrictStmt struct{ Src }

type TryStmt struct {
	Src
	Body   []Stmt
	ErrVar string
	Catch  []Stmt
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
func (EraseStmt) stmt()     {}
func (DeleteStmt) stmt()    {}
func (YieldStmt) stmt()     {}
func (ExitStmt) stmt()      {}
func (SelectStmt) stmt()    {}
func (DataStmt) stmt()      {}
func (ReadStmt) stmt()      {}
func (RestoreStmt) stmt()   {}
func (EnumStmt) stmt()      {}
func (StrictStmt) stmt()    {}
func (TryStmt) stmt()       {}

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

// FmtExpr formats one value. Spec is a printf body such as ".2f".
type FmtExpr struct {
	Src
	X    Expr
	Spec string
}

type CallExpr struct {
	Src
	Name string
	Args []Expr
}

// FuncRefExpr is @Name.
type FuncRefExpr struct {
	Src
	Name string
}

// FuncExpr is an anonymous Function() … End Function.
type FuncExpr struct {
	Src
	Decl *FuncDecl
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

func (BinaryExpr) expr()  {}
func (UnaryExpr) expr()   {}
func (FmtExpr) expr()     {}
func (CallExpr) expr()    {}
func (FuncRefExpr) expr() {}
func (FuncExpr) expr()    {}
func (IdentExpr) expr()   {}
func (FieldExpr) expr()   {}
func (MethodExpr) expr()  {}
func (NewExpr) expr()     {}
func (IndexExpr) expr()   {}
func (NumberExpr) expr()  {}
func (StringExpr) expr()  {}
func (VecExpr) expr()     {}
