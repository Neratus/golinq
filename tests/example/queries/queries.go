package queries

import (
	"hi/joins"
	"hi/models"
	"hi/predicates"

	"github.com/Neratus/golinq"
)

var activeUsersQuery = golinq.Select[models.User](db, "ID", "Name", "Age").
	Where(predicates.Adult).
	OrderBy("Age", false).
	Limit(10).
	Offset(5).
	ToList()

var userWithOrders = golinq.Select[models.User](db, "ID", "Name").
	Where(predicates.ActiveUser).
	Join(joins.UserOrderJoin).
	ToList()
