package dialect

type SQLDialect struct {
	QuoteLeft            rune
	QuotRight            rune
	SeparatorDB          rune
	SeparatorAlias       rune
	AsKeyword            string
	PlaceholderStatic    string
	PlaceholderTemplate  string
	EQ                   string
	NEQ                  string
	GT                   string
	GE                   string
	LT                   string
	LE                   string
	AND                  string
	OR                   string
	NOT                  string
	IsNull               string
	IsNotNull            string
	AllFields            rune
	Separator            rune
	InnerJoin            string
	LeftJoin             string
	RightJoin            string
	AscSort              string
	DescSort             string
	Limit                string
	Offset               string
	LimitOffsetSeparator string
	QueryEnd             string
	LikeOp               string
	LikeAll              string
	LikeOne              string
}
