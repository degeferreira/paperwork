package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type asset struct {
	Path     string   `json:"path"`
	Owner    string   `json:"owner"`
	SHA256   string   `json:"sha256"`
	Official string   `json:"official"`
	Commands []string `json:"commands,omitempty"`
}

func owner(p string) string {
	l := strings.ToLower(p)
	if strings.Contains(l, "paperwork") {
		return "Paperwork"
	}
	for _, s := range []string{"pressroom", "website", "micropost", "blog", "publicações"} {
		if strings.Contains(l, s) {
			return "Pressroom"
		}
	}
	if strings.Contains(l, "vendor/std/") {
		return "Official SilverBullet"
	}
	return "Relay / deployment"
}
func trimTrailing(b []byte) string {
	lines := strings.Split(string(b), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t\r")
	}
	return strings.Join(lines, "\n")
}
func inventory(hub, up, out string) error {
	if hub == "" || up == "" || out == "" {
		return fmt.Errorf("hub, upstream and output required")
	}
	var assets []asset
	rx := regexp.MustCompile(`(?:command|slashCommand)\.define\s*\{\s*name\s*=\s*"([^"]+)"`)
	groups := []string{"infra/customizations", "infra/scripts", "infra/systemd", "infra/vendor/Std", "infra/website", "runtime/space/Library/Relay", "runtime/space/Library/Pressroom", "runtime/space/Library/Paperwork"}
	for _, group := range groups {
		root := filepath.Join(hub, group)
		list, e := files(root)
		if e != nil {
			return e
		}
		for _, n := range list {
			p := filepath.ToSlash(filepath.Join(group, n))
			b, e := os.ReadFile(filepath.Join(root, n))
			if e != nil {
				return e
			}
			a := asset{Path: p, Owner: owner(p), SHA256: fmt.Sprintf("%x", sha256.Sum256(b)), Official: "custom deployment or extension"}
			if group == "infra/vendor/Std" {
				official, e := os.ReadFile(filepath.Join(up, "libraries/Library/Std", n))
				if e != nil {
					a.Official = "missing upstream"
				} else if string(b) == string(official) {
					a.Official = "identical"
				} else if trimTrailing(b) == trimTrailing(official) {
					a.Official = "trailing whitespace only"
				} else {
					a.Official = "functional difference; review required"
				}
			}
			for _, m := range rx.FindAllStringSubmatch(string(b), -1) {
				a.Commands = append(a.Commands, m[1])
			}
			assets = append(assets, a)
		}
	}
	upstreamFiles, e := files(filepath.Join(up, "libraries/Library/Std"))
	if e != nil {
		return e
	}
	var missing []string
	for _, n := range upstreamFiles {
		if _, e = os.Stat(filepath.Join(hub, "infra/vendor/Std", n)); os.IsNotExist(e) {
			missing = append(missing, n)
		}
	}
	for _, p := range []string{"runtime/space/index.md", "runtime/space/Microposts.md"} {
		b, e := os.ReadFile(filepath.Join(hub, p))
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return e
		}
		assets = append(assets, asset{Path: p, Owner: owner(p), SHA256: fmt.Sprintf("%x", sha256.Sum256(b)), Official: "local entry page"})
	}
	compose, e := os.ReadFile(filepath.Join(hub, "infra/compose.yaml"))
	if e != nil {
		return e
	}
	var mounts []string
	for _, line := range strings.Split(string(compose), "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "- ") && strings.Contains(s, ":/") {
			mounts = append(mounts, strings.TrimPrefix(s, "- "))
		}
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	return jsonNew(out, map[string]any{"official_version": "2.11.1", "official_revision": "1340f7369e782a9c99e898de75cd588051fdb7cd", "assets": assets, "upstream_std_missing_locally": missing, "compose_mounts": mounts, "secrets_read": false, "production_changed": false, "scope": "source files and installed runtime libraries; no container inspection or personal-note listing"})
}
