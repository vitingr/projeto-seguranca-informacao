//go:build mage

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/magefile/mage/sh"
)

// Bruno CLI and collection layout.
const (
	brunoCLI        = "@usebruno/cli@4.3.0"
	brunoDir        = "bruno"
	brunoCollection = "backend_si"
)

// brunoHandWritten are the entries of bruno/ that "mage bruno" never touches.
// Everything else in bruno/ is generated from the contract.
var brunoHandWritten = []string{"opencollection.yml", "environments", "tests"}

// Bruno regenerates the request folders of the Bruno collection (one per
// contract tag) from api/openapi.json. The hand-written files are kept.
// Requires Node.js with npx.
func Bruno() error {
	tmp, err := os.MkdirTemp("", "bruno-import-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	// The CLI writes the collection to <output>/<collection name>/.
	err = sh.RunV("npx", "--yes", brunoCLI, "import", "openapi",
		"--source", contractFile,
		"--output", tmp,
		"--collection-name", brunoCollection,
		"--collection-format", "opencollection",
		"--group-by", "tags")
	if err != nil {
		return fmt.Errorf("import the contract with the Bruno CLI: %w", err)
	}
	return syncGenerated(filepath.Join(tmp, brunoCollection), brunoDir, brunoHandWritten)
}

// syncGenerated makes dst hold the generated entries of src: it removes every
// entry of dst that is not in keep, then copies every entry of src that is
// not in keep. Entries in keep are left untouched on both sides.
func syncGenerated(src, dst string, keep []string) error {
	srcEntries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read the generated collection: %w", err)
	}
	if !slices.ContainsFunc(srcEntries, func(e fs.DirEntry) bool { return !slices.Contains(keep, e.Name()) }) {
		return fmt.Errorf("the generated collection in %s has no request folders", src)
	}

	err = os.MkdirAll(dst, 0o750)
	if err != nil {
		return err
	}
	dstEntries, err := os.ReadDir(dst)
	if err != nil {
		return err
	}
	for _, e := range dstEntries {
		if slices.Contains(keep, e.Name()) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	for _, e := range srcEntries {
		if slices.Contains(keep, e.Name()) {
			continue
		}
		if err := copyEntry(src, dst, e); err != nil {
			return err
		}
	}
	return nil
}

// copyEntry copies the entry e of the directory src into dst. Directories
// are copied with os.CopyFS, which only reads inside the given root.
func copyEntry(src, dst string, e fs.DirEntry) error {
	if e.IsDir() {
		return os.CopyFS(filepath.Join(dst, e.Name()), os.DirFS(filepath.Join(src, e.Name())))
	}
	data, err := fs.ReadFile(os.DirFS(src), e.Name())
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, e.Name()), data, 0o600)
}
