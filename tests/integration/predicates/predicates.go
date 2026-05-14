package predicates

import (
	"golinq/tests/integration/models"

	"github.com/Neratus/golinq"
)

var CountryPopulationHigh = golinq.Predicate[models.Country](func(c models.Country) bool {
	return c.Population > 100000000
})
