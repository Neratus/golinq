package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Neratus/golinq"
	"github.com/Neratus/golinq/internal/ast"
)

func needValueField(node *golinq.ConditionNode) bool {
	switch node.Type {
	case golinq.Cmp, golinq.Field, golinq.Const, golinq.Like:
		return true
	default:
		return false
	}
}

func renderLiteral(v interface{}) (string, error) {
	switch val := v.(type) {
	case string:
		return strconv.Quote(val), nil
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", val), nil
	case bool:
		return fmt.Sprintf("%t", val), nil
	default:
		return "", fmt.Errorf("unsupported literal type: %T", v)
	}
}

func nodeTypeString(t golinq.NodeType) string {
	switch t {
	case golinq.And:
		return "And"
	case golinq.Or:
		return "Or"
	case golinq.Not:
		return "Not"
	case golinq.Cmp:
		return "Cmp"
	case golinq.Field:
		return "Field"
	case golinq.Const:
		return "Const"
	case golinq.Param:
		return "Param"
	case golinq.Like:
		return "Like"
	default:
		return "Unknown"
	}
}

func generateFuncName(qspec *ast.QuerySpec, hashSuffix string) string {
	var name strings.Builder
	name.WriteString("Get" + qspec.StructName)
	for _, step := range qspec.Steps {
		if step.Type == ast.StepWhere && step.PredicateRef != nil {
			name.WriteString("_" + step.PredicateRef.PredicateName)
		}
		if step.Type == ast.StepJoin && step.JoinRef != nil {
			name.WriteString("_" + step.JoinRef.JoinName)
		}
	}
	nameStr := name.String()
	nameStr = strings.ReplaceAll(nameStr, ".", "_")
	return "Query_" + nameStr + "_" + hashSuffix
}
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
