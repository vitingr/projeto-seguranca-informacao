//go:build mage

package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	gooseModule   = "github.com/pressly/goose/v3/cmd/goose@v3.29.0"
	govulnModule  = "golang.org/x/vuln/cmd/govulncheck@v1.9.0"
	gooseNoDriver = "no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb"

	binaryPath = "bin/backend_si"
	certDir    = "certs"
	certFile   = "certs/server.crt"
	keyFile    = "certs/server.key"
)

const golangciLintVersion = "2.14.0"

const (
	contractFile  = "api/openapi.json"
	redoclyConfig = "redocly.yaml"
	redoclyCLI    = "@redocly/cli@2.72.1"
)

func Build() error {
	return sh.RunV("go", "build", "-trimpath", "-o", binaryPath, "./cmd/server")
}

func Run() error {
	env, err := hostEnv()
	if err != nil {
		return err
	}
	return sh.RunWithV(env, "go", "run", "./cmd/server")
}

func Test() error {
	if err := sh.RunV("go", "test", "-race", "-covermode=atomic", "-coverprofile=coverage.out", "./..."); err != nil {
		return err
	}
	if err := sh.RunV("go", "tool", "cover", "-func=coverage.out"); err != nil {
		return err
	}
	return sh.RunV("go", "test", "-race", "-tags", "mage", "./magefiles")
}

func TestIntegration() error {
	env, err := hostEnv()
	if err != nil {
		return err
	}
	return sh.RunWithV(map[string]string{"TEST_DATABASE_URL": env["DATABASE_URL"]},
		"go", "test", "-race", "-count=1", "-run", "Integration", "-v", "./...")
}

func Fmt() error {
	mg.Deps(checkGolangciLint)
	return sh.RunV("golangci-lint", "fmt")
}

func Lint() error {
	mg.Deps(checkGolangciLint)
	return sh.RunV("golangci-lint", "run")
}

func checkGolangciLint() error {
	out, err := sh.Output("golangci-lint", "version", "--short")
	if err != nil {
		return fmt.Errorf("golangci-lint v%s is required but could not be run: %w", golangciLintVersion, err)
	}
	if got := strings.TrimPrefix(strings.TrimSpace(out), "v"); got != golangciLintVersion {
		return fmt.Errorf("golangci-lint v%s is required, found v%s: install the pinned version (see GOLANGCI_LINT_VERSION in .github/workflows/backend.yml)", golangciLintVersion, got)
	}
	return nil
}

func Vuln() error {
	return sh.RunV("go", "run", govulnModule, "./...")
}

func ContractLint() error {
	return sh.RunV("npx", "--yes", redoclyCLI, "lint", contractFile, "--config", redoclyConfig)
}

func CI() {
	mg.SerialDeps(Fmt, Lint, Vuln, Test)
}

func Cert() error {
	if fileExists(certFile) && fileExists(keyFile) {
		fmt.Println("certs/server.crt e certs/server.key já existem; nada a fazer")
		return nil
	}
	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return err
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("gerar chave: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("gerar número de série: %w", err)
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "localhost", Organization: []string{"backend_si (desenvolvimento)"}},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("criar certificado: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("serializar chave: %w", err)
	}

	if err := writePEM(keyFile, "PRIVATE KEY", keyDER, 0o600); err != nil {
		return err
	}
	if err := writePEM(certFile, "CERTIFICATE", certDER, 0o644); err != nil {
		return err
	}
	fmt.Printf("certificado gerado em %s (válido até %s)\n", certFile, tmpl.NotAfter.Format(time.DateOnly))
	return nil
}

func MigrateUp() error {
	return goose("up")
}

func MigrateDown() error {
	return goose("down")
}

func Seed() error {
	env, err := hostEnv()
	if err != nil {
		return err
	}
	return sh.RunWithV(env, "go", "run", "./cmd/server", "seed")
}

func DockerUp() error {
	if !fileExists(certFile) || !fileExists(keyFile) {
		return errors.New("certificado ausente: execute 'mage cert' antes")
	}
	env := map[string]string{}
	if uid, gid := os.Getuid(), os.Getgid(); uid > 0 && gid > 0 {
		env["APP_UID"] = strconv.Itoa(uid)
		env["APP_GID"] = strconv.Itoa(gid)
	}
	return sh.RunWithV(env, "docker", "compose", "up", "--detach", "--build", "--wait")
}

func DockerDown() error {
	return sh.RunV("docker", "compose", "down")
}

func goose(command string) error {
	env, err := hostEnv()
	if err != nil {
		return err
	}
	env["GOOSE_DRIVER"] = "postgres"
	env["GOOSE_DBSTRING"] = env["DATABASE_URL"]
	env["GOOSE_MIGRATION_DIR"] = "migrations"
	return sh.RunWithV(env, "go", "run", "-tags", gooseNoDriver, gooseModule, command)
}

func hostEnv() (map[string]string, error) {
	env, err := readDotEnv(".env")
	if err != nil {
		return nil, err
	}
	for k := range env {
		if v, ok := os.LookupEnv(k); ok {
			env[k] = v
		}
	}
	get := func(k string) string {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
		return env[k]
	}

	if get("DATABASE_URL") == "" {
		user, pass, db := get("POSTGRES_USER"), get("POSTGRES_PASSWORD"), get("POSTGRES_DB")
		if user == "" || pass == "" || db == "" {
			return nil, errors.New("defina DATABASE_URL ou POSTGRES_USER/POSTGRES_PASSWORD/POSTGRES_DB no .env")
		}
		port := get("POSTGRES_HOST_PORT")
		if port == "" {
			port = "5432"
		}
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(user, pass),
			Host:     net.JoinHostPort("127.0.0.1", port),
			Path:     "/" + db,
			RawQuery: "sslmode=disable",
		}
		env["DATABASE_URL"] = u.String()
	} else {
		env["DATABASE_URL"] = get("DATABASE_URL")
	}
	return env, nil
}

func readDotEnv(path string) (map[string]string, error) {
	env := map[string]string{}
	f, err := os.Open(filepath.Clean(path))
	if errors.Is(err, os.ErrNotExist) {
		return env, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s:%d: linha sem '='", path, n)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		env[key] = value
	}
	return env, sc.Err()
}

func writePEM(path, blockType string, der []byte, perm os.FileMode) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
