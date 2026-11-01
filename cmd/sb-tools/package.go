package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func files(root string) ([]string, error) {
	var list []string
	e := filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink refused")
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("special file refused")
		}
		r, _ := filepath.Rel(root, p)
		list = append(list, filepath.ToSlash(r))
		return nil
	})
	sort.Strings(list)
	return list, e
}
func copyFile(src, dst string, mode os.FileMode) error {
	b, e := os.ReadFile(src)
	if e != nil {
		return e
	}
	s, e := os.Lstat(src)
	if e != nil {
		return e
	}
	if !s.Mode().IsRegular() {
		return errors.New("regular source required")
	}
	if e = os.MkdirAll(filepath.Dir(dst), 0700); e != nil {
		return e
	}
	return newFile(dst, b, mode)
}
func copyTree(src, dst string) error {
	list, e := files(src)
	if e != nil {
		return e
	}
	for _, r := range list {
		if strings.HasSuffix(r, ".py") || strings.HasSuffix(r, ".pyc") {
			return errors.New("Python asset refused")
		}
		if e = copyFile(filepath.Join(src, r), filepath.Join(dst, r), 0600); e != nil {
			return e
		}
	}
	return nil
}
func manifest(root, name, out string) error {
	if name != "Relay" && name != "Pressroom" && name != "Paperwork" {
		return errors.New("invalid library name")
	}
	if out == "" {
		return errors.New("output required")
	}
	list, e := files(filepath.Join(root, "Library"))
	if e != nil {
		return e
	}
	v, e := os.ReadFile(filepath.Join(root, "VERSION"))
	if e != nil {
		return e
	}
	version := strings.TrimSpace(string(v))
	if strings.ContainsAny(version, "\r\n") {
		return errors.New("invalid version")
	}
	lines := []string{"---", "tags: meta/library", "name: Library/" + name + "/Manifest", "version: " + version, "files:"}
	var hashes []string
	for _, r := range list {
		if filepath.Base(r) == "Manifest.md" {
			continue
		}
		q, _ := json.Marshal(r)
		b, e := os.ReadFile(filepath.Join(root, "Library", r))
		if e != nil {
			return e
		}
		lines = append(lines, "  - "+string(q))
		hashes = append(hashes, fmt.Sprintf("  %s: %x", q, sha256.Sum256(b)))
	}
	lines = append(lines, "asset_hashes:")
	lines = append(lines, hashes...)
	lines = append(lines, "---", "", "# "+name, "", "Biblioteca para SilverBullet 2.11.1. Requer o serviço auxiliar da mesma versão.", "Instalar esta biblioteca não instala nem configura o serviço no servidor.", "")
	return newFile(out, []byte(strings.Join(lines, "\n")), 0600)
}

const sbImage = "ghcr.io/silverbulletmd/silverbullet:2.11.1-slim@sha256:07dcc90f41eaa258814080bc3f9af47854e56b23ad0c8c7ffc2ac4bc98568a92"

func buildPackage(source, dest, kind, binary, press, paper, pb string) (err error) {
	if source == "" || dest == "" || binary == "" {
		return errors.New("source, destination and binary required")
	}
	if kind != "relay" && kind != "paperwork" {
		return errors.New("invalid kind")
	}
	if kind == "paperwork" && (press != "" || paper != "") {
		return errors.New("standalone Paperwork cannot include other packages")
	}
	if paper != "" && pb == "" {
		return errors.New("paperwork-binary required")
	}
	source, _ = filepath.Abs(source)
	dest, _ = filepath.Abs(dest)
	r, _ := filepath.Rel(source, dest)
	if r == "." || (!strings.HasPrefix(r, ".."+string(os.PathSeparator)) && r != "..") {
		return errors.New("destination must be outside source")
	}
	if err = os.Mkdir(dest, 0700); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dest)
		}
	}()
	name := "Relay"
	service := "integration-service"
	if kind == "paperwork" {
		name = "Paperwork"
		service = "paperwork"
	}
	for _, x := range []struct{ s, d string }{{filepath.Join(source, "Library"), "libraries/" + name}} {
		if err = copyTree(x.s, filepath.Join(dest, x.d)); err != nil {
			return err
		}
	}
	if err = copyFile(filepath.Join(source, "config.example.json"), filepath.Join(dest, "config.example.json"), 0600); err != nil {
		return err
	}
	if err = copyFile(binary, filepath.Join(dest, "bin", service), 0700); err != nil {
		return err
	}
	if kind == "relay" {
		var config map[string]any
		if err = readJSON(filepath.Join(dest, "config.example.json"), &config); err != nil {
			return err
		}
		enabled, _ := config["enabled"].(map[string]any)
		if enabled == nil {
			enabled = map[string]any{}
		}
		enabled["daily"] = false
		enabled["zotero"] = false
		config["enabled"] = enabled
		data, e := json.MarshalIndent(config, "", "  ")
		if e != nil {
			return e
		}
		if err = os.WriteFile(filepath.Join(dest, "config.example.json"), append(data, '\n'), 0600); err != nil {
			return err
		}
	}
	version, e := os.ReadFile(filepath.Join(source, "VERSION"))
	if e != nil {
		return e
	}
	meta := map[string]any{"version": strings.TrimSpace(string(version)), "kind": kind, "silverbullet": "2.11.1", "pressroom": press != "", "implementation": "go"}
	stdlib := filepath.Join(source, "resources", "Std")
	if _, e := os.Stat(stdlib); e != nil {
		if paper != "" {
			stdlib = filepath.Join(paper, "resources", "Std")
		} else {
			stdlib = filepath.Join(source, "..", "paperwork", "resources", "Std")
		}
	}
	if err = copyTree(stdlib, filepath.Join(dest, "libraries", "Std")); err != nil {
		return fmt.Errorf("native Std library required: %w", err)
	}
	libs := []string{"Std", name}
	if press != "" {
		if err = copyTree(filepath.Join(press, "Library"), filepath.Join(dest, "libraries/Pressroom")); err != nil {
			return err
		}
		libs = append(libs, "Pressroom")
		var config map[string]any
		if err = readJSON(filepath.Join(dest, "config.example.json"), &config); err != nil {
			return err
		}
		config["website"] = map[string]bool{"enabled": false, "blog_enabled": false}
		b, _ := json.MarshalIndent(config, "", "  ")
		if err = os.WriteFile(filepath.Join(dest, "config.example.json"), append(b, '\n'), 0600); err != nil {
			return err
		}
	}
	if paper != "" {
		if err = copyTree(filepath.Join(paper, "Library"), filepath.Join(dest, "libraries/Paperwork")); err != nil {
			return err
		}
		libs = append(libs, "Paperwork")
		if err = copyFile(pb, filepath.Join(dest, "bin/paperwork"), 0700); err != nil {
			return err
		}
		if err = copyFile(filepath.Join(paper, "config.example.json"), filepath.Join(dest, "paperwork.config.example.json"), 0600); err != nil {
			return err
		}
		if err = newFile(filepath.Join(dest, "libraries/Paperwork/Deployment.md"), []byte("```space-lua\n-- priority: 20\nconfig.set(\"paperwork.port\", 8089)\n```\n"), 0600); err != nil {
			return err
		}
	}
	compose := "name: silverbullet-" + kind + "\nservices:\n  silverbullet:\n    image: " + sbImage + "\n    restart: unless-stopped\n    ports: [\"127.0.0.1:13000:3000\"]\n    environment:\n      PUID: ${PUID:-1000}\n      PGID: ${PGID:-1000}\n      SB_RUNTIME_API: \"0\"\n    volumes:\n      - ./runtime/silverbullet:/data\n      - ./runtime/space:/space\n"
	for _, n := range libs {
		compose += "      - ./libraries/" + n + ":/space/Library/" + n + ":ro\n"
	}
	compose += "    mem_limit: 384m\n    pids_limit: 128\n"
	if kind == "relay" {
		compose += worker("integration-service", false, press != "")
	}
	if kind == "paperwork" || paper != "" {
		compose += worker("paperwork", kind == "relay", false)
	}
	if err = newFile(filepath.Join(dest, "compose.yaml"), []byte(compose), 0600); err != nil {
		return err
	}
	if err = newFile(filepath.Join(dest, "Dockerfile"), []byte("FROM debian:bookworm-slim\nRUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git openssh-client poppler-utils util-linux && rm -rf /var/lib/apt/lists/*\nARG HUB_UID=1000\nARG HUB_GID=1000\nRUN groupadd -g ${HUB_GID} extensions && useradd -u ${HUB_UID} -g ${HUB_GID} -M -d /tmp extensions\nENV HOME=/tmp\nCOPY bin/ /usr/local/bin/\nUSER ${HUB_UID}:${HUB_GID}\n"), 0600); err != nil {
		return err
	}
	if err = jsonNew(filepath.Join(dest, "package.json"), meta); err != nil {
		return err
	}
	list, e := files(dest)
	if e != nil {
		return e
	}
	hashes := map[string]string{}
	for _, p := range list {
		b, e := os.ReadFile(filepath.Join(dest, p))
		if e != nil {
			return e
		}
		hashes[p] = fmt.Sprintf("%x", sha256.Sum256(b))
	}
	return jsonNew(filepath.Join(dest, "checksums.json"), hashes)
}
func worker(name string, shared, press bool) string {
	s := "  " + name + ":\n    build: .\n    init: true\n    restart: unless-stopped\n    user: \"${PUID:-1000}:${PGID:-1000}\"\n    network_mode: service:silverbullet\n    depends_on: [silverbullet]\n    command: [\"/usr/local/bin/" + name + "\"]\n    environment:\n"
	if name == "integration-service" {
		s += "      HUB_CONFIG: /config/integrations.json\n      HUB_STATE: /state/state.db\n"
		if press {
			s += "      HUB_PRESSROOM: \"1\"\n"
		}
		s += "    volumes:\n      - ./runtime/config:/config:ro\n      - ./runtime/state:/state\n      - ./runtime/secrets:/run/secrets:ro\n      - ./runtime/calendar:/calendar\n"
	} else {
		s += "      PAPERWORK_HOME: /paperwork\n      PAPERWORK_CONFIG: /paperwork/config.json\n"
		if shared {
			s += "      PAPERWORK_RELAY_CONFIG: /config/integrations.json\n      PAPERWORK_RELAY_SETTINGS: /relay-settings\n"
		}
		s += "    volumes:\n      - ./runtime/paperwork:/paperwork\n      - ./runtime/backups:/paperwork-backups\n      - ./runtime/secrets:/run/secrets:ro\n"
		if shared {
			s += "      - ./runtime/config:/config:ro\n      - ./runtime/state/configuration:/relay-settings:ro\n"
		}
	}
	return s + "    read_only: true\n    tmpfs: [/tmp]\n    cap_drop: [ALL]\n    security_opt: [no-new-privileges:true]\n    mem_limit: 256m\n    pids_limit: 96\n"
}
func bootstrap(root string) (err error) {
	if root == "" {
		return errors.New("package required")
	}
	var meta map[string]any
	if err = readJSON(filepath.Join(root, "package.json"), &meta); err != nil {
		return err
	}
	runtime := filepath.Join(root, "runtime")
	if err = os.Mkdir(runtime, 0700); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(runtime)
		}
	}()
	for _, p := range []string{"silverbullet", "space", "config", "state/configuration/secrets", "secrets", "calendar", "paperwork", "backups"} {
		if err = os.MkdirAll(filepath.Join(runtime, p), 0700); err != nil {
			return err
		}
	}
	target := "config/integrations.json"
	if meta["kind"] == "paperwork" {
		target = "paperwork/config.json"
	}
	if err = copyFile(filepath.Join(root, "config.example.json"), filepath.Join(runtime, target), 0600); err != nil {
		return err
	}
	if _, e := os.Stat(filepath.Join(root, "paperwork.config.example.json")); e == nil {
		if err = copyFile(filepath.Join(root, "paperwork.config.example.json"), filepath.Join(runtime, "paperwork/config.json"), 0600); err != nil {
			return err
		}
	}
	if meta["kind"] == "paperwork" {
		if err = newFile(filepath.Join(runtime, "space/index.md"), []byte("# Paperwork\n\n${paperwork.home()}\n\nConfigure o acesso ao Space antes de incorporar notas.\n"), 0600); err != nil {
			return err
		}
	}
	for _, n := range []string{"silverbullet_token", "vikunja_token", "zotero_key", "readeck_token"} {
		if err = newFile(filepath.Join(runtime, "secrets", n), nil, 0600); err != nil {
			return err
		}
	}
	uid, gid := os.Getuid(), os.Getgid()
	if uid < 0 {
		uid = 1000
	}
	if gid < 0 {
		gid = 1000
	}
	err = newFile(filepath.Join(root, ".env"), []byte("PUID="+strconv.Itoa(uid)+"\nPGID="+strconv.Itoa(gid)+"\n"), 0600)
	if os.IsExist(err) {
		err = nil
	}
	return err
}
