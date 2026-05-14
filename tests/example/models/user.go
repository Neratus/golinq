package models

// golinq:model
type User struct {
	ID       int    `golinq:"primary_key,column=user_id"`
	Name     string `golinq:"column=full_name"`
	Age      int
	Active   bool `golinq:"skip"`
	Password string
}
