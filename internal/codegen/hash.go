package codegen

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/Neratus/golinq"
)

func canonicalString(node *golinq.SelectQueryAST) string {
	var buf bytes.Buffer
	writeCanonical(&buf, node)
	return buf.String()
}

func writeCanonical(w io.Writer, node *golinq.SelectQueryAST) {
	fmt.Fprintf(w, "SelectFields:")
	for _, f := range node.SelectFields {
		fmt.Fprintf(w, "(%s,%s)", f.TableAlias, f.ColumnName)
	}
	fmt.Fprintf(w, ";From:(%s,%s)", node.From.Name, node.From.Alias)
	fmt.Fprintf(w, ";Joins:")
	for _, j := range node.Joins {
		fmt.Fprintf(w, "[%s", j.Type)
		if j.Left == nil {
			fmt.Fprintf(w, "nil")
		} else {
			fmt.Fprintf(w, "(%s,%s)", j.Left.Name, j.Left.Alias)
		}
		fmt.Fprintf(w, "(%s,%s)", j.Right.Name, j.Right.Alias)
		fmt.Fprintf(w, "On:")
		writeConditionCanonical(w, j.On)
		fmt.Fprintf(w, "]")
	}
	fmt.Fprintf(w, ";Where:")
	writeConditionCanonical(w, node.Where)
}

func writeConditionCanonical(w io.Writer, node *golinq.ConditionNode) {
	if node == nil {
		fmt.Fprintf(w, "nil")
		return
	}
	fmt.Fprintf(w, "%d", node.Type)
	if needValueField(node) {
		fmt.Fprintf(w, "%v", node.Value)
	}
	if node.Type == golinq.Param {
		fmt.Fprintf(w, "idx%d", node.ParamIndex)
	}
	fmt.Fprintf(w, "[")
	for _, ch := range node.Children {
		writeConditionCanonical(w, ch)
	}
	fmt.Fprintf(w, "]")
}

func getHash(expr *golinq.SelectQueryAST) string {
	hash := canonicalString(expr)
	if len(hash) >= 6 {
		return hash[:6]
	}
	return hash + strings.Repeat("0", 6-len(hash))
}
