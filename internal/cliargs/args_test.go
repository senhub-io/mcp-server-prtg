package cliargs

import (
	"testing"

	"github.com/alexflint/go-arg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseSettingsFromEnvironment(t *testing.T) {
	t.Setenv("PRTG_DB_HOST", "db.example")
	t.Setenv("PRTG_DB_PORT", "5433")
	t.Setenv("PRTG_DB_NAME", "prtg_data_exporter_collection")
	t.Setenv("PRTG_DB_USER", "reader")
	t.Setenv("PRTG_DB_PASSWORD", "a b'c")
	t.Setenv("PRTG_DB_SSLMODE", "disable")

	var args ParsedArgs
	p, err := arg.NewParser(arg.Config{}, &args)
	require.NoError(t, err)
	require.NoError(t, p.Parse([]string{"run"}))

	assert.Equal(t, "db.example", args.DBHost)
	assert.Equal(t, 5433, args.DBPort)
	assert.Equal(t, "prtg_data_exporter_collection", args.DBName)
	assert.Equal(t, "reader", args.DBUser)
	assert.Equal(t, "a b'c", args.DBPassword)
	assert.Equal(t, "disable", args.DBSSLMode)
}

func TestDatabaseFlagOverridesEnvironment(t *testing.T) {
	t.Setenv("PRTG_DB_PASSWORD", "from-env")

	var args ParsedArgs
	p, err := arg.NewParser(arg.Config{}, &args)
	require.NoError(t, err)
	require.NoError(t, p.Parse([]string{"run", "--db-password", "from-flag"}))

	assert.Equal(t, "from-flag", args.DBPassword)
}
