package database

import (
	"context"
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockDB(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()

	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	logger := zerolog.Nop()
	return &DB{conn: mockDB, logger: &logger}, mock
}

func TestExecuteCustomQuery_ValidSelect(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = '15s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT \* FROM \(.+\) AS custom_query LIMIT \$1`).
		WithArgs(100).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "data"}).
				AddRow(1, []byte(`{"a":1}`)),
		)
	mock.ExpectRollback()

	results, err := db.ExecuteCustomQuery(context.Background(), "SELECT id, data FROM t", 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1), results[0]["id"])
	assert.Equal(t, `{"a":1}`, results[0]["data"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteCustomQuery_LimitClamped(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = '15s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT \* FROM \(.+\) AS custom_query LIMIT \$1`).
		WithArgs(1000).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectRollback()

	_, err := db.ExecuteCustomQuery(context.Background(), "SELECT id FROM t", 5000)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteCustomQuery_NegativeLimitBecomes100(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = '15s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT \* FROM \(.+\) AS custom_query LIMIT \$1`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectRollback()

	_, err := db.ExecuteCustomQuery(context.Background(), "SELECT id FROM t", -5)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteCustomQuery_WithQuery(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = '15s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT \* FROM \(.+\) AS custom_query LIMIT \$1`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(42))
	mock.ExpectRollback()

	_, err := db.ExecuteCustomQuery(context.Background(), "WITH c AS (SELECT 1 AS n) SELECT n FROM c", 0)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteCustomQuery_TrailingSemicolon(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = '15s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT \* FROM \(.+\) AS custom_query LIMIT \$1`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectRollback()

	_, err := db.ExecuteCustomQuery(context.Background(), "SELECT id FROM t;", 0)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteCustomQuery_Rejected(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"not select", "DELETE FROM x"},
		{"multiple statements", "SELECT 1; DROP TABLE x"},
		{"line comment", "SELECT 1 -- x"},
		{"block comment", "SELECT /* x */ 1"},
		{"empty", ""},
		{"restricted table prtg_server_address", "SELECT password FROM prtg_server_address"},
		{"restricted quoted table", `SELECT * FROM public."target_database"`},
		{"pg_read_file", "SELECT pg_read_file('/etc/passwd')"},
		{"dblink_exec", "SELECT dblink_exec('host=x', 'DROP TABLE t')"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := newMockDB(t)

			_, err := db.ExecuteCustomQuery(context.Background(), tt.query, 100)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrCustomQueryRejected)
		})
	}
}

func TestExecuteCustomQuery_AcceptsLiteralContainingBlacklistedSubstring(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = '15s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT \* FROM \(.+\) AS custom_query LIMIT \$1`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectRollback()

	_, err := db.ExecuteCustomQuery(context.Background(), "SELECT created_at FROM t WHERE name LIKE '%update%'", 0)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteCustomQuery_PartialWordIdentifierAccepted(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = '15s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT \* FROM \(.+\) AS custom_query LIMIT \$1`).
		WithArgs(100).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("x"))
	mock.ExpectRollback()

	_, err := db.ExecuteCustomQuery(context.Background(), "SELECT name FROM prtg_sensor WHERE name = 'setting_x'", 0)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExecuteCustomQuery_QueryErrorStillRollsBack(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL statement_timeout = '15s'")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT \* FROM \(.+\) AS custom_query LIMIT \$1`).
		WithArgs(100).
		WillReturnError(errors.New("boom"))
	mock.ExpectRollback()

	_, err := db.ExecuteCustomQuery(context.Background(), "SELECT id FROM t", 0)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrCustomQueryRejected)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLikeContains(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc", "%abc%"},
		{"_", `%\_%`},
		{"50%", `%50\%%`},
		{`a\b`, `%a\\b%`},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, likeContains(tt.input), "input %q", tt.input)
	}
}
