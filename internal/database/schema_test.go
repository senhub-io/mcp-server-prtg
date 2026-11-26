package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockDBWithSchema(t *testing.T, f schemaFeatures) (*DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock := newMockDB(t)
	db.schema = f

	return db, mock
}

func TestDetectSchema(t *testing.T) {
	db, mock := newMockDB(t)

	rows := sqlmock.NewRows([]string{"soft_delete", "tree_depth", "channel_values", "collection_status"}).
		AddRow(true, false, true, false)
	mock.ExpectQuery("EXISTS").WillReturnRows(rows)

	f, err := detectSchema(context.Background(), db.conn)
	require.NoError(t, err)
	assert.True(t, f.softDelete)
	assert.False(t, f.treeDepth)
	assert.True(t, f.channelValues)
	assert.False(t, f.collectionStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotDeleted(t *testing.T) {
	db, _ := newMockDBWithSchema(t, schemaFeatures{softDelete: true})
	assert.Equal(t, " AND NOT s.is_deleted", db.notDeleted("s"))

	dbLegacy, _ := newMockDBWithSchema(t, schemaFeatures{})
	assert.Equal(t, "", dbLegacy.notDeleted("s"))
}

func TestGroupDepthExpr(t *testing.T) {
	db, _ := newMockDBWithSchema(t, schemaFeatures{treeDepth: true})
	assert.Equal(t, "g.tree_depth", db.groupDepthExpr("g"))

	dbComputed, _ := newMockDBWithSchema(t, schemaFeatures{})
	expr := dbComputed.groupDepthExpr("g")
	assert.Contains(t, expr, "WITH RECURSIVE")
	assert.Contains(t, expr, "g.id")
	assert.Contains(t, expr, "depth < 64")
}

func TestDeviceDepthExpr(t *testing.T) {
	db, _ := newMockDBWithSchema(t, schemaFeatures{treeDepth: true})
	assert.Equal(t, "d.tree_depth", db.deviceDepthExpr("d", "g"))

	dbComputed, _ := newMockDBWithSchema(t, schemaFeatures{})
	assert.Contains(t, dbComputed.deviceDepthExpr("d", "g"), "+ 1")
}

func TestRootGroupCondition(t *testing.T) {
	db, _ := newMockDBWithSchema(t, schemaFeatures{})
	cond := db.rootGroupCondition("g")
	assert.Contains(t, cond, "g.self_group_id IS NULL")
	assert.Contains(t, cond, "g.self_group_id = g.id")
	assert.Contains(t, cond, "NOT EXISTS")
}

func TestGetTagsSoftDelete(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{softDelete: true})

	rows := sqlmock.NewRows([]string{"id", "name", "sensor_count"}).
		AddRow(1, "prod", 3)
	mock.ExpectQuery(`(?s)FROM prtg_tag t\s+LEFT JOIN prtg_sensor_tag st ON t\.id = st\.prtg_tag_id\s+LEFT JOIN prtg_sensor s ON st\.prtg_sensor_id = s\.id AND st\.prtg_server_address_id = s\.prtg_server_address_id AND NOT s\.is_deleted.*GROUP BY t\.id, t\.name`).
		WithArgs("%prod%", 10).
		WillReturnRows(rows)

	tags, err := db.GetTags(context.Background(), "prod", 10)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, 1, tags[0].ID)
	assert.Equal(t, "prod", tags[0].Name)
	assert.Equal(t, 3, tags[0].SensorCount)
	assert.Equal(t, 0, tags[0].ServerID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetTagsNoSoftDelete(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{})

	rows := sqlmock.NewRows([]string{"id", "name", "sensor_count"}).
		AddRow(2, "dev", 0)
	mock.ExpectQuery(`(?s)LEFT JOIN prtg_sensor s ON st\.prtg_sensor_id = s\.id AND st\.prtg_server_address_id = s\.prtg_server_address_id\s+WHERE`).
		WithArgs(100).
		WillReturnRows(rows)

	tags, err := db.GetTags(context.Background(), "", 0)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupsSoftDeleteNoTreeDepth(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{softDelete: true})

	rows := sqlmock.NewRows([]string{"id", "prtg_server_address_id", "name", "is_probe_node", "self_group_id", "full_path", "tree_depth"}).
		AddRow(1, 1, "root", false, nil, "/root", 0)
	mock.ExpectQuery(`(?s)WITH RECURSIVE.*FROM prtg_group g.*AND NOT g\.is_deleted`).
		WithArgs(10).
		WillReturnRows(rows)

	groups, err := db.GetGroups(context.Background(), "", nil, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "root", groups[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGroupsTreeDepth(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{treeDepth: true})

	rows := sqlmock.NewRows([]string{"id", "prtg_server_address_id", "name", "is_probe_node", "self_group_id", "full_path", "tree_depth"}).
		AddRow(1, 1, "root", false, nil, "/root", 0)
	mock.ExpectQuery(`(?s)SELECT.*g\.tree_depth.*FROM prtg_group g`).
		WithArgs(10).
		WillReturnRows(rows)

	groups, err := db.GetGroups(context.Background(), "", nil, 10)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStatisticsCollectionStatus(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{collectionStatus: true})

	now := time.Now().UTC().Truncate(time.Second)

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"total_sensors", "total_devices", "total_groups", "total_tags", "total_probes"}).
			AddRow(10, 5, 2, 3, 1),
	)
	mock.ExpectQuery(`SELECT s\.status`).WillReturnRows(
		sqlmock.NewRows([]string{"status", "count"}).AddRow(3, 10),
	)
	mock.ExpectQuery(`SELECT s\.sensor_type`).WillReturnRows(
		sqlmock.NewRows([]string{"sensor_type", "count"}).AddRow("ping", 10),
	)
	mock.ExpectQuery("data_collection_status").WillReturnRows(
		sqlmock.NewRows([]string{"max", "has_errors"}).AddRow(now, true),
	)

	stats, err := db.GetStatistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 10, stats.TotalSensors)
	require.NotNil(t, stats.LastCollectionUTC)
	assert.True(t, now.Equal(*stats.LastCollectionUTC))
	assert.True(t, stats.CollectionErrors)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStatisticsCollectionStatusFailure(t *testing.T) {
	db, mock := newMockDBWithSchema(t, schemaFeatures{collectionStatus: true})

	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"total_sensors", "total_devices", "total_groups", "total_tags", "total_probes"}).
			AddRow(10, 5, 2, 3, 1),
	)
	mock.ExpectQuery(`SELECT s\.status`).WillReturnRows(
		sqlmock.NewRows([]string{"status", "count"}).AddRow(3, 10),
	)
	mock.ExpectQuery(`SELECT s\.sensor_type`).WillReturnRows(
		sqlmock.NewRows([]string{"sensor_type", "count"}).AddRow("ping", 10),
	)
	mock.ExpectQuery("data_collection_status").WillReturnError(errors.New("boom"))

	stats, err := db.GetStatistics(context.Background())
	require.NoError(t, err)
	assert.Nil(t, stats.LastCollectionUTC)
	assert.False(t, stats.CollectionErrors)
	assert.NoError(t, mock.ExpectationsWereMet())
}
