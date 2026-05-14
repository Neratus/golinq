package postgres_dialect

import "github.com/Neratus/golinq/internal/dialect"

func NewPostgresDialect() *dialect.SQLDialect {
	return &dialect.SQLDialect{
		QuoteLeft:            '"',
		QuotRight:            '"',
		SeparatorDB:          '.',
		SeparatorAlias:       ' ',
		AsKeyword:            "AS",
		PlaceholderStatic:    "",
		PlaceholderTemplate:  "$%d",
		EQ:                   "=",
		NEQ:                  "!=",
		GT:                   ">",
		GE:                   ">=",
		LT:                   "<",
		LE:                   "<=",
		AND:                  "AND",
		OR:                   "OR",
		NOT:                  "NOT",
		IsNull:               "IS NULL",
		IsNotNull:            "IS NOT NULL",
		AllFields:            '*',
		Separator:            ',',
		InnerJoin:            "INNER JOIN",
		LeftJoin:             "LEFT JOIN",
		RightJoin:            "RIGHT JOIN",
		AscSort:              "ASC",
		DescSort:             "DESC",
		Limit:                "LIMIT",
		Offset:               "OFFSET",
		LimitOffsetSeparator: " OFFSET ",
		QueryEnd:             ";",
		LikeOp:               "LIKE",
		LikeAll:              "%",
		LikeOne:              "_",
	}
}
