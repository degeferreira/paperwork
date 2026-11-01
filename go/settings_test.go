package paperwork

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func settingsFixture(t *testing.T, path string, value any) {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = PrivateWrite(path, b); err != nil {
		t.Fatal(err)
	}
}

func TestSharedOCRSettingsPreserveConnections(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PAPERWORK_CONFIG", filepath.Join(root, "paperwork.json"))
	t.Setenv("PAPERWORK_RELAY_CONFIG", filepath.Join(root, "relay.json"))
	t.Setenv("PAPERWORK_RELAY_SETTINGS", filepath.Join(root, "relay-settings"))
	settingsFixture(t, os.Getenv("PAPERWORK_RELAY_CONFIG"), map[string]any{
		"timezone": "America/Sao_Paulo", "silverbullet_url": "http://127.0.0.1:3000",
		"vikunja": map[string]any{"url": "http://127.0.0.1:3456/api/v1", "project_id": 42},
		"zotero":  map[string]any{"user_id": 123},
	})
	for _, name := range []string{"silverbullet_token", "vikunja_token", "zotero_key"} {
		if err := PrivateWrite(filepath.Join(os.Getenv("PAPERWORK_RELAY_SETTINGS"), "secrets", name), []byte("synthetic")); err != nil {
			t.Fatal(err)
		}
	}
	before, err := Configuration()
	if err != nil {
		t.Fatal(err)
	}
	s := DefaultSettings()
	s.Timezone = "America/Sao_Paulo"
	s.Provider = "local-compatible"
	s.Model = "test"
	if err = SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err = readJSON(ConfigPath(), &stored); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"silverbullet_token_file", "vikunja", "zotero"} {
		want, _ := json.Marshal(before[key])
		got, _ := json.Marshal(stored[key])
		if string(want) != string(got) {
			t.Fatalf("%s changed: %s -> %s", key, want, got)
		}
	}
	s.Timezone = "UTC"
	if err = SaveSettings(s); err != ErrShared {
		t.Fatalf("shared connection change: %v", err)
	}
	for _, name := range []string{"silverbullet_token", "vikunja_token", "zotero_key"} {
		if err = SaveCredential(name, "replacement"); err == nil {
			t.Fatalf("accepted shared credential %s", name)
		}
		b, err := os.ReadFile(filepath.Join(os.Getenv("PAPERWORK_RELAY_SETTINGS"), "secrets", name))
		if err != nil || string(b) != "synthetic" {
			t.Fatalf("shared credential changed: %s", name)
		}
	}
}

func TestStandaloneCredentialsWireConnections(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PAPERWORK_CONFIG", filepath.Join(root, "config.json"))
	t.Setenv("PAPERWORK_RELAY_CONFIG", "")
	settingsFixture(t, ConfigPath(), map[string]any{"vikunja": map[string]any{"url": "http://127.0.0.1:3456/api/v1", "project_id": 42}, "zotero": map[string]any{"group_id": 123}})
	for _, name := range []string{"silverbullet_token", "vikunja_token", "zotero_key"} {
		if err := SaveCredential(name, " synthetic "); err != nil {
			t.Fatal(err)
		}
	}
	config, err := Configuration()
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{stringValue(config["silverbullet_token_file"]), stringValue(object(config["vikunja"])["token_file"]), stringValue(object(config["zotero"])["key_file"])}
	for _, path := range paths {
		value, err := credential(path)
		if err != nil || value != "synthetic" {
			t.Fatalf("credential not wired: %s %v", path, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("credential permissions: %v", info.Mode())
		}
	}
	if number(object(config["vikunja"])["project_id"]) != 42 || number(object(config["zotero"])["group_id"]) != 123 {
		t.Fatal("connection identities changed")
	}
	if err = SaveSettings(DefaultSettings()); err != nil {
		t.Fatal(err)
	}
	config, err = Configuration()
	if err != nil {
		t.Fatal(err)
	}
	if number(object(config["vikunja"])["project_id"]) != 42 {
		t.Fatal("saving settings lost project ID")
	}
}
