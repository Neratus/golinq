package sqlgen

import (
	"fmt"
	"strings"

	"github.com/Neratus/golinq/internal/condition"
	"github.com/Neratus/golinq/internal/dialect"
)

func buildLogical(node *condition.ConditionNode, operator string, d *dialect.SQLDialect, params *[]interface{}, paramValues []interface{}, ph *int) (string, error) {
	if len(node.Children) == 0 {
		return "", nil
	}
	var parts []string
	for _, child := range node.Children {
		part, err := buildCond(child, d, params, paramValues, ph)
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, " "+operator+" ")), nil
}

func placeholder(d *dialect.SQLDialect, ph int) string {
	if d.PlaceholderStatic != "" {
		return d.PlaceholderStatic
	}
	return fmt.Sprintf(d.PlaceholderTemplate, ph)
}

func buildCond(node *condition.ConditionNode, d *dialect.SQLDialect, params *[]any, paramValues []any, ph *int) (string, error) {
	if node == nil {
		return "", nil
	}

	switch node.Type {
	case condition.Field:
		val, ok := node.Value.(string)
		if !ok {
			return "", fmt.Errorf("Field node value is not string: %T", node.Value)
		}
		return val, nil

	case condition.Const:
		return literalToSQL(node.Value, d)

	case condition.Param:
		idx := node.ParamIndex - 1
		if idx < 0 || idx >= len(paramValues) {
			return "", fmt.Errorf("param index %d out of range (len=%d)", node.ParamIndex, len(paramValues))
		}
		*params = append(*params, paramValues[idx])
		p := placeholder(d, *ph)
		*ph++
		return p, nil

	case condition.Cmp:
		if len(node.Children) != 2 {
			return "", fmt.Errorf("Cmp node must have 2 children, got %d", len(node.Children))
		}
		left, err := buildCond(node.Children[0], d, params, paramValues, ph)
		if err != nil {
			return "", err
		}
		right, err := buildCond(node.Children[1], d, params, paramValues, ph)
		if err != nil {
			return "", err
		}
		opStr, ok := node.Value.(string)
		if !ok {
			return "", fmt.Errorf("Cmp node value is not string: %T", node.Value)
		}
		op := getOperator(d, opStr)
		return fmt.Sprintf("(%s %s %s)", left, op, right), nil

	case condition.And:
		if d.AND == "" {
			return "", fmt.Errorf("dialect AND operator is empty")
		}
		return buildLogical(node, d.AND, d, params, paramValues, ph)

	case condition.Or:
		if d.OR == "" {
			return "", fmt.Errorf("dialect OR operator is empty")
		}
		return buildLogical(node, d.OR, d, params, paramValues, ph)

	case condition.Not:
		if len(node.Children) != 1 {
			return "", fmt.Errorf("Not node must have 1 child, got %d", len(node.Children))
		}
		child, err := buildCond(node.Children[0], d, params, paramValues, ph)
		if err != nil {
			return "", err
		}
		if d.NOT == "" {
			return "", fmt.Errorf("dialect NOT operator is empty")
		}
		return fmt.Sprintf("(%s %s)", d.NOT, child), nil

	case condition.Like:
		if len(node.Children) != 2 {
			return "", fmt.Errorf("Like node must have 2 children, got %d", len(node.Children))
		}
		left, err := buildCond(node.Children[0], d, params, paramValues, ph)
		if err != nil {
			return "", err
		}
		right, err := buildCond(node.Children[1], d, params, paramValues, ph)
		if err != nil {
			return "", err
		}
		likeType, _ := node.Value.(string)
		var rightExpr string
		switch likeType {
		case "HasPrefix":
			rightExpr = fmt.Sprintf("%s || '%s'", right, d.LikeAll)
		case "HasSuffix":
			rightExpr = fmt.Sprintf("'%s' || %s", d.LikeAll, right)
		default:
			rightExpr = fmt.Sprintf("'%s' || %s || '%s'", d.LikeAll, right, d.LikeAll)
		}
		if d.LikeOp == "" {
			return "", fmt.Errorf("dialect LikeOp is empty")
		}
		return fmt.Sprintf("%s %s %s", left, d.LikeOp, rightExpr), nil

	case condition.Func:
		return "", fmt.Errorf("Func node is not supported in SQL generation (should have been replaced by Const)")

	default:
		return "", fmt.Errorf("unsupported node type: %v", node.Type)
	}
}

func Generate(ast *condition.SelectQueryAST, dialect *dialect.SQLDialect, paramValues []interface{}) (string, []interface{}, error) {
	if ast == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}
	var params []any
	var ph = 1
	var query strings.Builder

	query.WriteString("SELECT ")
	for i, f := range ast.SelectFields {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(quoteIdentifier(dialect, f.TableAlias))
		query.WriteString(".")
		query.WriteString(quoteIdentifier(dialect, f.ColumnName))
	}

	query.WriteString(" FROM ")
	query.WriteString(quoteIdentifier(dialect, ast.From.Name))
	query.WriteString(" AS ")
	query.WriteString(quoteIdentifier(dialect, ast.From.Alias))

	for _, j := range ast.Joins {
		var joinKeyword string
		switch j.Type {
		case "INNER":
			joinKeyword = dialect.InnerJoin
		case "LEFT":
			joinKeyword = dialect.LeftJoin
		default:
			return "", nil, fmt.Errorf("unsupported join type: %s", j.Type)
		}
		if joinKeyword == "" {
			return "", nil, fmt.Errorf("dialect join keyword for %s is empty", j.Type)
		}
		query.WriteString(" ")
		query.WriteString(joinKeyword)
		query.WriteString(" ")
		query.WriteString(quoteIdentifier(dialect, j.Right.Name))
		query.WriteString(" AS ")
		query.WriteString(quoteIdentifier(dialect, j.Right.Alias))
		query.WriteString(" ON ")

		onSQL, err := buildCond(j.On, dialect, &params, paramValues, &ph)
		if err != nil {
			return "", nil, err
		}
		query.WriteString(onSQL)
	}

	if ast.Where != nil {
		whereSQL, err := buildCond(ast.Where, dialect, &params, paramValues, &ph)
		if err != nil {
			return "", nil, err
		}
		if whereSQL != "" && whereSQL != "()" {
			query.WriteString(" WHERE ")
			query.WriteString(whereSQL)
		}
	}

	if ast.OrderBy != nil {
		if ast.OrderBy.MappingSQL == "" {
			return "", nil, fmt.Errorf("order by field MappingSQL is empty")
		}
		query.WriteString(" ORDER BY ")
		query.WriteString(quoteIdentifier(dialect, ast.OrderBy.TableAlias))
		query.WriteString(".")
		query.WriteString(quoteIdentifier(dialect, ast.OrderBy.MappingSQL))
		if ast.OrderBy.Desc {
			if dialect.DescSort == "" {
				return "", nil, fmt.Errorf("dialect DescSort is empty")
			}
			query.WriteString(" " + dialect.DescSort)
		} else {
			if dialect.AscSort == "" {
				query.WriteString(" ASC")
			} else {
				query.WriteString(" " + dialect.AscSort)
			}
		}
	}

	if ast.Limit < 0 || ast.Offset < 0 {
		return "", nil, fmt.Errorf("limit or offset cannot be negative")
	}
	if ast.Limit > 0 {
		if dialect.Limit == "" {
			return "", nil, fmt.Errorf("dialect Limit keyword is empty")
		}
		fmt.Fprintf(&query, " %s %d", dialect.Limit, ast.Limit)
	}
	if ast.Offset > 0 {
		if dialect.Offset == "" {
			return "", nil, fmt.Errorf("dialect Offset keyword is empty")
		}
		fmt.Fprintf(&query, " %s %d", dialect.Offset, ast.Offset)
	}

	if dialect.QueryEnd != "" {
		query.WriteString(dialect.QueryEnd)
	}

	fmt.Println("Res Query: ", query.String())
	return query.String(), params, nil
}
