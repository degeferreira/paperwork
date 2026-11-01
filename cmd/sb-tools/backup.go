package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

func allowed(n string) bool {
	if n == "" || strings.Contains(n, "\\") || path.IsAbs(n) || path.Clean(n) != n {
		return false
	}
	for _, p := range strings.Split(n, "/") {
		if strings.HasPrefix(p, ".") || strings.Contains(p, ":") {
			return false
		}
	}
	return !strings.HasPrefix(n, "Library/") && !strings.HasPrefix(n, "Attachments/") && (strings.HasSuffix(n, ".md") || n == "Bibliography/citekeys.json" || n == "Bibliography/references.json" || (strings.HasPrefix(n, "Bibliography/Entries/") && strings.HasSuffix(n, ".bib")))
}
func envSettings(file string) (map[string]string, error) {
	b, e := os.ReadFile(file)
	if e != nil {
		return nil, e
	}
	m := map[string]string{}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		k, v, ok := strings.Cut(l, "=")
		if ok {
			m[k] = strings.TrimSpace(v)
		}
	}
	return m, nil
}

var privateKey = regexp.MustCompile(`-----BEGIN [^\r\n]*PRIVATE KEY-----`)

type noteSnapshot map[string]json.RawMessage

func snapshot(ctx context.Context, base, token string, secrets [][]byte) (map[string][]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error { return errors.New("redirect refused") }}
	get := func(suffix string) ([]byte, error) {
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, base+suffix, nil)
		if e != nil {
			return nil, errors.New("invalid Space URL")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Sync-Mode", "true")
		r, e := client.Do(req)
		if e != nil {
			return nil, errors.New("Space request failed")
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			return nil, errors.New("Space request returned non-200")
		}
		b, e := io.ReadAll(io.LimitReader(r.Body, 64<<20+1))
		if len(b) > 64<<20 {
			return nil, errors.New("Space response exceeds 64 MiB limit")
		}
		return b, e
	}
	listing := func() (noteSnapshot, error) {
		b, e := get("/.fs")
		if e != nil {
			return nil, e
		}
		var rows []json.RawMessage
		if e = json.Unmarshal(b, &rows); e != nil {
			return nil, errors.New("invalid Space listing")
		}
		m := noteSnapshot{}
		for _, r := range rows {
			var item struct {
				Name string `json:"name"`
				Perm string `json:"perm"`
			}
			if e = json.Unmarshal(r, &item); e != nil {
				return nil, e
			}
			if allowed(item.Name) && item.Perm != "ro" {
				if _, ok := m[item.Name]; ok {
					return nil, errors.New("duplicate Space filename")
				}
				m[item.Name] = r
			}
		}
		return m, nil
	}
	before, e := listing()
	if e != nil {
		return nil, e
	}
	result := map[string][]byte{}
	for n := range before {
		parts := strings.Split(n, "/")
		for i := range parts {
			parts[i] = url.PathEscape(parts[i])
		}
		b, e := get("/.fs/" + strings.Join(parts, "/"))
		if e != nil {
			return nil, e
		}
		if !utf8.Valid(b) {
			return nil, errors.New("non UTF-8 note refused")
		}
		if privateKey.Match(b) {
			return nil, errors.New("secret detected; snapshot stopped")
		}
		for _, s := range secrets {
			if len(s) > 0 && bytes.Contains(b, s) {
				return nil, errors.New("secret detected; snapshot stopped")
			}
		}
		result[n] = b
	}
	after, e := listing()
	if e != nil {
		return nil, e
	}
	// Compare decoded objects so JSON field order does not create false drift.
	decode := func(m noteSnapshot) map[string]any {
		v := map[string]any{}
		for k, r := range m {
			var x any
			_ = json.Unmarshal(r, &x)
			v[k] = x
		}
		return v
	}
	if !reflect.DeepEqual(decode(before), decode(after)) {
		return nil, errors.New("Space changed during snapshot; retry next cycle")
	}
	return result, nil
}
func collectSecrets(runtime string) (string, [][]byte, error) {
	b, e := os.ReadFile(filepath.Join(runtime, "secrets/silverbullet_token"))
	if e != nil {
		return "", nil, errors.New("SilverBullet token missing")
	}
	token := strings.TrimSpace(string(b))
	if token == "" {
		return "", nil, errors.New("SilverBullet token missing")
	}
	entries, e := os.ReadDir(filepath.Join(runtime, "secrets"))
	if e != nil {
		return "", nil, e
	}
	values := [][]byte{[]byte(token)}
	for _, d := range entries {
		if d.IsDir() {
			continue
		}
		if strings.HasSuffix(d.Name(), "_token") || strings.HasSuffix(d.Name(), "_key") {
			b, e := os.ReadFile(filepath.Join(runtime, "secrets", d.Name()))
			if e != nil {
				return "", nil, errors.New("cannot read secret scan inputs")
			}
			values = append(values, bytes.TrimSpace(b))
		}
	}
	var creds map[string]any
	e = readJSON(filepath.Join(runtime, "secrets/bootstrap_credentials.json"), &creds)
	if e != nil && !os.IsNotExist(e) {
		return "", nil, errors.New("invalid bootstrap credentials")
	}
	for _, v := range creds {
		if s, ok := v.(string); ok && s != "" {
			values = append(values, []byte(s))
		}
	}
	return token, values, nil
}
func safeCheckout(root string) error {
	st, e := os.Lstat(root)
	if e != nil {
		return e
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe checkout")
	}
	return filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("checkout symlink refused")
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			r, _ := filepath.Rel(root, p)
			if !allowed(filepath.ToSlash(r)) {
				return errors.New("unexpected file in isolated checkout")
			}
		}
		return nil
	})
}
func gitRun(ctx context.Context, repo, key string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if _, e := os.Stat(key); e == nil {
		q := "'" + strings.ReplaceAll(key, "'", "'\\''") + "'"
		cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -i "+q+" -o IdentitiesOnly=yes -o BatchMode=yes -o StrictHostKeyChecking=yes")
	}
	b, e := cmd.Output()
	if e != nil {
		return nil, errors.New("Git operation failed (diagnostics suppressed)")
	}
	return b, nil
}
func backup(infra, runtime, override string) (err error) {
	runtime, e := filepath.Abs(runtime)
	if e != nil {
		return e
	}
	gitDir := filepath.Join(runtime, "git")
	if e = os.MkdirAll(gitDir, 0700); e != nil {
		return e
	}
	unlock, e := backupLock(filepath.Join(gitDir, "backup.lock"))
	if e != nil {
		return e
	}
	defer unlock()
	statusPath := filepath.Join(gitDir, "status.json")
	status := map[string]any{}
	if e = readJSON(statusPath, &status); e != nil && !os.IsNotExist(e) {
		return errors.New("invalid previous backup status")
	}
	defer func() {
		now := time.Now().UTC().Format(time.RFC3339)
		if err != nil {
			status["error"] = err.Error()
			status["last_error"] = now
		} else {
			status["error"] = nil
			status["last_push"] = now
		}
		b, _ := json.MarshalIndent(status, "", "  ")
		tmp, e := os.CreateTemp(gitDir, ".status-")
		if e != nil {
			if err == nil {
				err = e
			}
			return
		}
		n := tmp.Name()
		defer os.Remove(n)
		_, e = tmp.Write(append(b, '\n'))
		if c := tmp.Close(); e == nil {
			e = c
		}
		if e == nil {
			e = os.Rename(n, statusPath)
		}
		if err == nil {
			err = e
		}
	}()
	settings, e := envSettings(filepath.Join(infra, ".env"))
	if e != nil {
		return errors.New("cannot read backup settings")
	}
	base := override
	if base == "" {
		port := settings["SILVERBULLET_PORT"]
		if port == "" {
			port = "13000"
		}
		base = "http://127.0.0.1:" + port
	}
	u, e := url.Parse(base)
	if e != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid Space URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 170*time.Second)
	defer cancel()
	token, secrets, e := collectSecrets(runtime)
	if e != nil {
		return e
	}
	notes, e := snapshot(ctx, strings.TrimRight(base, "/"), token, secrets)
	if e != nil {
		return e
	}
	repo := filepath.Join(gitDir, "notes")
	if e = os.MkdirAll(repo, 0700); e != nil {
		return e
	}
	if e = safeCheckout(repo); e != nil {
		return e
	}
	key := filepath.Join(runtime, "secrets/github_notes_ed25519")
	git := func(args ...string) ([]byte, error) { return gitRun(ctx, repo, key, args...) }
	if _, e = os.Stat(filepath.Join(repo, ".git")); os.IsNotExist(e) {
		for _, args := range [][]string{{"init", "-b", "main"}, {"config", "user.name", "SilverBullet backup"}, {"config", "user.email", "silverbullet@localhost"}} {
			if _, e = git(args...); e != nil {
				return e
			}
		}
	}
	list, e := filesWithoutGit(repo)
	if e != nil {
		return e
	}
	for _, n := range list {
		if _, ok := notes[n]; !ok {
			if e = os.Remove(filepath.Join(repo, n)); e != nil {
				return e
			}
		}
	}
	for n, b := range notes {
		p := filepath.Join(repo, n)
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return e
		}
		if e = os.WriteFile(p, b, 0600); e != nil {
			return e
		}
	}
	if _, e = git("add", "-A"); e != nil {
		return e
	}
	changed, e := git("diff", "--cached", "--name-only")
	if e != nil {
		return e
	}
	if len(changed) > 0 {
		if _, e = git("commit", "-m", "Notes snapshot "+time.Now().UTC().Format(time.RFC3339)); e != nil {
			return e
		}
	}
	remote := settings["NOTES_REMOTE"]
	if remote == "" {
		return errors.New("local snapshot committed; GitHub remote not configured")
	}
	names, e := git("remote")
	if e != nil {
		return e
	}
	found := false
	for _, n := range strings.Fields(string(names)) {
		if n == "origin" {
			found = true
		}
	}
	if !found {
		_, e = git("remote", "add", "origin", remote)
	} else {
		var b []byte
		b, e = git("remote", "get-url", "origin")
		if e == nil && strings.TrimSpace(string(b)) != remote {
			return errors.New("remote changed; review before pushing")
		}
	}
	if e != nil {
		return e
	}
	if _, e = git("fetch", "origin"); e != nil {
		return e
	}
	refs, e := git("for-each-ref", "--format=%(refname)", "refs/remotes/origin/main")
	if e != nil {
		return e
	}
	if len(bytes.TrimSpace(refs)) > 0 {
		if _, e = git("merge-base", "--is-ancestor", "origin/main", "HEAD"); e != nil {
			return errors.New("remote diverged; review isolated checkout; no force push or live rebase")
		}
	}
	if _, e = git("push", "origin", "HEAD:main"); e != nil {
		return e
	}
	status["files"] = len(notes)
	fmt.Printf("Snapshot pushed: %d files\n", len(notes))
	return nil
}
func filesWithoutGit(root string) ([]string, error) { return files(root) }
