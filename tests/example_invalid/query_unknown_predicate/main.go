package queries

import "github.com/Neratus/golinq"

type User struct{}

var q = golinq.Select[User](db, "ID").Where(NonExistentPredicate).ToList()
