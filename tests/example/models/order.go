package models

// golinq:model
type Order struct {
	ID     int `golinq:"primary_key"`
	UserID int
	Amount float64
}
