package queries

import (
	"golinq/tests/integration/joins"
	"golinq/tests/integration/models"
	"golinq/tests/integration/predicates"

	"github.com/Neratus/golinq"
)

var db *golinq.DB

var GetHighPopulationCountries = golinq.Select[models.Country](db, "ID", "Name", "Population", "Area").
	Where(predicates.CountryPopulationHigh).
	OrderBy("Population", true).
	ToList()

var GetCountriesWithCities = golinq.Select[models.Country](db, "ID", "Name").
	Join(joins.CountryCityJoin).
	ToList()

var GetCountriesWithLanguages = golinq.Select[models.Country](db, "ID", "Name").
	Join(joins.CountryCountryLanguageJoin).
	Join(joins.CountryLanguageLanguageJoin).
	ToList()
