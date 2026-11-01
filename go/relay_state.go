package paperwork

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const relayStateVersion = 1

// RelayInventory summarizes only Daily/Zotero-owned state. It never returns
// stored values, note paths, or personal content.
type RelayInventory struct {
	Version     int `json:"version"`
	KVKeys      int `json:"kv_keys"`
	Tasks       int `json:"tasks"`
	Notes       int `json:"notes"`
	DailyTasks  int `json:"daily_tasks"`
	ZoteroNotes int `json:"zotero_notes"`
}

func tableColumns(ctx context.Context, tx *sql.Tx, table string) (map[string]bool, error) {
	rows, err := tx.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var def any
		if err = rows.Scan(&cid, &name, &typ, &notnull, &def, &pk); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

// InventoryRelayState must run against a quiescent, consistent SQLite source.
// Use a SQLite backup/snapshot; do not point it at a live main file.
func InventoryRelayState(ctx context.Context, source *sql.DB) (RelayInventory, error) {
	tx, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return RelayInventory{}, err
	}
	defer tx.Rollback()
	return inventoryRelayTransaction(ctx, tx)
}

func inventoryRelayTransaction(ctx context.Context, tx *sql.Tx) (RelayInventory, error) {
	cols, err := tableColumns(ctx, tx, "kv")
	if err != nil {
		return RelayInventory{}, err
	}
	if !cols["key"] || !cols["value"] {
		return RelayInventory{}, errors.New("Relay kv schema mismatch")
	}
	for table, required := range map[string][]string{"tasks": {"origin", "task_id", "phase", "base", "path", "error"}, "notes": {"source", "identity", "path", "block_hash"}} {
		c, e := tableColumns(ctx, tx, table)
		if e != nil {
			return RelayInventory{}, e
		}
		for _, col := range required {
			if !c[col] {
				return RelayInventory{}, fmt.Errorf("Relay %s schema missing %s", table, col)
			}
		}
	}
	out := RelayInventory{Version: relayStateVersion}
	queries := []struct {
		q    string
		dest *int
	}{
		{"SELECT count(*) FROM kv WHERE key IN ('zotero_user','zotero_version') OR key LIKE 'zotero_removed:%' OR key='job:daily' OR key='job:zotero'", &out.KVKeys},
		{"SELECT count(*) FROM tasks", &out.Tasks},
		{"SELECT count(*) FROM notes", &out.Notes},
		{"SELECT count(*) FROM tasks WHERE path LIKE 'Daily/%'", &out.DailyTasks},
		{"SELECT count(*) FROM notes WHERE source='ZOTERO'", &out.ZoteroNotes},
	}
	for _, q := range queries {
		if err = tx.QueryRowContext(ctx, q.q).Scan(q.dest); err != nil {
			return RelayInventory{}, err
		}
	}
	return out, nil
}

// MigrateRelayState copies Relay's original rows into namespaced tables in
// Paperwork. The transaction and version marker make retry safe. It refuses
// unversioned destination rows to avoid silently mixing identities.
func (s *Store) MigrateRelayState(ctx context.Context, source *sql.DB) (RelayInventory, error) {
	if source == nil || s == nil || s.DB == nil {
		return RelayInventory{}, errors.New("source and destination required")
	}
	if source == s.DB {
		return RelayInventory{}, errors.New("source and destination must be separate databases")
	}
	sourceTx, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return RelayInventory{}, err
	}
	defer sourceTx.Rollback()
	inv, err := inventoryRelayTransaction(ctx, sourceTx)
	if err != nil {
		return inv, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return inv, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS relay_state_meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS relay_kv(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS relay_tasks(origin TEXT PRIMARY KEY,task_id INTEGER,phase TEXT NOT NULL,base TEXT,path TEXT,error TEXT);
CREATE TABLE IF NOT EXISTS relay_notes(source TEXT,identity TEXT,path TEXT,block_hash TEXT,PRIMARY KEY(source,identity));`); err != nil {
		return inv, err
	}
	var version int
	err = tx.QueryRowContext(ctx, "SELECT CAST(value AS INTEGER) FROM relay_state_meta WHERE key='migration_version'").Scan(&version)
	if err == nil {
		if version == relayStateVersion {
			return inv, tx.Commit()
		}
		return inv, fmt.Errorf("unsupported Relay state version %d", version)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return inv, err
	}
	var existing int
	if err = tx.QueryRowContext(ctx, "SELECT (SELECT count(*) FROM relay_kv)+(SELECT count(*) FROM relay_tasks)+(SELECT count(*) FROM relay_notes)").Scan(&existing); err != nil {
		return inv, err
	}
	if existing != 0 {
		return inv, errors.New("unmarked Relay destination state exists; refusing merge")
	}
	for _, table := range []struct{ name, cols string }{{"kv", "key,value"}, {"tasks", "origin,task_id,phase,base,path,error"}, {"notes", "source,identity,path,block_hash"}} {
		rows, e := sourceTx.QueryContext(ctx, "SELECT "+table.cols+" FROM "+table.name)
		if e != nil {
			return inv, e
		}
		defer rows.Close()
		for rows.Next() {
			switch table.name {
			case "kv":
				var k, v string
				if e = rows.Scan(&k, &v); e == nil {
					_, e = tx.ExecContext(ctx, "INSERT INTO relay_kv VALUES(?,?)", k, v)
				}
			case "tasks":
				var origin, phase string
				var id sql.NullInt64
				var base, path, msg sql.NullString
				if e = rows.Scan(&origin, &id, &phase, &base, &path, &msg); e == nil {
					_, e = tx.ExecContext(ctx, "INSERT INTO relay_tasks VALUES(?,?,?,?,?,?)", origin, id, phase, base, path, msg)
				}
			case "notes":
				var sourceName, identity, path, hash string
				if e = rows.Scan(&sourceName, &identity, &path, &hash); e == nil {
					_, e = tx.ExecContext(ctx, "INSERT INTO relay_notes VALUES(?,?,?,?)", sourceName, identity, path, hash)
				}
			}
			if e != nil {
				rows.Close()
				return inv, e
			}
		}
		if e = rows.Err(); e != nil {
			rows.Close()
			return inv, e
		}
		rows.Close()
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO relay_state_meta VALUES('migration_version',?)", fmt.Sprint(relayStateVersion)); err != nil {
		return inv, err
	}
	return inv, tx.Commit()
}
