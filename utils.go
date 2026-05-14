package golinq

import "strings"

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
