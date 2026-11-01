package paperwork

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRelayStateMigrationIsAtomicAndIdempotent(t *testing.T) {
	ctx := context.Background()
	source, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "relay.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	_, err = source.Exec(`CREATE TABLE kv(key TEXT PRIMARY KEY,value TEXT NOT NULL);CREATE TABLE tasks(origin TEXT PRIMARY KEY,task_id INTEGER,phase TEXT NOT NULL,base TEXT,path TEXT,error TEXT);CREATE TABLE notes(source TEXT,identity TEXT,path TEXT,block_hash TEXT,PRIMARY KEY(source,identity));
INSERT INTO kv VALUES('zotero_user','123'),('zotero_version','42'),('job:daily','{"last_success":"synthetic"}'),('calendar:summary','{"tasks":1}');
INSERT INTO tasks VALUES('origin-daily',9,'synced','{"title":"synthetic"}','Daily/2026-10-31.md',NULL);
INSERT INTO notes VALUES('ZOTERO','ABCD1234','Literature/Example','hash-z'),('READECK','item-1','Read/Example','hash-r');`)
	if err != nil {
		t.Fatal(err)
	}
	dest, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer dest.Close()
	inv, err := dest.MigrateRelayState(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Tasks != 1 || inv.Notes != 2 || inv.DailyTasks != 1 || inv.ZoteroNotes != 1 || inv.KVKeys != 3 {
		t.Fatalf("inventory: %+v", inv)
	}
	if _, err = dest.MigrateRelayState(ctx, source); err != nil {
		t.Fatalf("retry: %v", err)
	}
	for q, want := range map[string]int{"SELECT count(*) FROM relay_kv": 4, "SELECT count(*) FROM relay_tasks": 1, "SELECT count(*) FROM relay_notes": 2} {
		var got int
		if err = dest.DB.QueryRow(q).Scan(&got); err != nil || got != want {
			t.Fatalf("%s: got %d want %d (%v)", q, got, want, err)
		}
	}
	var base, path string
	if err = dest.DB.QueryRow("SELECT base,path FROM relay_tasks WHERE origin='origin-daily'").Scan(&base, &path); err != nil || base != "{\"title\":\"synthetic\"}" || path != "Daily/2026-10-31.md" {
		t.Fatalf("task identity changed: %q %q %v", base, path, err)
	}
}

func TestRelayStateMigrationRefusesUnmarkedRows(t *testing.T) {
	ctx := context.Background()
	source, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "relay.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	_, err = source.Exec(`CREATE TABLE kv(key TEXT PRIMARY KEY,value TEXT NOT NULL);CREATE TABLE tasks(origin TEXT PRIMARY KEY,task_id INTEGER,phase TEXT NOT NULL,base TEXT,path TEXT,error TEXT);CREATE TABLE notes(source TEXT,identity TEXT,path TEXT,block_hash TEXT,PRIMARY KEY(source,identity));`)
	if err != nil {
		t.Fatal(err)
	}
	dest, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer dest.Close()
	if _, err = dest.DB.Exec("CREATE TABLE relay_kv(key TEXT PRIMARY KEY,value TEXT NOT NULL);INSERT INTO relay_kv VALUES('unknown','state')"); err != nil {
		t.Fatal(err)
	}
	if _, err = dest.MigrateRelayState(ctx, source); err == nil {
		t.Fatal("accepted unmarked destination state")
	}
}
