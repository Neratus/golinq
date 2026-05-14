package models

// golinq:model
type Country struct {
	ID         string  `golinq:"primary_key,column=id"`
	Name       string  `golinq:"column=name"`
	Population int64   `golinq:"column=population"`
	Area       float64 `golinq:"column=area"`
}

// golinq:model
type City struct {
	ID         string `golinq:"primary_key,column=id"`
	Name       string `golinq:"column=name"`
	Population int64  `golinq:"column=population"`
	CountryID  string `golinq:"column=country_id"`
}

// golinq:model
type Language struct {
	ID   string `golinq:"primary_key,column=id"`
	Name string `golinq:"column=name"`
	Code string `golinq:"column=code"`
}

// golinq:model
type CountryLanguage struct {
	CountryID  string `golinq:"column=country_id"`
	LanguageID string `golinq:"column=language_id"`
	IsOfficial bool   `golinq:"column=is_official"`
}
