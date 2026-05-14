package golinq

import (
	"testing"

	"github.com/Neratus/golinq/internal/dialect"
	postgres_dialect "github.com/Neratus/golinq/internal/dialect/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_SimpleSelect(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "id"},
			{TableAlias: "users", ColumnName: "name"},
		},
		From: Relation{Name: "users", Alias: "users"},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, params, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."id" AS "users.id", "users"."name" AS "users.name" FROM "users" AS "users";`
	assert.Equal(t, expected, sqlStr)
	assert.Empty(t, params)
}

func TestGenerate_WhereWithLiteral(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "age"},
		},
		From: Relation{Name: "users", Alias: "users"},
		Where: &ConditionNode{
			Type:  Cmp,
			Value: "GE",
			Children: []*ConditionNode{
				{Type: Field, Value: "users.age"},
				{Type: Const, Value: 18},
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."age" AS "users.age" FROM "users" AS "users" WHERE ("users"."age" >= 18);`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_WhereWithParameter(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "name"},
		},
		From: Relation{Name: "users", Alias: "users"},
		Where: &ConditionNode{
			Type:  Cmp,
			Value: "EQ",
			Children: []*ConditionNode{
				{Type: Field, Value: "users.name"},
				{Type: Param, ParamIndex: 1},
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	paramValues := []interface{}{"Alice"}
	sqlStr, params, err := Generate(ast, dialect, paramValues)
	require.NoError(t, err)
	expected := `SELECT "users"."name" AS "users.name" FROM "users" AS "users" WHERE ("users"."name" = $1);`
	assert.Equal(t, expected, sqlStr)
	assert.Len(t, params, 1)
	assert.Equal(t, "Alice", params[0])
}

func TestGenerate_WhereAndOr(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "id"},
		},
		From: Relation{Name: "users", Alias: "users"},
		Where: &ConditionNode{
			Type: And,
			Children: []*ConditionNode{
				{
					Type:  Cmp,
					Value: "GT",
					Children: []*ConditionNode{
						{Type: Field, Value: "users.age"},
						{Type: Const, Value: 18},
					},
				},
				{
					Type: Or,
					Children: []*ConditionNode{
						{
							Type:  Cmp,
							Value: "EQ",
							Children: []*ConditionNode{
								{Type: Field, Value: "users.status"},
								{Type: Const, Value: "active"},
							},
						},
						{
							Type:  Cmp,
							Value: "EQ",
							Children: []*ConditionNode{
								{Type: Field, Value: "users.status"},
								{Type: Const, Value: "pending"},
							},
						},
					},
				},
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."id" AS "users.id" FROM "users" AS "users" WHERE (("users"."age" > 18) AND (("users"."status" = 'active') OR ("users"."status" = 'pending')));`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_Join(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "name"},
			{TableAlias: "orders", ColumnName: "amount"},
		},
		From: Relation{Name: "users", Alias: "users"},
		Joins: []JoinNode{
			{
				Type:  "INNER",
				Right: Relation{Name: "orders", Alias: "orders"},
				On: &ConditionNode{
					Type:  Cmp,
					Value: "EQ",
					Children: []*ConditionNode{
						{Type: Field, Value: "users.id"},
						{Type: Field, Value: "orders.user_id"},
					},
				},
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."name" AS "users.name", "orders"."amount" AS "orders.amount" FROM "users" AS "users" INNER JOIN "orders" AS "orders" ON ("users"."id" = "orders"."user_id");`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_LeftJoin(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "name"},
		},
		From: Relation{Name: "users", Alias: "users"},
		Joins: []JoinNode{
			{
				Type:  "LEFT",
				Right: Relation{Name: "orders", Alias: "orders"},
				On: &ConditionNode{
					Type:  Cmp,
					Value: "EQ",
					Children: []*ConditionNode{
						{Type: Field, Value: "users.id"},
						{Type: Field, Value: "orders.user_id"},
					},
				},
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."name" AS "users.name" FROM "users" AS "users" LEFT JOIN "orders" AS "orders" ON ("users"."id" = "orders"."user_id");`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_OrderBy(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "id"},
		},
		From: Relation{Name: "users", Alias: "users"},
		OrderBy: &OrderByClause{
			TableAlias: "users",
			Field:      "age",
			MappingSQL: "age",
			Desc:       true,
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."id" AS "users.id" FROM "users" AS "users" ORDER BY "users"."age" DESC;`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_LimitOffset(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "name"},
		},
		From:   Relation{Name: "users", Alias: "users"},
		Limit:  10,
		Offset: 5,
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."name" AS "users.name" FROM "users" AS "users" LIMIT 10 OFFSET 5;`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_LikeContains(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "name"},
		},
		From: Relation{Name: "users", Alias: "users"},
		Where: &ConditionNode{
			Type: Like,
			Children: []*ConditionNode{
				{Type: Field, Value: "users.name"},
				{Type: Const, Value: "john"},
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."name" AS "users.name" FROM "users" AS "users" WHERE "users"."name" LIKE '%' || 'john' || '%';`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_LikeHasPrefix(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "name"},
		},
		From: Relation{Name: "users", Alias: "users"},
		Where: &ConditionNode{
			Type:  Like,
			Value: "HasPrefix",
			Children: []*ConditionNode{
				{Type: Field, Value: "users.name"},
				{Type: Const, Value: "john"},
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."name" AS "users.name" FROM "users" AS "users" WHERE "users"."name" LIKE 'john' || '%';`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_MultipleParams(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "id"},
		},
		From: Relation{Name: "users", Alias: "users"},
		Where: &ConditionNode{
			Type: And,
			Children: []*ConditionNode{
				{
					Type:  Cmp,
					Value: "GT",
					Children: []*ConditionNode{
						{Type: Field, Value: "users.age"},
						{Type: Param, ParamIndex: 1},
					},
				},
				{
					Type:  Cmp,
					Value: "EQ",
					Children: []*ConditionNode{
						{Type: Field, Value: "users.name"},
						{Type: Param, ParamIndex: 2},
					},
				},
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	paramValues := []interface{}{18, "Alice"}
	sqlStr, params, err := Generate(ast, dialect, paramValues)
	require.NoError(t, err)
	expected := `SELECT "users"."id" AS "users.id" FROM "users" AS "users" WHERE (("users"."age" > $1) AND ("users"."name" = $2));`
	assert.Equal(t, expected, sqlStr)
	assert.Equal(t, []interface{}{18, "Alice"}, params)
}

func TestGenerate_NilWhere(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{
			{TableAlias: "users", ColumnName: "id"},
		},
		From:  Relation{Name: "users", Alias: "users"},
		Where: nil,
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT "users"."id" AS "users.id" FROM "users" AS "users";`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_NegativeLimitOffset(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{{TableAlias: "users", ColumnName: "id"}},
		From:         Relation{Name: "users", Alias: "users"},
		Limit:        -1,
		Offset:       -5,
	}
	dialect := postgres_dialect.NewPostgresDialect()
	_, _, err := Generate(ast, dialect, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit or offset cannot be negative")
}

func TestGenerate_EmptySelectFields(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{},
		From:         Relation{Name: "users", Alias: "users"},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	expected := `SELECT  FROM "users" AS "users";`
	assert.Equal(t, expected, sqlStr)
}

func TestGenerate_QuoteIdentifier(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{{TableAlias: `my"table`, ColumnName: `col"name`}},
		From:         Relation{Name: `my"table`, Alias: `t`},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	sqlStr, _, err := Generate(ast, dialect, nil)
	require.NoError(t, err)
	assert.Contains(t, sqlStr, `""`)
	assert.Contains(t, sqlStr, `;`)
}

func TestGenerate_UnsupportedJoinType(t *testing.T) {
	ast := &SelectQueryAST{
		SelectFields: []SelectField{{TableAlias: "users", ColumnName: "id"}},
		From:         Relation{Name: "users", Alias: "users"},
		Joins: []JoinNode{
			{
				Type:  "CROSS",
				Right: Relation{Name: "orders", Alias: "orders"},
				On:    nil,
			},
		},
	}
	dialect := postgres_dialect.NewPostgresDialect()
	_, _, err := Generate(ast, dialect, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported join type: CROSS")
}

func TestGenerate_MissingDialectOperators(t *testing.T) {
	dialect := &dialect.SQLDialect{
		QuoteLeft:           '"',
		QuotRight:           '"',
		InnerJoin:           "INNER JOIN",
		LeftJoin:            "LEFT JOIN",
		PlaceholderTemplate: "$%d",
	}
	ast := &SelectQueryAST{
		SelectFields: []SelectField{{TableAlias: "users", ColumnName: "id"}},
		From:         Relation{Name: "users", Alias: "users"},
		Where: &ConditionNode{
			Type: And,
			Children: []*ConditionNode{
				{
					Type:  Cmp,
					Value: "EQ",
					Children: []*ConditionNode{
						{Type: Field, Value: "users.id"},
						{Type: Const, Value: 1},
					},
				},
			},
		},
	}
	_, _, err := Generate(ast, dialect, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dialect AND operator is empty")
}
