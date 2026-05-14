package main

import "github.com/Neratus/golinq"

type Order struct{}

var j = golinq.Join(Order{}, MissingModel{}, func(o Order, m MissingModel) bool { return true })
