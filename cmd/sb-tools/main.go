// sb-tools is offline packaging and migration tooling. It never starts services.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: sb-tools package|bootstrap|manifest|notes-backup|inventory [flags]")
	}
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	switch args[0] {
	case "package":
		source := f.String("source", "", "Relay or Paperwork checkout")
		dest := f.String("destination", "", "new package directory")
		kind := f.String("kind", "relay", "relay or paperwork")
		binary := f.String("binary", "", "compiled service executable")
		press := f.String("pressroom", "", "optional private Pressroom checkout (Relay binary must include it)")
		paper := f.String("paperwork", "", "optional Paperwork checkout")
		pb := f.String("paperwork-binary", "", "compiled Paperwork executable")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		return buildPackage(*source, *dest, *kind, *binary, *press, *paper, *pb)
	case "bootstrap":
		root := f.String("package", "", "clean package directory")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		return bootstrap(*root)
	case "manifest":
		root := f.String("source", "", "checkout containing Library and VERSION")
		name := f.String("name", "Relay", "Relay, Pressroom or Paperwork")
		out := f.String("output", "", "new output file (never edits checkout)")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		return manifest(*root, *name, *out)
	case "notes-backup":
		infra := f.String("infra", ".", "infra directory, defaults match legacy script layout")
		runtime := f.String("runtime", "", "runtime directory; default sibling of infra")
		url := f.String("url", "", "override local Space URL")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		if *runtime == "" {
			*runtime = filepath.Join(*infra, "..", "runtime")
		}
		return backup(*infra, *runtime, *url)
	case "inventory":
		hub := f.String("hub", "", "silverbullet-hub root")
		up := f.String("upstream", "", "official SilverBullet checkout")
		out := f.String("output", "", "new JSON output")
		if e := f.Parse(args[1:]); e != nil {
			return e
		}
		return inventory(*hub, *up, *out)
	}
	return errors.New("unknown command")
}
func readJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func newFile(path string, b []byte, mode os.FileMode) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	c := f.Close()
	if e != nil {
		return e
	}
	return c
}
func jsonNew(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return newFile(path, append(b, '\n'), 0600)
}
