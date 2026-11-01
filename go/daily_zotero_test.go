package paperwork

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/degeferreira/paperwork/go/integration"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDailyZoteroConsumersUseMigratedNamespace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PAPERWORK_CONFIG", filepath.Join(root, "config.json"))
	t.Setenv("PAPERWORK_RELAY_CONFIG", "")
	config := map[string]any{"timezone": "UTC", "silverbullet_url": "http://space", "vikunja": map[string]any{"url": "http://tasks", "inbox_id": 1}, "zotero": map[string]any{"user_id": 123}}
	settingsFixture(t, ConfigPath(), config)
	for _, name := range []string{"silverbullet_token", "vikunja_token", "zotero_key"} {
		if err := SaveCredential(name, "synthetic"); err != nil {
			t.Fatal(err)
		}
	}
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server, err := NewServer(store)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	files := map[string]string{"Daily/2026-10-31.md": "# Diário\n\n- [ ] Example <!-- sb-task:{\"origin\":\"60500c63-b114-4bc1-acda-7b83d033ca97\",\"id\":9} -->\n"}
	task := map[string]any{"id": 9, "title": "Example", "done": false, "priority": 0, "project_id": 1, "due_date": "0001-01-01T00:00:00Z", "labels": []any{}}
	writes := 0
	creates := 0
	deletedVersion := "42"
	server.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status := 200
		header := http.Header{}
		header.Set("ETag", "v1")
		var value any
		raw := ""
		switch r.URL.Host {
		case "space":
			if r.URL.Path == "/.fs" {
				list := []string{}
				for p := range files {
					list = append(list, p)
				}
				value = list
			} else {
				p := strings.TrimPrefix(r.URL.Path, "/.fs/")
				if r.Method == "GET" {
					v, ok := files[p]
					if !ok {
						status = 404
					} else {
						raw = v
					}
				} else if r.Method == "PUT" {
					b, _ := io.ReadAll(r.Body)
					files[p] = string(b)
					writes++
				} else {
					t.Fatalf("unexpected Space operation %s", r.Method)
				}
			}
		case "tasks":
			if r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "/tasks") {
				creates++
				t.Fatal("created duplicate task")
			}
			if r.URL.Path != "/tasks/9" {
				t.Fatalf("unexpected task request %s %s", r.Method, r.URL)
			}
			value = task
		case "api.zotero.org":
			header.Set("Last-Modified-Version", "42")
			switch r.URL.Path {
			case "/keys/current":
				value = map[string]any{"userID": 123, "access": map[string]any{"user": map[string]any{"library": true, "notes": true}}}
			case "/users/123/items/top":
				value = []any{map[string]any{"key": "ABCD1234", "data": map[string]any{"title": "Synthetic book", "date": "2024", "creators": []any{}}}}
			case "/users/123/items":
				value = map[string]any{}
			case "/users/123/deleted":
				header.Set("Last-Modified-Version", deletedVersion)
				value = map[string]any{"items": []any{}}
			default:
				t.Fatalf("unexpected Zotero request: %s", r.URL)
			}
		default:
			t.Fatalf("unexpected network request %s", r.URL)
		}
		if value != nil {
			b, _ := json.Marshal(value)
			raw = string(b)
		}
		return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(raw))}, nil
	})
	if err = server.Integration.State.SaveTask("60500c63-b114-4bc1-acda-7b83d033ca97", 9, "synced", integration.TaskProperties(task), "Daily/2026-10-31.md", nil); err != nil {
		t.Fatal(err)
	}
	if _, err = server.Integration.ReconcileDaily(context.Background(), "Daily/2026-10-31.md"); err != nil {
		t.Fatal(err)
	}
	first := files["Daily/2026-10-31.md"]
	if _, err = server.Integration.ReconcileDaily(context.Background(), "Daily/2026-10-31.md"); err != nil {
		t.Fatal(err)
	}
	if files["Daily/2026-10-31.md"] != first || creates != 0 {
		t.Fatal("Daily retry changed identity")
	}
	if err = server.Integration.State.SaveNote("ZOTERO", "ABCD1234", "Literature/Existing.md", "existing block"); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/v1/paperwork/zotero/search?q=Synthetic", nil)
	req.RemoteAddr = "127.0.0.1:1"
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "Literature/Existing.md") {
		t.Fatalf("search lost note mapping: %d %s", response.Code, response.Body.String())
	}
	if err = server.Integration.State.Set("zotero_version", 41); err != nil {
		t.Fatal(err)
	}
	if err = server.Integration.SyncZotero(context.Background()); err != nil {
		t.Fatal(err)
	}
	version, err := server.Integration.State.Get("zotero_version", nil)
	if err != nil || version != float64(42) {
		t.Fatalf("checkpoint: %v %v", version, err)
	}
	deletedVersion = "44"
	if err = server.Integration.SyncZotero(context.Background()); err == nil {
		t.Fatal("accepted inconsistent Zotero versions")
	}
	version, _ = server.Integration.State.Get("zotero_version", nil)
	if version != float64(42) {
		t.Fatal("advanced checkpoint after conflict")
	}
	var legacyTasks, localTasks int
	if err = store.DB.QueryRow("SELECT count(*) FROM relay_tasks").Scan(&legacyTasks); err != nil {
		t.Fatal(err)
	}
	store.DB.QueryRow("SELECT count(*) FROM tasks").Scan(&localTasks)
	if legacyTasks != 1 || localTasks != 0 {
		t.Fatal("state namespaces collided")
	}
	_ = writes
}
