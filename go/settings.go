package paperwork

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"
)

type TranscriptionConfig struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	URL             string `json:"url"`
	ExternalAllowed bool   `json:"external_allowed"`
	KeyFile         string `json:"key_file"`
}
type ConnectionConfig struct {
	URL       string `json:"url"`
	TokenFile string `json:"token_file"`
}
type Settings struct {
	Timezone        string `json:"timezone"`
	SilverBulletURL string `json:"silverbullet_url"`
	VikunjaURL      string `json:"vikunja_url"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	LocalURL        string `json:"local_url"`
	ExternalAllowed bool   `json:"external_allowed"`
}

func DefaultSettings() Settings {
	return Settings{Timezone: "UTC", SilverBulletURL: "http://127.0.0.1:3000", VikunjaURL: "http://127.0.0.1:3456/api/v1", Provider: "manual", LocalURL: "http://127.0.0.1:11434/v1"}
}
func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
func ConfigPath() string { return env("PAPERWORK_CONFIG", "/paperwork/config.json") }
func readJSON(path string, out any) error {
	data, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(data, out)
}
func object(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
func stringValue(v any) string { s, _ := v.(string); return s }
func boolValue(v any) bool     { b, _ := v.(bool); return b }
func decodeObject(v any, out any) error {
	data, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return json.Unmarshal(data, out)
}
func Configuration() (map[string]any, error) {
	config := map[string]any{}
	if e := readJSON(ConfigPath(), &config); e != nil && !os.IsNotExist(e) {
		return nil, e
	}
	delete(config, "__shared")
	if shared := os.Getenv("PAPERWORK_RELAY_CONFIG"); shared != "" {
		settings := env("PAPERWORK_RELAY_SETTINGS", "/relay-settings")
		source := map[string]any{}
		effective := map[string]any{}
		e := readJSON(filepath.Join(settings, "effective.json"), &effective)
		if e == nil {
			source = object(effective["connections"])
		} else if !os.IsNotExist(e) {
			return nil, e
		}
		if len(source) == 0 {
			if e = readJSON(shared, &source); e != nil {
				return nil, e
			}
		}
		credential := func(name string) string {
			path := filepath.Join(settings, "secrets", name)
			if _, e := os.Stat(path); e == nil {
				return path
			}
			return filepath.Join("/run/secrets", name)
		}
		for _, key := range []string{"silverbullet_url", "timezone"} {
			if _, ok := source[key]; !ok {
				return nil, errors.New("Incomplete shared settings")
			}
			config[key] = source[key]
		}
		config["silverbullet_token_file"] = credential("silverbullet_token")
		vk := object(source["vikunja"])
		vk["token_file"] = credential("vikunja_token")
		config["vikunja"] = vk
		z := object(source["zotero"])
		z["key_file"] = credential("zotero_key")
		config["zotero"] = z
		config["__shared"] = true
		for _, key := range []string{"daily_interval", "zotero_interval"} {
			if _, ok := config[key]; !ok {
				if v, found := source[key]; found {
					config[key] = v
				}
			}
		}
	}
	return config, nil
}
func PrivateWrite(path string, data []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	temp := path + ".tmp"
	f, e := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer os.Remove(temp)
	_, e = f.Write(data)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	if e = os.Rename(temp, path); e != nil {
		return e
	}
	dir, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer dir.Close()
	return dir.Sync()
}
func writeConfig(config map[string]any) error {
	data, e := json.MarshalIndent(config, "", "  ")
	if e != nil {
		return e
	}
	return PrivateWrite(ConfigPath(), append(data, '\n'))
}
func ServerURL(value string) (string, error) {
	u, e := url.Parse(value)
	if e != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return "", errors.New("Invalid server URL")
	}
	return strings.TrimRight(value, "/"), nil
}
func supportedProvider(p string) bool {
	switch p {
	case "manual", "gemini", "claude", "openai", "openai-compatible", "local-compatible":
		return true
	}
	return false
}
func GetSettings() (map[string]any, error) {
	config, e := Configuration()
	if e != nil {
		return nil, e
	}
	s := DefaultSettings()
	if v := stringValue(config["timezone"]); v != "" {
		s.Timezone = v
	}
	if v := stringValue(config["silverbullet_url"]); v != "" {
		s.SilverBulletURL = v
	}
	if v := stringValue(object(config["vikunja"])["url"]); v != "" {
		s.VikunjaURL = v
	}
	ocr := object(config["transcription"])
	if v := stringValue(ocr["provider"]); v != "" {
		s.Provider = v
	}
	s.Model = stringValue(ocr["model"])
	if v := stringValue(ocr["url"]); v != "" {
		s.LocalURL = v
	}
	s.ExternalAllowed = boolValue(ocr["external_allowed"])
	out := map[string]any{}
	_ = decodeObject(s, &out)
	profiles := map[string]any{}
	for k, v := range object(config["transcription_profiles"]) {
		m := object(v)
		u := stringValue(m["url"])
		if u == "" {
			u = DefaultSettings().LocalURL
		}
		profiles[k] = map[string]any{"model": stringValue(m["model"]), "url": u}
	}
	profiles[s.Provider] = map[string]any{"model": s.Model, "url": s.LocalURL}
	out["profiles"] = profiles
	out["shared_connections"] = boolValue(config["__shared"])
	return out, nil
}

var ErrShared = errors.New("Configure shared server connections in Relay settings")

func SaveSettings(s Settings) error {
	if !supportedProvider(s.Provider) {
		return errors.New("Unsupported provider")
	}
	if len([]rune(s.Model)) > 150 {
		return errors.New("Invalid model")
	}
	sb, e := ServerURL(s.SilverBulletURL)
	if e != nil {
		return e
	}
	vk, e := ServerURL(s.VikunjaURL)
	if e != nil {
		return e
	}
	local, e := ServerURL(s.LocalURL)
	if e != nil {
		return e
	}
	if _, e = time.LoadLocation(s.Timezone); e != nil {
		return errors.New("Invalid timezone")
	}
	config, e := Configuration()
	if e != nil {
		return e
	}
	if boolValue(config["__shared"]) && (sb != strings.TrimRight(stringValue(config["silverbullet_url"]), "/") || vk != strings.TrimRight(stringValue(object(config["vikunja"])["url"]), "/") || s.Timezone != stringValue(config["timezone"])) {
		return ErrShared
	}
	config["timezone"] = s.Timezone
	config["silverbullet_url"] = sb
	secretDir := filepath.Join(filepath.Dir(ConfigPath()), "secrets")
	if _, ok := config["silverbullet_token_file"]; !ok {
		config["silverbullet_token_file"] = filepath.Join(secretDir, "silverbullet_token")
	}
	vkConfig := object(config["vikunja"])
	vkConfig["url"] = vk
	if stringValue(vkConfig["token_file"]) == "" {
		vkConfig["token_file"] = filepath.Join(secretDir, "vikunja_token")
	}
	config["vikunja"] = vkConfig
	profiles := object(config["transcription_profiles"])
	previous := object(config["transcription"])
	if provider := stringValue(previous["provider"]); provider != "" {
		profiles[provider] = previous
	}
	profiles[s.Provider] = map[string]any{"model": s.Model, "url": local}
	config["transcription_profiles"] = profiles
	key := map[string]string{"gemini": "gemini_key", "claude": "claude_key", "openai": "openai_key", "openai-compatible": "compatible_key"}[s.Provider]
	if key == "" {
		key = "local_key"
	}
	config["transcription"] = TranscriptionConfig{s.Provider, s.Model, local, s.ExternalAllowed, filepath.Join(secretDir, key)}
	return writeConfig(config)
}
func SaveCredential(name, value string) error {
	valid := map[string]bool{"silverbullet_token": true, "vikunja_token": true, "zotero_key": true, "gemini_key": true, "claude_key": true, "openai_key": true, "compatible_key": true, "local_key": true}
	if !valid[name] {
		return os.ErrNotExist
	}
	config, e := Configuration()
	if e != nil {
		return e
	}
	if boolValue(config["__shared"]) && (name == "silverbullet_token" || name == "vikunja_token" || name == "zotero_key") {
		return errors.New("Configure shared credentials in Relay settings")
	}
	if len(value) < 1 || len(value) > 4096 {
		return errors.New("Invalid credential")
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\n\r") {
		return errors.New("Invalid credential")
	}
	path := filepath.Join(filepath.Dir(ConfigPath()), "secrets", name)
	if e = PrivateWrite(path, []byte(value+"\n")); e != nil {
		return e
	}
	switch name {
	case "silverbullet_token":
		config["silverbullet_token_file"] = path
	case "vikunja_token":
		connection := object(config["vikunja"])
		connection["token_file"] = path
		config["vikunja"] = connection
	case "zotero_key":
		connection := object(config["zotero"])
		connection["key_file"] = path
		config["zotero"] = connection
	default:
		return nil
	}
	return writeConfig(config)
}
