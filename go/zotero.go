package paperwork

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func plainText(value string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(value))
	parts := []string{}
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			break
		}
		if kind == html.TextToken {
			parts = append(parts, string(tokenizer.Text()))
		}
	}
	return Literal(strings.Join(parts, " "))
}
func ZoteroNote(item map[string]any, children []map[string]any, typ string, id int) string {
	data := object(item["data"])
	key := stringValue(item["key"])
	library := "user"
	if typ != "users" {
		library = "group"
	}
	title, ok := data["title"].(string)
	if !ok {
		title = "Sem título"
	}
	libJSON, _ := json.Marshal(fmt.Sprintf("%s:%d", library, id))
	keyJSON, _ := json.Marshal(key)
	lines := []string{"---", "type: literature", "zoteroLibrary: " + string(libJSON), "zoteroKey: " + string(keyJSON), "---", "", "# " + Literal(title), "", "[Abrir item Zotero](https://www.zotero.org/" + typ + "/" + strconv.Itoa(id) + "/items/" + key + ")", "", "## Minhas notas", "", "", "## Metadados importados", "", Literal(stringValue(data["abstractNote"])), "", "## Notas Zotero", ""}
	for _, child := range children {
		c := object(child["data"])
		switch c["itemType"] {
		case "note":
			lines = append(lines, plainText(stringValue(c["note"])), "")
		case "annotation":
			lines = append(lines, "p. "+Literal(stringValue(c["annotationPageLabel"])), "> "+Literal(stringValue(c["annotationText"])), Literal(stringValue(c["annotationComment"])), "")
		}
	}
	return strings.Join(lines, "\n") + "\n"
}
func FetchZotero(ctx context.Context, client *http.Client, typ string, id int, key, token string) (map[string]any, []map[string]any, error) {
	base := fmt.Sprintf("https://api.zotero.org/%s/%d", typ, id)
	headers := map[string]string{"Zotero-API-Version": "3", "Zotero-API-Key": token}
	data, _, e := request(ctx, client, "GET", base+"/items/"+key, headers, nil)
	if e != nil {
		return nil, nil, e
	}
	var item map[string]any
	if e = json.Unmarshal(data, &item); e != nil {
		return nil, nil, e
	}
	if item["key"] != key {
		return nil, nil, errors.New("Zotero identity mismatch")
	}
	children := []map[string]any{}
	fetchChildren := func(childKey string) error {
		for start := 0; ; {
			data, h, e := request(ctx, client, "GET", fmt.Sprintf("%s/items/%s/children?start=%d&limit=100", base, childKey, start), headers, nil)
			if e != nil {
				return e
			}
			var batch []map[string]any
			if e = json.Unmarshal(data, &batch); e != nil {
				return e
			}
			children = append(children, batch...)
			start += len(batch)
			total := start
			if v := h.Get("Total-Results"); v != "" {
				total, e = strconv.Atoi(v)
				if e != nil {
					return e
				}
			}
			if len(batch) == 0 || start >= total {
				return nil
			}
		}
	}
	if e = fetchChildren(key); e != nil {
		return nil, nil, e
	}
	attachments := append([]map[string]any{}, children...)
	for _, a := range attachments {
		if object(a["data"])["itemType"] == "attachment" {
			if e = fetchChildren(stringValue(a["key"])); e != nil {
				return nil, nil, e
			}
		}
	}
	return item, children, nil
}
func (s *Store) ImportReference(ctx context.Context, alias string, space Space, client *http.Client) (map[string]any, error) {
	var typ, key string
	var id int
	var notePath *string
	e := s.DB.QueryRow("SELECT library_type,library_id,item_key,note_path FROM reference_aliases WHERE alias=? AND retired=0", alias).Scan(&typ, &id, &key, &notePath)
	if e != nil {
		return nil, e
	}
	config, e := Configuration()
	if e != nil {
		return nil, e
	}
	path := str(notePath)
	// Shared settings supply credentials only. Reference imports execute locally.
	if path == "" && typ == "users" {
		var exists int
		if e = s.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='relay_notes'").Scan(&exists); e != nil {
			return nil, e
		}
		if exists > 0 {
			var legacyUser string
			e = s.DB.QueryRow("SELECT value FROM relay_kv WHERE key='zotero_user'").Scan(&legacyUser)
			if e != nil && e != sql.ErrNoRows {
				return nil, e
			}
			if legacyUser == strconv.Itoa(id) {
				var legacyPath string
				e = s.DB.QueryRow("SELECT path FROM relay_notes WHERE source='ZOTERO' AND identity=?", key).Scan(&legacyPath)
				if e != nil && e != sql.ErrNoRows {
					return nil, e
				}
				if e == nil {
					path = strings.TrimSuffix(legacyPath, ".md")
					if !safeNote(path) {
						return nil, errors.New("Legacy reference path needs manual review")
					}
				}
			}
		}
	}

	keyFile := stringValue(object(config["zotero"])["key_file"])
	if keyFile == "" {
		keyFile = filepath.Join(filepath.Dir(ConfigPath()), "secrets/zotero_key")
	}
	token, e := credential(keyFile)
	if e != nil {
		return nil, errors.New("Zotero or SilverBullet credentials not configured")
	}
	if space == nil {
		space, e = NewSpaceConnection()
		if e != nil {
			return nil, e
		}
	}
	item, children, e := FetchZotero(ctx, client, typ, id, key, token)
	if e != nil {
		return nil, e
	}
	if path == "" {
		path = fmt.Sprintf("Literature/Zotero-%s-%d-%s", typ, id, key)
	}
	if !safeNote(path) {
		return nil, errors.New("Reference note path needs manual review before linking")
	}
	existing, _, e := space.Read(ctx, path+".md")
	if e != nil {
		return nil, e
	}
	if existing == nil {
		if e = space.Write(ctx, path+".md", ZoteroNote(item, children, typ, id), ""); e != nil {
			return nil, e
		}
	}
	_, e = s.DB.Exec("UPDATE reference_aliases SET note_path=? WHERE alias=?", path, alias)
	return map[string]any{"path": path, "alias": alias}, e
}

var _ = os.ErrNotExist
