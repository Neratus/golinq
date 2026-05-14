package golinq

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	myast "github.com/Neratus/golinq/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makePredicateInline(name, modelType string, body string) *myast.PredicateMeta {
	expr, err := parser.ParseExpr(body)
	if err != nil {
		panic(err)
	}
	funcLit, ok := expr.(*ast.FuncLit)
	if !ok {
		panic("not a func literal")
	}
	args := []*myast.PredicateArg{
		{Name: "u", TypeName: modelType, PackageName: "models", Declared: false},
	}
	return &myast.PredicateMeta{
		PredicateName: name,
		PackageName:   "predicates",
		ModelType:     modelType,
		Args:          args,
		FuncBody:      funcLit,
	}
}

func parseExpr(t *testing.T, src string) ast.Expr {
	expr, err := parser.ParseExpr(src)
	require.NoError(t, err)
	return expr
}

func TestBuildExpr_SimpleCompare(t *testing.T) {
	expr := parseExpr(t, `u.Age >= 18`)
	paramNames := map[string]int{"u": 0}
	paramToAlias := map[string]string{"u": "User"}
	models := make(map[string]*myast.ModelMeta)
	imports := make(map[string]string)

	node, err := buildExpr(expr, paramNames, nil, paramToAlias, models, false, imports)
	require.NoError(t, err)

	assert.Equal(t, Cmp, node.Type)
	assert.Equal(t, "GE", node.Value)
	assert.Len(t, node.Children, 2)
	assert.Equal(t, Field, node.Children[0].Type)
	assert.Contains(t, node.Children[0].Value, "User.")
	assert.Equal(t, Const, node.Children[1].Type)
	assert.Equal(t, 18, node.Children[1].Value)
}

func TestBuildExpr_AndOr(t *testing.T) {
	expr := parseExpr(t, `u.Age > 18 && u.Status == "active" || u.Status == "pending"`)
	paramNames := map[string]int{"u": 0}
	paramToAlias := map[string]string{"u": "User"}
	models := make(map[string]*myast.ModelMeta)

	node, err := buildExpr(expr, paramNames, nil, paramToAlias, models, false, nil)
	require.NoError(t, err)

	assert.Equal(t, Or, node.Type)
	assert.Len(t, node.Children, 2)
	assert.Equal(t, And, node.Children[0].Type)
	assert.Equal(t, Cmp, node.Children[1].Type)
}

func TestBuildExpr_WithParameter(t *testing.T) {
	expr := parseExpr(t, `u.Age > minAge`)
	paramNames := map[string]int{"u": 0, "minAge": 1}
	paramToAlias := map[string]string{"u": "User"}
	args := []ast.Expr{parseExpr(t, `18`)}
	models := make(map[string]*myast.ModelMeta)

	node, err := buildExpr(expr, paramNames, args, paramToAlias, models, false, nil)
	require.NoError(t, err)

	assert.Equal(t, Cmp, node.Type)
	assert.Equal(t, "GT", node.Value)
	right := node.Children[1]
	assert.Equal(t, Const, right.Type)
	assert.Equal(t, 18, right.Value)
}

func TestBuildExpr_Not(t *testing.T) {
	expr := parseExpr(t, `!(u.Age >= 18)`)
	paramNames := map[string]int{"u": 0}
	paramToAlias := map[string]string{"u": "User"}
	models := make(map[string]*myast.ModelMeta)

	node, err := buildExpr(expr, paramNames, nil, paramToAlias, models, false, nil)
	require.NoError(t, err)

	assert.Equal(t, Not, node.Type)
	assert.Len(t, node.Children, 1)
	inner := node.Children[0]
	assert.Equal(t, Cmp, inner.Type)
	assert.Equal(t, "GE", inner.Value)
}

func TestBuildExpr_LikeContains(t *testing.T) {
	expr := parseExpr(t, `strings.Contains(u.Name, "john")`)
	paramNames := map[string]int{"u": 0}
	paramToAlias := map[string]string{"u": "User"}
	models := make(map[string]*myast.ModelMeta)
	imports := map[string]string{"strings": "strings"}

	node, err := buildExpr(expr, paramNames, nil, paramToAlias, models, false, imports)
	require.NoError(t, err)

	assert.Equal(t, Like, node.Type)
	assert.Nil(t, node.Value)
	assert.Len(t, node.Children, 2)
	assert.Equal(t, Field, node.Children[0].Type)
	assert.Equal(t, Const, node.Children[1].Type)
	assert.Equal(t, "john", node.Children[1].Value)
}

func TestBuildExpr_UserFunc(t *testing.T) {
	expr := parseExpr(t, `someFunc(u.Age, 10)`)
	paramNames := map[string]int{"u": 0}
	paramToAlias := map[string]string{"u": "User"}
	models := make(map[string]*myast.ModelMeta)

	node, err := buildExpr(expr, paramNames, nil, paramToAlias, models, false, nil)
	require.NoError(t, err)

	assert.Equal(t, Func, node.Type)
	assert.Equal(t, "someFunc", node.Value)
	assert.Len(t, node.Children, 2)
	assert.Equal(t, Field, node.Children[0].Type)
	assert.Equal(t, Const, node.Children[1].Type)
}

func TestBuildSelectAstTree_NoJoin(t *testing.T) {
	models := map[string]*myast.ModelMeta{
		"User": {
			StructName: "User",
			TableName:  "user",
			Fields: []myast.StructField{
				{FieldName: "ID", MappingSQL: "user_id", FieldType: "int"},
				{FieldName: "Name", MappingSQL: "full_name", FieldType: "string"},
				{FieldName: "Age", MappingSQL: "age", FieldType: "int"},
			},
		},
	}
	predMeta := &myast.PredicateMeta{
		PredicateName: "Adult",
		PackageName:   "predicates",
		ModelType:     "User",
		Args: []*myast.PredicateArg{
			{Name: "u", TypeName: "User", PackageName: "", Declared: false},
		},
		FuncBody: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{
					List: []*ast.Field{
						{Names: []*ast.Ident{{Name: "u"}}, Type: &ast.Ident{Name: "User"}},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.BinaryExpr{
								X:  &ast.SelectorExpr{X: &ast.Ident{Name: "u"}, Sel: &ast.Ident{Name: "Age"}},
								Op: token.GEQ,
								Y:  &ast.BasicLit{Kind: token.INT, Value: "18"},
							},
						},
					},
				},
			},
		},
	}
	qspec := &myast.QuerySpec{
		PackageName: "queries",
		PackagePath: "/fake/queries.go",
		StructName:  "User",
		SelectCols: []myast.SelectFieldSpec{
			{ColumnName: "ID"},
			{ColumnName: "Name"},
		},
		Steps: []myast.QueryStep{
			{
				Type:          myast.StepWhere,
				Predicate:     "Adult",
				PredicateRef:  predMeta,
				PredicateArgs: nil,
			},
		},
		Method: "ToList",
	}
	fileImports := map[string]*myast.FileImports{}
	ast, err := BuildSelectAstTree(qspec, models, fileImports)
	require.NoError(t, err)
	assert.NotNil(t, ast)
	assert.Len(t, ast.SelectFields, 2)
	assert.Equal(t, "user_id", ast.SelectFields[0].ColumnName)
	assert.Equal(t, "full_name", ast.SelectFields[1].ColumnName)
	assert.Equal(t, "user", ast.From.Name)
	assert.Equal(t, "User", ast.From.Alias)
	assert.NotNil(t, ast.Where)
	assert.Equal(t, ToList, ast.Method)
}
func TestBuildSelectAstTree_WithJoin(t *testing.T) {
	models := map[string]*myast.ModelMeta{
		"User": {
			StructName: "User",
			TableName:  "user",
			Fields: []myast.StructField{
				{FieldName: "ID", MappingSQL: "user_id", FieldType: "int"},
				{FieldName: "Name", MappingSQL: "full_name", FieldType: "string"},
			},
		},
		"Order": {
			StructName: "Order",
			TableName:  "order",
			Fields: []myast.StructField{
				{FieldName: "ID", MappingSQL: "order_id", FieldType: "int"},
				{FieldName: "UserID", MappingSQL: "user_id", FieldType: "int"},
			},
		},
	}

	joinMeta := &myast.JoinMeta{
		JoinName:       "UserOrderJoin",
		PackageName:    "joins",
		LeftModelType:  "User",
		RightModelType: "Order",
		JoinType:       "INNER",
		LeftArg:        myast.PredicateArg{Name: "u", TypeName: "User"},
		RightArg:       myast.PredicateArg{Name: "o", TypeName: "Order"},
		OnFunc: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{
					List: []*ast.Field{
						{Names: []*ast.Ident{{Name: "u"}}, Type: &ast.Ident{Name: "User"}},
						{Names: []*ast.Ident{{Name: "o"}}, Type: &ast.Ident{Name: "Order"}},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.BinaryExpr{
								X:  &ast.SelectorExpr{X: &ast.Ident{Name: "u"}, Sel: &ast.Ident{Name: "ID"}},
								Op: token.EQL,
								Y:  &ast.SelectorExpr{X: &ast.Ident{Name: "o"}, Sel: &ast.Ident{Name: "UserID"}},
							},
						},
					},
				},
			},
		},
	}

	qspec := &myast.QuerySpec{
		PackageName: "queries",
		PackagePath: "/fake/queries.go",
		StructName:  "User",
		SelectCols: []myast.SelectFieldSpec{
			{ColumnName: "ID"},
		},
		Steps: []myast.QueryStep{
			{
				Type:    myast.StepJoin,
				Join:    "UserOrderJoin",
				JoinRef: joinMeta,
			},
		},
		Method: "ToList",
	}

	fileImports := map[string]*myast.FileImports{}
	astTree, err := BuildSelectAstTree(qspec, models, fileImports)
	require.NoError(t, err)
	assert.Len(t, astTree.Joins, 1)
	assert.Equal(t, "INNER", astTree.Joins[0].Type)
	assert.Equal(t, "order", astTree.Joins[0].Right.Name)
	assert.Equal(t, "Order", astTree.Joins[0].Right.Alias)
	assert.NotNil(t, astTree.Joins[0].On)
	assert.Equal(t, Cmp, astTree.Joins[0].On.Type)
}
