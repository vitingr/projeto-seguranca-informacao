//go:build mage

package main

import (
	"encoding/pem"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		dst     map[string]string
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
		wantErr  string
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
			"login with the test user prefix",
			map[string]string{"SEED_ADMIN_LOGIN": "E2E.admin", "SEED_ADMIN_PASSWORD": "from-dotenv"},
			noEnv, nil, `must not start with "e2e."`,
		},
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

var loginRule = regexp.MustCompile(`^[a-z0-9._-]{3,64}$`)

func TestBrunoComumEnv(t *testing.T) {
	t.Parallel()

	parse := func(env []string) (login, password string) {
		t.Helper()
		if len(env) != 2 {
			t.Fatalf("brunoComumEnv() has %d entries, want 2", len(env))
		}
		var ok bool
		if login, ok = strings.CutPrefix(env[0], "BRUNO_COMUM_LOGIN="); !ok {
			t.Fatalf("first entry is not BRUNO_COMUM_LOGIN")
		}
		if password, ok = strings.CutPrefix(env[1], "BRUNO_COMUM_PASSWORD="); !ok {
			t.Fatalf("second entry is not BRUNO_COMUM_PASSWORD")
		}
		return login, password
	}

	login, password := parse(brunoComumEnv())
	if !loginRule.MatchString(login) || !strings.HasPrefix(login, brunoComumPrefix) || len(login) != 30 {
		t.Errorf("login %q must match the login rule, start with %q and have 30 characters", login, brunoComumPrefix)
	}
	if strings.HasPrefix(login, "bruno.") {
		t.Errorf("login %q uses the bruno. prefix", login)
	}
	if n := len(password); n < 15 || n > 128 {
		t.Errorf("password has %d characters, want 15 to 128", n)
	}

	login2, password2 := parse(brunoComumEnv())
	if login2 == login || password2 == password {
		t.Error("two calls returned the same credentials")
	}

	for _, override := range []string{"", "https://localhost:8443"} {
		args := brunoRunArgs(override)
		for _, a := range args {
			if strings.Contains(a, login) || strings.Contains(a, password) {
				t.Errorf("bru argument %q carries a credential", a)
			}
		}
		if i := slices.Index(args, "run"); i < 0 || len(args) < i+3 || args[i+1] != brunoTests || args[i+2] != brunoMatrix {
			t.Errorf("args = %v, want run %s %s", args, brunoTests, brunoMatrix)
		}
		if i := slices.Index(args, "run"); i < 0 || len(args) < i+4 || args[i+3] != brunoTickets {
			t.Errorf("args = %v, want %s after %s", args, brunoTickets, brunoMatrix)
		}
		if i := slices.Index(args, "run"); i < 0 || len(args) < i+5 || args[i+4] != brunoTracking {
			t.Errorf("args = %v, want %s after %s", args, brunoTracking, brunoTickets)
		}
		if got := fmt.Sprint(args); override != "" && !strings.Contains(got, "baseUrl="+override) {
			t.Errorf("args = %v, want the baseUrl override", args)
		}
	}
}
