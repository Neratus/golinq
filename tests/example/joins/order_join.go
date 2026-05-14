package joins

import (
	"hi/models"

	"github.com/Neratus/golinq"
)

var UserOrderJoin = golinq.Join(
	models.User{},
	models.Order{},
	func(u models.User, o models.Order) bool {
		return u.ID == o.UserID
	},
)

var AnotherJoin = golinq.Join(models.User{}, models.Order{}, userOrderCondition)

func userOrderCondition(u models.User, o models.Order) bool {
	return u.ID == o.UserID && o.Amount > 100
}
