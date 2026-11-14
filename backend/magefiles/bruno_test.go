//go:build mage

package main

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncGenerated(t *testing.T) {
	t.Parallel()

	handWritten := map[string]string{
		"opencollection.yml":     "hand-written collection",
		"environments/local.yml": "hand-written environment",
		"tests/csrf.yml":         "hand-written request",
	}
	imported := map[string]string{
		"opencollection.yml":     "generated collection",
		"environments/local.yml": "generated environment",
		"auth/folder.yml":        "auth folder",
		"auth/Log in.yml":        "new login request",
		"areas/folder.yml":       "areas folder",
	}
	// What bruno/ must hold after the sync: the hand-written files as they
	// were, plus the imported request folders and nothing else.
	synced := map[string]string{
		"opencollection.yml":     "hand-written collection",
		"environments/local.yml": "hand-written environment",
		"tests/csrf.yml":         "hand-written request",
		"auth/folder.yml":        "auth folder",
		"auth/Log in.yml":        "new login request",
		"areas/folder.yml":       "areas folder",
	}

	tests := []struct {
		name    string
		src     map[string]string
		dst     map[string]string // nil: dst does not exist yet
		want    map[string]string
		wantErr bool
	}{
		{
			name: "replaces generated folders and keeps hand-written files",
			src:  imported,
			dst: merge(handWritten, map[string]string{
				"auth/folder.yml":      "old auth folder",
				"auth/Old request.yml": "removed from the contract",
				"users/folder.yml":     "tag removed from the contract",
			}),
			want: synced,
		},
		{
			name: "running twice gives the same result",
			src:  imported,
			dst:  synced,
			want: synced,
		},
		{
			name: "creates bruno/ when it does not exist",
			src:  imported,
			want: map[string]string{
				"auth/folder.yml":  "auth folder",
				"auth/Log in.yml":  "new login request",
				"areas/folder.yml": "areas folder",
			},
		},
		{
			name:    "fails when the import has no request folders",
			src:     map[string]string{"opencollection.yml": "generated collection"},
			dst:     handWritten,
			want:    handWritten,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := t.TempDir()
			writeTree(t, src, tt.src)
			dst := filepath.Join(t.TempDir(), "bruno")
			if tt.dst != nil {
				writeTree(t, dst, tt.dst)
			}

			err := syncGenerated(src, dst, brunoHandWritten)
			if (err != nil) != tt.wantErr {
				t.Fatalf("syncGenerated() error = %v, want error: %v", err, tt.wantErr)
			}
			if got := readTree(t, dst); !maps.Equal(got, tt.want) {
				t.Errorf("bruno/ = %v, want %v", got, tt.want)
			}
		})
	}
}

func merge(a, b map[string]string) map[string]string {
	out := maps.Clone(a)
	maps.Copy(out, b)
	return out
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return files
	}
	fsys := os.DirFS(root)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		files[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
