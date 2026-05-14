package main

import "github.com/Neratus/golinq"

type User struct{}

var pred = golinq.Predicate[User](func(u string) bool { return true })
