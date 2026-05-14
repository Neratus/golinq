package predicates

import (
	"hi/models"

	"github.com/Neratus/golinq"
)

var Adult = golinq.Predicate[models.User](func(u models.User) bool {
	return u.Age >= 18
})

var ActiveUser = golinq.Predicate[models.User](IsActive)

func IsActive(u models.User) bool {
	return u.Active == true
}
