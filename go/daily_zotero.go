package paperwork

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/degeferreira/paperwork/go/integration"
)

func (s *Server) initializeIntegration() error {
	config, err := Configuration()
	if err != nil {
		return err
	}
	if stringValue(config["timezone"]) == "" {
		config["timezone"] = "UTC"
	}
	if stringValue(config["silverbullet_url"]) == "" {
		config["silverbullet_url"] = "http://127.0.0.1:3000"
	}
	state, err := integration.BindPaperworkState(s.Store.DB)
	if err != nil {
		return err
	}
	s.Integration, err = integration.NewService(config, state, s.Client)
	return err
}

func (s *Server) registerIntegration(m *http.ServeMux) {
	on := func(path string, fn func(*http.Request) (any, error)) {
		m.HandleFunc(path, s.endpoint(func(w http.ResponseWriter, r *http.Request) (any, error) {
			if err := s.reloadIntegration(); err != nil {
				return nil, err
			}
			return fn(r)
		}))
	}
	p := "/v1/paperwork"
	on("POST "+p+"/daily/open", func(r *http.Request) (any, error) { return s.Integration.OpenDaily(r.Context()) })
	on("POST "+p+"/daily/reconcile", func(r *http.Request) (any, error) {
		var b struct {
			Path string `json:"path"`
		}
		if err := decode(nil, r, &b, false); err != nil {
			return nil, err
		}
		if len(b.Path) > 200 {
			return nil, bad(422, "Invalid daily path")
		}
		return s.Integration.ReconcileDaily(r.Context(), b.Path)
	})
	on("GET "+p+"/zotero/identity", func(r *http.Request) (any, error) {
		id, err := s.Integration.ZoteroUser(r.Context())
		return map[string]any{"user_id": id}, err
	})
	on("GET "+p+"/zotero/search", func(r *http.Request) (any, error) {
		q := r.URL.Query().Get("q")
		if len([]rune(q)) > 200 {
			return nil, bad(422, "Query too long")
		}
		return s.Integration.ZoteroSearch(r.Context(), q)
	})
	on("POST "+p+"/zotero/items/{key}/import", func(r *http.Request) (any, error) { return s.Integration.ImportZotero(r.Context(), r.PathValue("key")) })
	on("GET "+p+"/zotero/items/{key}/citation", func(r *http.Request) (any, error) { return s.Integration.Citation(r.Context(), r.PathValue("key")) })
	on("POST "+p+"/zotero/items/{key}/cite", func(r *http.Request) (any, error) { return s.Integration.CiteZotero(r.Context(), r.PathValue("key")) })
	on("GET "+p+"/status", func(r *http.Request) (any, error) { return s.integrationStatus() })
	on("GET /health/ready", func(r *http.Request) (any, error) {
		return map[string]any{"ok": true}, s.Store.DB.PingContext(r.Context())
	})
}

func (s *Server) integrationStatus() (any, error) {
	jobs := map[string]any{}
	for _, name := range []string{"daily", "zotero"} {
		v, err := s.Integration.State.Get("job:"+name, map[string]any{})
		if err != nil {
			return nil, err
		}
		jobs[name] = v
	}
	var count int
	if err := s.Store.DB.QueryRow("SELECT count(*) FROM relay_tasks WHERE phase='uncertain'").Scan(&count); err != nil {
		return nil, err
	}
	return map[string]any{"jobs": jobs, "uncertain_tasks": count, "owner": "Paperwork"}, nil
}

// StartIntegrationWorkers is explicitly called by the service entrypoint, never
// by constructors or the offline migration tool. HTTP and jobs share s.mu.
func (s *Server) StartIntegrationWorkers() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		last := map[string]time.Time{}
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				for _, name := range []string{"daily", "zotero"} {
					config, err := Configuration()
					if err != nil {
						continue
					}
					if !boolValue(object(config["enabled"])[name]) {
						continue
					}
					interval := 30
					if name == "zotero" {
						interval = 600
					}
					if n, e := strconv.Atoi(stringify(config[name+"_interval"])); e == nil && n >= 15 {
						interval = n
					}
					if time.Since(last[name]) < time.Duration(interval)*time.Second {
						continue
					}
					s.mu.Lock()
					if s.ctx.Err() != nil {
						s.mu.Unlock()
						return
					}
					if err = s.reloadIntegration(); err != nil {
						s.mu.Unlock()
						continue
					}
					err = s.Integration.RunJob(s.ctx, name, func(ctx context.Context) error {
						if name == "zotero" {
							return s.Integration.SyncZotero(ctx)
						}
						files, e := s.Integration.Files(ctx)
						if e != nil {
							return e
						}
						for _, path := range files {
							if integration.DailyPattern.MatchString(path) {
								if _, e = s.Integration.ReconcileDaily(ctx, path); e != nil {
									return e
								}
							}
						}
						return nil
					})
					_ = err
					s.mu.Unlock()
					last[name] = time.Now()
				}
			}
		}
	}()
}
func stringify(v any) string { b, _ := json.Marshal(v); return string(b) }

func (s *Server) reloadIntegration() error {
	config, err := Configuration()
	if err != nil {
		return err
	}
	if stringValue(config["timezone"]) == "" {
		config["timezone"] = "UTC"
	}
	if stringValue(config["silverbullet_url"]) == "" {
		config["silverbullet_url"] = "http://127.0.0.1:3000"
	}
	zone, err := time.LoadLocation(stringValue(config["timezone"]))
	if err != nil {
		return bad(503, "Configure a valid timezone")
	}
	s.Integration.Config = config
	s.Integration.Zone = zone
	return nil
}
