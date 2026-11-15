//go:build mage

package main

import (
	"cmp"
	"context"
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

// Bruno CLI and collection layout.
const (
	brunoCLI        = "@usebruno/cli@4.3.0"
	brunoDir        = "bruno"
	brunoCollection = "backend_si"
	brunoTests      = "tests"             // the only folder "mage brunoRun" runs
	brunoEnv        = "local"             // environment in bruno/environments
	brunoBaseURL    = "https://localhost" // baseUrl of the local environment
	brunoReport     = "bruno-report.xml"  // JUnit report written by "mage brunoRun"
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

// BrunoRun runs the assertions in bruno/tests against the running API and
// writes bruno-report.xml (JUnit). It targets https://localhost, which is the
// API started by "mage dockerUp". Set BRUNO_BASE_URL for another address,
// such as https://localhost:8443 for "mage run". Requires Node.js with npx.
//
// The login flow uses the seed administrator: SEED_ADMIN_LOGIN and
// SEED_ADMIN_PASSWORD come from the environment or .env and reach the bru
// process as environment variables, never as arguments. Run "mage seed"
// first.
func BrunoRun() error {
	override := os.Getenv("BRUNO_BASE_URL")
	if err := checkLocalBaseURL(override); err != nil {
		return err
	}
	base := cmp.Or(override, brunoBaseURL)
	if err := brunoPreflight(base, certFile, exec.LookPath); err != nil {
		return err
	}

	args := []string{
		"--yes", brunoCLI, "run", brunoTests,
		"--env", brunoEnv,
		"--cacert", filepath.Join("..", certFile),
		"--ignore-truststore",
		"--reporter-junit", filepath.Join("..", brunoReport),
	}
	if override != "" {
		args = append(args, "--env-var", "baseUrl="+override)
	}
	dotEnv, err := readDotEnv(".env")
	if err != nil {
		return err
	}
	seedEnv, err := brunoSeedEnv(dotEnv, os.LookupEnv)
	if err != nil {
		return err
	}

	// The Bruno CLI must run from the collection folder.
	cmd := exec.CommandContext(context.Background(), "npx", args...)
	cmd.Dir = brunoDir
	cmd.Env = append(os.Environ(), seedEnv...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bruno run failed (report in %s): %w", brunoReport, err)
	}
	return nil
}

// brunoSeedVars are the variables the Bruno login flow reads with
// {{process.env.NAME}}.
var brunoSeedVars = []string{"SEED_ADMIN_LOGIN", "SEED_ADMIN_PASSWORD"}

// brunoSeedEnv returns the seed credentials as NAME=value entries for the
// bru process. A variable from the environment wins over .env. A missing
// value is an error: Bruno would send the literal "{{process.env.NAME}}"
// and the login would fail with a misleading 401.
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
		out = append(out, name+"="+value)
	}
	return out, nil
}

// checkLocalBaseURL accepts an empty value (the default base URL) or an https
// URL on localhost or 127.0.0.1, the only hosts the development certificate
// is valid for.
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

// brunoPreflight fails, with what to do next, when the collection cannot run:
// the certificate is missing, npx is not installed, or the API does not
// answer GET /healthz at base.
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
