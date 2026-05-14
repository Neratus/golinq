package joins

import (
	"golinq/tests/integration/models"

	"github.com/Neratus/golinq"
)

var CountryCityJoin = golinq.Join(
	models.Country{},
	models.City{},
	func(c models.Country, ct models.City) bool {
		return c.ID == ct.CountryID
	},
)

var CountryCountryLanguageJoin = golinq.Join(
	models.Country{},
	models.CountryLanguage{},
	func(c models.Country, cl models.CountryLanguage) bool {
		return c.ID == cl.CountryID
	},
)

var CountryLanguageLanguageJoin = golinq.Join(
	models.CountryLanguage{},
	models.Language{},
	func(cl models.CountryLanguage, l models.Language) bool {
		return cl.LanguageID == l.ID
	},
)
