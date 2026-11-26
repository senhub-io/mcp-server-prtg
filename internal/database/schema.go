package database

import (
	"context"
	"database/sql"
	"fmt"
)

type schemaFeatures struct {
	softDelete       bool
	treeDepth        bool
	channelValues    bool
	collectionStatus bool
}

func detectSchema(ctx context.Context, conn *sql.DB) (schemaFeatures, error) {
	query := `
		SELECT
			(SELECT COUNT(*) = 3 FROM information_schema.columns
				WHERE table_schema = current_schema() AND column_name = 'is_deleted'
					AND table_name IN ('prtg_sensor', 'prtg_device', 'prtg_group')),
			(SELECT COUNT(*) = 2 FROM information_schema.columns
				WHERE table_schema = current_schema() AND column_name = 'tree_depth'
					AND table_name IN ('prtg_device', 'prtg_group')),
			EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'prtg_channel_value'),
			EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'data_collection_status')
	`

	var f schemaFeatures
	err := conn.QueryRowContext(ctx, query).Scan(
		&f.softDelete,
		&f.treeDepth,
		&f.channelValues,
		&f.collectionStatus,
	)
	if err != nil {
		return schemaFeatures{}, fmt.Errorf("detecting database schema: %w", err)
	}

	return f, nil
}

func (db *DB) notDeleted(alias string) string {
	if db.schema.softDelete {
		return " AND NOT " + alias + ".is_deleted"
	}
	return ""
}

func (db *DB) groupDepthExpr(alias string) string {
	if db.schema.treeDepth {
		return alias + ".tree_depth"
	}
	return `(WITH RECURSIVE depth_chain AS (
			SELECT base.id, base.prtg_server_address_id, base.self_group_id, 0 AS depth
			FROM prtg_group base
			WHERE base.id = ` + alias + `.id
				AND base.prtg_server_address_id = ` + alias + `.prtg_server_address_id
			UNION ALL
			SELECT parent.id, parent.prtg_server_address_id, parent.self_group_id, chain.depth + 1
			FROM prtg_group parent
			INNER JOIN depth_chain chain
				ON parent.id = chain.self_group_id
				AND parent.prtg_server_address_id = chain.prtg_server_address_id
			WHERE chain.self_group_id IS NOT NULL
				AND chain.self_group_id <> chain.id
				AND chain.depth < 64
		)
		SELECT COALESCE(MAX(depth), 0) FROM depth_chain)`
}

func (db *DB) deviceDepthExpr(deviceAlias, groupAlias string) string {
	if db.schema.treeDepth {
		return deviceAlias + ".tree_depth"
	}
	return "(" + db.groupDepthExpr(groupAlias) + " + 1)"
}

func (db *DB) rootGroupCondition(alias string) string {
	return `(` + alias + `.self_group_id IS NULL
		OR ` + alias + `.self_group_id = ` + alias + `.id
		OR NOT EXISTS (
			SELECT 1 FROM prtg_group parent
			WHERE parent.id = ` + alias + `.self_group_id
				AND parent.prtg_server_address_id = ` + alias + `.prtg_server_address_id
		))`
}
