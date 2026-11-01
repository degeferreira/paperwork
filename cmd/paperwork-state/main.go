// paperwork-state operates on an offline SQLite snapshot, never starts workers.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	paperwork "github.com/degeferreira/paperwork/go"
	"net/url"
	"os"
	"path/filepath"
)

func run() error {
	source := flag.String("source", "", "consistent offline Relay SQLite snapshot")
	destination := flag.String("destination", "", "existing Paperwork library; omit for inventory only")
	stopped := flag.Bool("workers-stopped", false, "confirm both old and new workers are stopped")
	flag.Parse()
	if *source == "" {
		return errors.New("source snapshot required")
	}
	path, err := filepath.Abs(*source)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("regular snapshot file required")
	}
	uri := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return err
	}
	defer db.Close()
	var result paperwork.RelayInventory
	if *destination == "" {
		result, err = paperwork.InventoryRelayState(context.Background(), db)
	} else {
		if !*stopped {
			return errors.New("migration requires --workers-stopped")
		}
		target := filepath.Join(*destination, "paperwork.sqlite")
		destInfo, e := os.Stat(target)
		if e != nil {
			return e
		}
		if os.SameFile(info, destInfo) {
			return errors.New("source and destination are the same file")
		}
		store, e := paperwork.Open(*destination)
		if e != nil {
			return e
		}
		defer store.Close()
		result, err = store.MigrateRelayState(context.Background(), db)
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
