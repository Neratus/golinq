package codegen

import (
	"testing"

	"github.com/Neratus/golinq"
	myast "github.com/Neratus/golinq/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateFuncName(t *testing.T) {
	qspec := &myast.QuerySpec{
		StructName: "User",
		Steps: []myast.QueryStep{
			{Type: myast.StepWhere, PredicateRef: &myast.PredicateMeta{PredicateName: "Adult"}},
			{Type: myast.StepJoin, JoinRef: &myast.JoinMeta{JoinName: "UserOrderJoin"}},
		},
	}
	hash := "abc123"
	name := generateFuncName(qspec, hash)
	assert.Equal(t, "Query_GetUser_Adult_UserOrderJoin_abc123", name)
}

func TestGenerateFuncName_NoSteps(t *testing.T) {
	qspec := &myast.QuerySpec{StructName: "User"}
	hash := "xyz"
	name := generateFuncName(qspec, hash)
	assert.Equal(t, "Query_GetUser_xyz", name)
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"user_id", "UserId"},
		{"full_name", "FullName"},
		{"age", "Age"},
		{"", ""},
		{"a_b_c", "ABC"},
	}
	for _, tt := range tests {
		got := toCamelCase(tt.input)
		assert.Equal(t, tt.want, got)
	}
}

func TestRenderSelectQueryAST_Simple(t *testing.T) {
	ast := &golinq.SelectQueryAST{
		SelectFields: []golinq.SelectField{
			{TableAlias: "users", ColumnName: "id"},
			{TableAlias: "users", ColumnName: "name"},
		},
		From: golinq.Relation{Name: "users", Alias: "users"},
	}

	literal, err := renderSelectQueryAST(ast, 0)
	require.NoError(t, err)

	expected := `&golinq.SelectQueryAST{
	SelectFields: []golinq.SelectField{
		{TableAlias: "users", ColumnName: "id"},
		{TableAlias: "users", ColumnName: "name"},
	},
	From: golinq.Relation{Name: "users", Alias: "users"},
	Joins: []golinq.JoinNode{
	},
	Where: nil,
	OrderBy: nil,
	Limit: 0,
	Offset: 0,
	Method: golinq.ToList,
}`
	assert.Equal(t, expected, literal)
}

func TestGenerateResultStruct_NoJoin(t *testing.T) {
	ast := &golinq.SelectQueryAST{
		SelectFields: []golinq.SelectField{
			{TableAlias: "users", ColumnName: "id", GoType: "int"},
			{TableAlias: "users", ColumnName: "name", GoType: "string"},
		},
		Joins: []golinq.JoinNode{},
	}
	modelFields := []myast.StructField{} // не важно
	name, def := generateResultStruct(ast, modelFields)
	assert.Empty(t, name)
	assert.Empty(t, def)
}

func TestGenerateResultStruct_WithJoin(t *testing.T) {
	ast := &golinq.SelectQueryAST{
		SelectFields: []golinq.SelectField{
			{TableAlias: "users", ColumnName: "id", GoType: "int"},
			{TableAlias: "orders", ColumnName: "amount", GoType: "float64"},
		},
		From:  golinq.Relation{Name: "users", Alias: "users"},
		Joins: []golinq.JoinNode{{Right: golinq.Relation{Name: "orders", Alias: "orders"}}},
	}
	modelFields := []myast.StructField{}
	name, def := generateResultStruct(ast, modelFields)
	assert.Equal(t, "Result_users_orders", name)
	assert.Contains(t, def, "type Result_users_orders struct")
	assert.Contains(t, def, "Id int `golinq:\"column=id\"`")
	assert.Contains(t, def, "Amount float64 `golinq:\"column=amount\"`")
}
