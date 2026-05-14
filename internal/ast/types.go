package ast

import "go/ast"

type QueryStepType int

const (
	StepWhere QueryStepType = iota
	StepJoin
)

type StructField struct {
	FieldName  string
	MappingSQL string
	FieldType  string
	IsPrimary  bool
	Skip       bool
}

type ModelMeta struct {
	PackageName string
	PackagePath string
	StructName  string
	TableName   string
	FilePath    string
	Fields      []StructField
}

type PredicateArg struct {
	Name        string
	TypeName    string
	PackageName string
	Declared    bool
}

type OrderByClause struct {
	Field string
	Desc  bool
}

type PredicateMeta struct {
	PredicateName  string
	PackageName    string
	ModelType      string
	PackagePath    string
	GlobalVarName  string
	Args           []*PredicateArg
	FuncBody       *ast.FuncLit
	LinkedFuncName *string
	FuncDeclRef    *ast.FuncDecl
}

type JoinMeta struct {
	JoinName       string
	PackageName    string
	PackagePath    string
	GlobalVarName  string
	LeftArg        PredicateArg
	LeftModelType  string
	RightArg       PredicateArg
	RightModelType string
	OnFunc         *ast.FuncLit
	JoinType       string
	LinkedFuncName *string
	FuncDeclRef    *ast.FuncDecl
}

type FuncDeclMeta struct {
	Name        string
	PackageName string
	PackagePath string
	FuncDecl    *ast.FuncDecl
	IsUsed      bool
}

type QueryStep struct {
	Type          QueryStepType
	Predicate     string
	PredicateRef  *PredicateMeta
	Join          string
	JoinRef       *JoinMeta
	PredicateArgs []ast.Expr
}

type QuerySpec struct {
	PackageName      string
	PackagePath      string
	StructName       string
	SelectCols       []string
	Steps            []QueryStep
	LimitVal         int
	OffsetVal        int
	OrderBy          *OrderByClause
	Method           string
	ModelImportPath  string
	ModelImportAlias string
}

type FileImports struct {
	FilePath    string
	PackageName string
	Imports     map[string]string
}

type ProjectQueries struct {
	Models      map[string]*ModelMeta
	Predicates  map[string]*PredicateMeta
	Joins       map[string]*JoinMeta
	QueryCalls  map[string]*QuerySpec
	FileImports map[string]*FileImports
	FuncDecls   map[string]*FuncDeclMeta
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
