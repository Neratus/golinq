package main

import "github.com/Neratus/golinq"

var pred = golinq.Predicate[User](func(u User) bool { return true })
