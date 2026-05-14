package queries

import "github.com/Neratus/golinq"

type User struct{}
type Order struct{}
type Product struct{}

var j = golinq.Join(Order{}, Product{}, func(o Order, p Product) bool { return true })

var q = golinq.Select[User](db, "ID").Join(j).ToList()
