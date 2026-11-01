package paperwork

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// SearchZotero reads the configured personal or group library directly. It does
// not import notes, change checkpoints, or require a running Relay service.
func SearchZotero(ctx context.Context, client *http.Client, query string) (any, error) {
	config, err := Configuration()
	if err != nil {
		return nil, err
	}
	z := object(config["zotero"])
	typ := "users"
	id := number(z["user_id"])
	if group := number(z["group_id"]); group > 0 {
		typ = "groups"
		id = group
	}
	if id <= 0 {
		return nil, bad(503, "Configure Zotero user or group identity")
	}
	keyFile := stringValue(z["key_file"])
	if keyFile == "" {
		keyFile = filepath.Join(filepath.Dir(ConfigPath()), "secrets", "zotero_key")
	}
	token, err := credential(keyFile)
	if err != nil {
		return nil, bad(503, "Configure Zotero credential")
	}
	params := url.Values{"q": {query}, "qmode": {"everything"}, "itemType": {"-attachment || note"}, "limit": {"100"}, "format": {"json"}}
	endpoint := fmt.Sprintf("https://api.zotero.org/%s/%d/items/top?%s", typ, id, params.Encode())
	data, _, err := request(ctx, client, "GET", endpoint, map[string]string{"Zotero-API-Key": token, "Zotero-API-Version": "3"}, nil)
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	if err = json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		key := stringValue(item["key"])
		if !itemRE.MatchString(key) {
			return nil, bad(502, "Invalid Zotero identity in search response")
		}
		d := object(item["data"])
		result = append(result, map[string]any{"key": key, "title": stringValue(d["title"]), "date": stringValue(d["date"]), "creators": d["creators"], "library_type": typ, "library_id": id})
	}
	return result, nil
}

func (s *Server) registerZoteroSearch(m *http.ServeMux) {
	m.HandleFunc("GET /v1/paperwork/zotero/search", s.endpoint(func(w http.ResponseWriter, r *http.Request) (any, error) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len([]rune(q)) > 200 {
			return nil, bad(422, "Query too long")
		}
		return SearchZotero(r.Context(), s.Client, q)
	}))
}
