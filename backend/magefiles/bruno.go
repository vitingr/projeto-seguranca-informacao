//go:build mage

package main

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

const (
	brunoCLI        = "@usebruno/cli@4.3.0"
	brunoDir        = "bruno"
	brunoCollection = "backend_si"
	brunoTests      = "tests"
	brunoMatrix     = "tests/access-matrix"
	brunoTickets    = "tests/tickets"
	brunoTracking   = "tests/ticket-tracking"
	brunoEnv        = "local"
	brunoBaseURL    = "https://localhost"
	brunoReport     = "bruno-report.xml"
)

var brunoHandWritten = []string{"opencollection.yml", "environments", "tests"}

func Bruno() error {
	tmp, err := os.MkdirTemp("", "bruno-import-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

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

func BrunoRun() error {
	override := os.Getenv("BRUNO_BASE_URL")
	if err := checkLocalBaseURL(override); err != nil {
		return err
	}
	base := cmp.Or(override, brunoBaseURL)
	if err := brunoPreflight(base, certFile, exec.LookPath); err != nil {
		return err
	}

	args := brunoRunArgs(override)
	dotEnv, err := readDotEnv(".env")
	if err != nil {
		return err
	}
	seedEnv, err := brunoSeedEnv(dotEnv, os.LookupEnv)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(context.Background(), "npx", args...)
	cmd.Dir = brunoDir
	cmd.Env = slices.Concat(os.Environ(), seedEnv, brunoComumEnv())
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bruno run failed (report in %s): %w", brunoReport, err)
	}
	return nil
}

func brunoRunArgs(override string) []string {
	args := []string{
		"--yes", brunoCLI, "run", brunoTests, brunoMatrix, brunoTickets, brunoTracking,
		"--env", brunoEnv,
		"--cacert", filepath.Join("..", certFile),
		"--ignore-truststore",
		"--reporter-junit", filepath.Join("..", brunoReport),
	}
	if override != "" {
		args = append(args, "--env-var", "baseUrl="+override)
	}
	return args
}

const brunoComumPrefix = "e2e."

func brunoComumEnv() []string {
	return []string{
		"BRUNO_COMUM_LOGIN=" + brunoComumPrefix + strings.ToLower(rand.Text()),
		"BRUNO_COMUM_PASSWORD=" + rand.Text(),
	}
}

var brunoSeedVars = []string{"SEED_ADMIN_LOGIN", "SEED_ADMIN_PASSWORD"}

func brunoSeedEnv(dotEnv map[string]string, lookup func(string) (string, bool)) ([]string, error) {
	out := make([]string, 0, len(brunoSeedVars))
	for _, name := range brunoSeedVars {
		value, ok := lookup(name)
		if !ok {
			value = dotEnv[name]
		}
		if value == "" {
			return nil, fmt.Errorf("%s is empty: set SEED_ADMIN_LOGIN and SEED_ADMIN_PASSWORD in .env and run \"mage seed\"", name)
		}
		if name == "SEED_ADMIN_LOGIN" && strings.HasPrefix(strings.ToLower(value), brunoComumPrefix) {
			return nil, fmt.Errorf("SEED_ADMIN_LOGIN must not start with %q, the prefix of the test users", brunoComumPrefix)
		}
		out = append(out, name+"="+value)
	}
	return out, nil
}

func checkLocalBaseURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.User != nil {
		return fmt.Errorf("BRUNO_BASE_URL=%q must look like https://localhost:8443 (https, localhost or 127.0.0.1, no path)", raw)
	}
	return nil
}

func brunoPreflight(base, cert string, lookPath func(string) (string, error)) error {
	pemData, err := os.ReadFile(cert)
	if err != nil {
		return fmt.Errorf("certificate %s not found: run \"mage cert\" first: %w", cert, err)
	}
	if _, err = lookPath("npx"); err != nil {
		return fmt.Errorf("npx not found: install Node.js (LTS) to run the Bruno CLI: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		return fmt.Errorf("certificate %s has no PEM data: delete certs/ and run \"mage cert\"", cert)
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}},
	}
	healthURL := strings.TrimSuffix(base, "/") + "/healthz"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, healthURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("invalid base URL %q: %w", base, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("the API is not answering %s: start it with \"mage dockerUp\" (or set BRUNO_BASE_URL for \"mage run\"): %w", healthURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %d instead of 200: check the API logs", healthURL, resp.StatusCode)
	}
	return nil
}

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
