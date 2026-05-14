package golinq

import (
	"fmt"
	"strings"

	"github.com/Neratus/golinq/internal/dialect"
)

func isUpperCase(s string) bool {
	return s == strings.ToUpper(s)
}

func convertToSnakeCase(str string) string {
	res := ""
	for i, rune := range str {
		symb := string(rune)
		if i == 0 {
			res += strings.ToLower(symb)
		} else if isUpperCase(symb) {
			res += "_" + strings.ToLower(symb)
		} else {
			res += symb
		}
	}
	return res
}

func getOperator(d *dialect.SQLDialect, op string) string {
	switch op {
	case "EQ":
		return d.EQ
	case "NEQ":
		return d.NEQ
	case "GT":
		return d.GT
	case "GE":
		return d.GE
	case "LT":
		return d.LT
	case "LE":
		return d.LE
	default:
		return "="
	}
}

func quoteIdentifier(d *dialect.SQLDialect, name string) string {
	escaped := strings.ReplaceAll(name, string(d.QuoteLeft), string(d.QuoteLeft)+string(d.QuoteLeft))
	escaped = strings.ReplaceAll(escaped, string(d.QuotRight), string(d.QuotRight)+string(d.QuotRight))
	return string(d.QuoteLeft) + escaped + string(d.QuotRight)
}

func literalToSQL(val any, d *dialect.SQLDialect) (string, error) {
	switch v := val.(type) {
	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'", nil
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v), nil
	case bool:
		if v {
			return "TRUE", nil
		}
		return "FALSE", nil
	default:
		return "", fmt.Errorf("unsupported constant type: %T", v)
	}
}
