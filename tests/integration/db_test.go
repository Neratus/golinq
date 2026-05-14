package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/Neratus/golinq"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CountrySimple struct {
	Name       string
	Population int64
	Area       float64
}

type CountryLanguageManual struct {
	CountryID    string
	CountryName  string
	LanguageName string
	LanguageCode string
}

func TestIntegration_GolinqVsPGX(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://test:test@localhost:5433/testdb?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("Skipping integration test: cannot connect to DB: %v", err)
	}

	_, err = db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	require.NoError(t, err)
	schemaSQL, err := os.ReadFile("schema.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(schemaSQL))
	require.NoError(t, err)

	gdb, err := golinq.Connect(dsn)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("SelectCountriesWithPopulationGreaterThan100M", func(t *testing.T) {
		expectedNames := []string{"Mexico", "USA"}

		golinqRows, err := Query_Getmodels_Country_CountryPopulationHigh_Select(ctx, gdb)
		require.NoError(t, err)

		pgxRows, err := queryManualCountries(db, ctx)
		require.NoError(t, err)

		assert.Equal(t, len(expectedNames), len(golinqRows))
		assert.Equal(t, len(pgxRows), len(golinqRows))
		for i := range golinqRows {
			assert.Equal(t, pgxRows[i].Name, golinqRows[i].Name)
			assert.Equal(t, pgxRows[i].Population, golinqRows[i].Population)
		}
	})

	t.Run("SelectCountriesWithLanguages", func(t *testing.T) {
		golinqRows, err := Query_Getmodels_Country_CountryCountryLanguageJoin_CountryLanguageLanguageJoin_Select(ctx, gdb)
		require.NoError(t, err)

		pgxRows, err := queryManualCountryLanguages(db, ctx)
		require.NoError(t, err)

		assert.Equal(t, len(pgxRows), len(golinqRows))
		for i := range golinqRows {
			assert.Equal(t, pgxRows[i].CountryID, golinqRows[i].CountryId)
			assert.Equal(t, pgxRows[i].CountryName, golinqRows[i].CountryName)
			assert.Equal(t, pgxRows[i].LanguageName, golinqRows[i].LanguageName)
			assert.Equal(t, pgxRows[i].LanguageCode, golinqRows[i].LanguageCode)
		}
	})
}
func queryManualCountries(db *sql.DB, ctx context.Context) ([]CountrySimple, error) {
	rows, err := db.QueryContext(ctx, `
        SELECT name, population, area
        FROM country
        WHERE population > $1
        ORDER BY population DESC
    `, 100000000)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []CountrySimple
	for rows.Next() {
		var cr CountrySimple
		if err := rows.Scan(&cr.Name, &cr.Population, &cr.Area); err != nil {
			return nil, err
		}
		results = append(results, cr)
	}
	return results, rows.Err()
}

func queryManualCountryLanguages(db *sql.DB, ctx context.Context) ([]CountryLanguageManual, error) {
	rows, err := db.QueryContext(ctx, `
        SELECT c.id, c.name, l.name, l.code
        FROM country c
        JOIN country_language cl ON c.id = cl.country_id
        JOIN language l ON cl.language_id = l.id
        ORDER BY c.name DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []CountryLanguageManual
	for rows.Next() {
		var r CountryLanguageManual
		if err := rows.Scan(&r.CountryID, &r.CountryName, &r.LanguageName, &r.LanguageCode); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
