package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDatabaseConnectionStringQuotesValues(t *testing.T) {
	cases := map[string]string{
		"":           `password='' sslmode='disable'`,
		"with space": `password='with space' sslmode='disable'`,
		`it's`:       `password='it\'s' sslmode='disable'`,
		`back\slash`: `password='back\\slash' sslmode='disable'`,
	}

	for password, want := range cases {
		c := &Configuration{data: ConfigData{Database: DatabaseConfig{
			Host: "localhost", Port: 5432, Name: "prtg", User: "reader", SSLMode: "disable",
		}}}
		c.plain.dbPassword = password

		assert.Equal(t, "host='localhost' port=5432 dbname='prtg' user='reader' "+want, c.GetDatabaseConnectionString(), password)
	}
}
