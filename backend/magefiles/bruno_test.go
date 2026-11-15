//go:build mage

package main

import (
	"encoding/pem"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
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

func TestBrunoPreflight(t *testing.T) {
	t.Parallel()

	healthy := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(healthy.Close)
	failing := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)
	stopped := httptest.NewTLSServer(http.NotFoundHandler())
	stopped.Close()

	// All httptest servers share the same self-signed certificate.
	cert := filepath.Join(t.TempDir(), "server.crt")
	writeTree(t, filepath.Dir(cert), map[string]string{
		"server.crt": string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: healthy.Certificate().Raw})),
		"empty.crt":  "not a certificate",
	})
	emptyCert := filepath.Join(filepath.Dir(cert), "empty.crt")

	npxFound := func(string) (string, error) { return "npx", nil }
	npxMissing := func(string) (string, error) { return "", exec.ErrNotFound }

	tests := []struct {
		name     string
		base     string
		cert     string
		lookPath func(string) (string, error)
		wantErr  string // substring of the error; empty: no error
	}{
		{"API up", healthy.URL, cert, npxFound, ""},
		{"trailing slash in the base URL", healthy.URL + "/", cert, npxFound, ""},
		{"certificate missing", healthy.URL, filepath.Join(t.TempDir(), "missing.crt"), npxFound, `run "mage cert"`},
		{"certificate without PEM data", healthy.URL, emptyCert, npxFound, "no PEM data"},
		{"npx missing", healthy.URL, cert, npxMissing, "install Node.js"},
		{"API down", stopped.URL, cert, npxFound, `start it with "mage dockerUp"`},
		{"healthz not OK", failing.URL, cert, npxFound, "answered 500"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := brunoPreflight(tt.base, tt.cert, tt.lookPath)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("brunoPreflight() error = %v, want nil", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("brunoPreflight() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestCheckLocalBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw     string
		wantErr bool
	}{
		{"", false},
		{"https://localhost", false},
		{"https://localhost:8443", false},
		{"https://127.0.0.1:8443/", false},
		{"http://localhost:8443", true},
		{"https://example.com", true},
		{"https://localhost.example.com", true},
		{"https://localhost/api", true},
		{"https://localhost?x=1", true},
		{"https://user@localhost", true},
		{"localhost:8443", true},
		{"://bad", true},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()
			if err := checkLocalBaseURL(tt.raw); (err != nil) != tt.wantErr {
				t.Errorf("checkLocalBaseURL(%q) error = %v, want error: %v", tt.raw, err, tt.wantErr)
			}
		})
	}
}

func TestBrunoSeedEnv(t *testing.T) {
	t.Parallel()

	dotEnv := map[string]string{"SEED_ADMIN_LOGIN": "admin", "SEED_ADMIN_PASSWORD": "from-dotenv"}
	noEnv := func(string) (string, bool) { return "", false }
	tests := []struct {
		name    string
		dotEnv  map[string]string
		lookup  func(string) (string, bool)
		want    []string
		wantErr string
	}{
		{"from .env", dotEnv, noEnv, []string{"SEED_ADMIN_LOGIN=admin", "SEED_ADMIN_PASSWORD=from-dotenv"}, ""},
		{
			"environment wins", dotEnv,
			func(k string) (string, bool) {
				if k == "SEED_ADMIN_PASSWORD" {
					return "from-env", true
				}
				return "", false
			},
			[]string{"SEED_ADMIN_LOGIN=admin", "SEED_ADMIN_PASSWORD=from-env"},
			"",
		},
		{"password missing", map[string]string{"SEED_ADMIN_LOGIN": "admin"}, noEnv, nil, "SEED_ADMIN_PASSWORD is empty"},
		{"password empty in .env", map[string]string{"SEED_ADMIN_LOGIN": "admin", "SEED_ADMIN_PASSWORD": ""}, noEnv, nil, `run "mage seed"`},
		{"login missing", map[string]string{"SEED_ADMIN_PASSWORD": "x"}, noEnv, nil, "SEED_ADMIN_LOGIN is empty"},
		{
			"empty in the environment hides .env", dotEnv,
			func(k string) (string, bool) { return "", k == "SEED_ADMIN_PASSWORD" },
			nil, "SEED_ADMIN_PASSWORD is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := brunoSeedEnv(tt.dotEnv, tt.lookup)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("brunoSeedEnv() error = %v, want it to contain %q", err, tt.wantErr)
				}
				if strings.Contains(err.Error(), "from-dotenv") {
					t.Error("the error contains the password")
				}
				return
			}
			if err != nil || !slices.Equal(got, tt.want) {
				t.Fatalf("brunoSeedEnv() = %v, %v, want %v", got, err, tt.want)
			}
		})
	}
}
