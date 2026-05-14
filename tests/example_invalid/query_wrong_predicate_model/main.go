package main

import "github.com/Neratus/golinq"

type User struct {
	ID int
}

type Order struct {
	ID int
}

var orderPredicate = golinq.Predicate[Order](func(o Order) bool { return true })

var q = golinq.Select[User](db, "ID").Where(orderPredicate).ToList()
