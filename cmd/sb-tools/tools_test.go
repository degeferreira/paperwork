package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, p, s string) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte(s), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestAllowed(t *testing.T) {
	for _, n := range []string{"x.md", "Daily/t.md", "Bibliography/citekeys.json", "Bibliography/Entries/x.bib"} {
		if !allowed(n) {
			t.Error(n)
		}
	}
	for _, n := range []string{"../x.md", "/x.md", ".git/x.md", "a/../b.md", "a//b.md", "a\\b.md", "Library/x.md", "Attachments/x.md", "secret.json", "C:/x.md"} {
		if allowed(n) {
			t.Error(n)
		}
	}
}
func TestManifest(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "VERSION"), "0.1.0\n")
	put(t, filepath.Join(root, "Library/á.md"), "hello\n")
	put(t, filepath.Join(root, "Library/Manifest.md"), "old")
	out := filepath.Join(t.TempDir(), "manifest")
	if e := manifest(root, "Relay", out); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(out)
	if strings.Contains(string(b), "- \"Manifest.md\"") || !strings.Contains(string(b), "\"á.md\"") {
		t.Fatal(string(b))
	}
	if e := manifest(root, "Relay", out); e == nil {
		t.Fatal("overwrite")
	}
}
func fixture(t *testing.T) string {
	r := t.TempDir()
	put(t, filepath.Join(r, "VERSION"), "0.1.1-dev")
	put(t, filepath.Join(r, "Library/Home.md"), "# Home")
	put(t, filepath.Join(r, "resources/Std/APIs/Command.md"), "# Synthetic native API fixture")
	put(t, filepath.Join(r, "config.example.json"), "{\"enabled\":false}")
	put(t, filepath.Join(r, "runtime/secrets/token"), "DO NOT COPY")
	return r
}
func TestPackageBootstrap(t *testing.T) {
	for _, kind := range []string{"relay", "paperwork"} {
		t.Run(kind, func(t *testing.T) {
			source := fixture(t)
			bin := filepath.Join(t.TempDir(), "bin")
			put(t, bin, "compiled fixture")
			dest := filepath.Join(t.TempDir(), "pkg")
			if e := buildPackage(source, dest, kind, bin, "", "", ""); e != nil {
				t.Fatal(e)
			}
			if _, e := os.Stat(filepath.Join(dest, "runtime")); !os.IsNotExist(e) {
				t.Fatal("runtime copied")
			}
			if e := bootstrap(dest); e != nil {
				t.Fatal(e)
			}
			if e := bootstrap(dest); e == nil {
				t.Fatal("bootstrap reset")
			}
			s, e := os.Stat(filepath.Join(dest, "runtime/secrets/silverbullet_token"))
			if e != nil || s.Mode().Perm() != 0600 {
				t.Fatal("secret permissions")
			}
			b, _ := os.ReadFile(filepath.Join(dest, "compose.yaml"))
			if !strings.Contains(string(b), "libraries/Std:/space/Library/Std:ro") {
				t.Fatal("native Std missing")
			}
			if strings.Contains(string(b), "python") {
				t.Fatal("Python compose")
			}
			if e = buildPackage(source, dest, kind, bin, "", "", ""); e == nil {
				t.Fatal("overwrite")
			}
		})
	}
}
func TestPackageSharedAndIsolation(t *testing.T) {
	source, paper, press := fixture(t), fixture(t), fixture(t)
	put(t, filepath.Join(source, "backend/hub.py"), "not shipped")
	bin := filepath.Join(t.TempDir(), "bin")
	put(t, bin, "fixture")
	dest := filepath.Join(t.TempDir(), "pkg")
	if e := buildPackage(source, dest, "relay", bin, press, paper, bin); e != nil {
		t.Fatal(e)
	}
	if e := bootstrap(dest); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(dest, "runtime/paperwork/config.json")); e != nil {
		t.Fatal(e)
	}
	put(t, filepath.Join(source, "Library/forbidden.py"), "x")
	bad := filepath.Join(t.TempDir(), "bad")
	if e := buildPackage(source, bad, "relay", bin, "", "", ""); e == nil {
		t.Fatal("Python included")
	}
	if _, e := os.Stat(bad); !os.IsNotExist(e) {
		t.Fatal("partial package left")
	}
}
func TestSymlinkRejected(t *testing.T) {
	r := fixture(t)
	if e := os.Symlink("/etc/passwd", filepath.Join(r, "Library/link")); e != nil {
		t.Skip(e)
	}
	if _, e := files(filepath.Join(r, "Library")); e == nil {
		t.Fatal("symlink accepted")
	}
}
func spaceServer(t *testing.T, body string, drift bool) *httptest.Server {
	t.Helper()
	calls := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" || r.Header.Get("X-Sync-Mode") != "true" {
			t.Error("headers")
		}
		if r.URL.Path == "/.fs" {
			calls++
			stamp := 1
			if drift && calls > 1 {
				stamp = 2
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "note.md", "lastModified": stamp}, {"name": "Library/no.md"}, {"name": "ro.md", "perm": "ro"}})
			return
		}
		if r.URL.Path != "/.fs/note.md" {
			t.Error(r.URL.Path)
		}
		_, _ = w.Write([]byte(body))
	}))
}
func TestSnapshot(t *testing.T) {
	for _, c := range []struct {
		name, body string
		drift, bad bool
	}{{"good", "hello", false, false}, {"drift", "hello", true, true}, {"secret", "contains sensitive", false, true}, {"private", "-----BEGIN OPENSSH PRIVATE KEY-----", false, true}} {
		t.Run(c.name, func(t *testing.T) {
			s := spaceServer(t, c.body, c.drift)
			defer s.Close()
			notes, e := snapshot(context.Background(), s.URL, "token", [][]byte{[]byte("sensitive")})
			if (e != nil) != c.bad {
				t.Fatalf("%v", e)
			}
			if !c.bad && len(notes) != 1 {
				t.Fatal(notes)
			}
		})
	}
}
func TestBackupGit(t *testing.T) {
	if _, e := exec.LookPath("git"); e != nil {
		t.Skip(e)
	}
	root := t.TempDir()
	infra := filepath.Join(root, "infra")
	runtime := filepath.Join(root, "runtime")
	remote := filepath.Join(root, "remote.git")
	if b, e := exec.Command("git", "init", "--bare", remote).CombinedOutput(); e != nil {
		t.Fatalf("%s %v", b, e)
	}
	put(t, filepath.Join(infra, ".env"), "NOTES_REMOTE="+remote+"\n")
	put(t, filepath.Join(runtime, "secrets/silverbullet_token"), "token")
	s := spaceServer(t, "hello", false)
	defer s.Close()
	for i := 0; i < 2; i++ {
		if e := backup(infra, runtime, s.URL); e != nil {
			t.Fatal(e)
		}
	}
	var status map[string]any
	if e := readJSON(filepath.Join(runtime, "git/status.json"), &status); e != nil || status["files"] != float64(1) {
		t.Fatal(status, e)
	}
	unlock, e := backupLock(filepath.Join(runtime, "git/backup.lock"))
	if e != nil {
		t.Fatal(e)
	}
	if e = backup(infra, runtime, s.URL); e == nil {
		t.Fatal("concurrent backup")
	}
	unlock()
	put(t, filepath.Join(infra, ".env"), "NOTES_REMOTE=other\n")
	if e = backup(infra, runtime, s.URL); e == nil || !strings.Contains(e.Error(), "remote changed") {
		t.Fatal(e)
	}
}
func TestSafeCheckout(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "secret.txt"), "x")
	if e := safeCheckout(root); e == nil {
		t.Fatal("unexpected file accepted")
	}
}
func TestTrailingWhitespace(t *testing.T) {
	if trimTrailing([]byte("a \n b\t\n")) != trimTrailing([]byte("a\n b\n")) {
		t.Fatal("whitespace")
	}
	if trimTrailing([]byte(" a\n")) == trimTrailing([]byte("a\n")) {
		t.Fatal("leading whitespace ignored")
	}
}
