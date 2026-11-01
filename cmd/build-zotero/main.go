// build-zotero packages the JavaScript emitted from TypeScript for Zotero 7.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func run() error {
	root := "zotero"
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err = json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for _, name := range []string{"manifest.json", "bootstrap.js", "aliases.js"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: name, Method: zip.Store, ModifiedDate: 33}
		entry, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err = entry.Write(data); err != nil {
			return err
		}
	}
	if err = archive.Close(); err != nil {
		return err
	}
	dest := filepath.Join(root, "dist")
	if err = os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	name := "paperwork-zotero-" + manifest.Version + ".xpi"
	if err = os.WriteFile(filepath.Join(dest, name), buffer.Bytes(), 0644); err != nil {
		return err
	}
	sum := sha256.Sum256(buffer.Bytes())
	if err = os.WriteFile(filepath.Join(dest, name+".sha256"), []byte(fmt.Sprintf("%x  %s\n", sum, name)), 0644); err != nil {
		return err
	}
	fmt.Println(name)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
