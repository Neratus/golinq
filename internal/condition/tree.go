package condition

type NodeType int

const (
	And NodeType = iota
	Or
	Not
	Cmp
	Field
	Const
	Func
	Param
	Like
)

type ConditionNode struct {
	Type       NodeType
	Value      any
	Children   []*ConditionNode
	ParamIndex int
}

type SelectField struct {
	TableAlias string
	ColumnName string
	GoType     string
}

type Relation struct {
	Name  string
	Alias string
}

type JoinNode struct {
	Type  string
	Left  *Relation
	Right Relation
	On    *ConditionNode
}

type QueryMethod int

const (
	ToList QueryMethod = iota
	First
)

type OrderByClause struct {
	TableAlias string
	Field      string
	MappingSQL string
	Desc       bool
}

type SelectQueryAST struct {
	SelectFields []SelectField
	From         Relation
	Joins        []JoinNode
	Where        *ConditionNode
	OrderBy      *OrderByClause
	Limit        int64
	Offset       int64
	Method       QueryMethod
}
