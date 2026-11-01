package paperwork

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestImportPersistsIdentityAcrossReopen(t *testing.T) {
	root := t.TempDir()
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(store)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	data := []byte("%PDF-1.4\nexample\n")
	body, _ := json.Marshal(map[string]string{"name": "example.pdf", "data": base64.StdEncoding.EncodeToString(data)})
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/v1/paperwork/documents", bytes.NewReader(body))
		req.RemoteAddr = "127.0.0.1:1234"
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("import: %d %s", res.Code, res.Body.String())
		}
		var got struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.ID != Hash(data) {
			t.Fatalf("identity: %q", got.ID)
		}
	}
	server.Close()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var count int
	if err := store.DB.QueryRow("SELECT count(*) FROM documents").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("duplicate documents: %d", count)
	}
}

func TestServerRejectsNonLoopback(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server, err := NewServer(store)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	req := httptest.NewRequest("GET", "/health/live", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != 403 {
		t.Fatalf("status: %d", res.Code)
	}
}
