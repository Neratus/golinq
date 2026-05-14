package queries

import "github.com/Neratus/golinq"

type User struct {
	Name string
}

var q = golinq.Select[User](db, "Name").OrderBy("NonExistentField").ToList()
