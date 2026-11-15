//go:build mage

// Magefile do projeto. Execute "go tool mage -l" para listar os alvos.
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

// Pinned versions of the tools run with "go run". Upgrade them on purpose,
// checking that each one supports the go version in go.mod.
const (
	gooseModule   = "github.com/pressly/goose/v3/cmd/goose@v3.29.0"
	govulnModule  = "golang.org/x/vuln/cmd/govulncheck@v1.9.0"
	gooseNoDriver = "no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb"

	binaryPath = "bin/backend_si"
	certDir    = "certs"
	certFile   = "certs/server.crt"
	keyFile    = "certs/server.key"
)

// golangciLintVersion is the golangci-lint version expected on the PATH. Keep
// it in sync with GOLANGCI_LINT_VERSION in .github/workflows/backend.yml, at
// the repository root, so local and CI lint results match.
const golangciLintVersion = "2.14.0"

// Contract files and the pinned npm package that lints them. Packages run with
// npx are pinned to an exact version, never a range or a dist-tag.
const (
	contractFile  = "api/openapi.json"
	redoclyConfig = "redocly.yaml"
	redoclyCLI    = "@redocly/cli@2.72.1"
)

// Build compila o binário em bin/backend_si.
func Build() error {
	return sh.RunV("go", "build", "-trimpath", "-o", binaryPath, "./cmd/server")
}

// Run executa o servidor localmente (fora do Docker), lendo o .env.
func Run() error {
	env, err := hostEnv()
	if err != nil {
		return err
	}
	return sh.RunWithV(env, "go", "run", "./cmd/server")
}

// Test roda os testes com detector de corrida e cobertura (coverage.out).
func Test() error {
	if err := sh.RunV("go", "test", "-race", "-covermode=atomic", "-coverprofile=coverage.out", "./..."); err != nil {
		return err
	}
	if err := sh.RunV("go", "tool", "cover", "-func=coverage.out"); err != nil {
		return err
	}
	// The magefile helpers have their own tests behind the mage build tag,
	// which "./..." does not include.
	return sh.RunV("go", "test", "-race", "-tags", "mage", "./magefiles")
}

// Fmt formata o código com gofumpt e goimports (via golangci-lint fmt).
func Fmt() error {
	mg.Deps(checkGolangciLint)
	return sh.RunV("golangci-lint", "fmt")
}

// Lint executa o golangci-lint com a configuração de .golangci.yml.
func Lint() error {
	mg.Deps(checkGolangciLint)
	return sh.RunV("golangci-lint", "run")
}

// checkGolangciLint fails when the golangci-lint on the PATH is missing or is
// not the pinned version, so local results do not drift from CI.
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

// Vuln procura vulnerabilidades conhecidas com o govulncheck.
func Vuln() error {
	return sh.RunV("go", "run", govulnModule, "./...")
}

// ContractLint validates api/openapi.json as OpenAPI 3.1 with the pinned
// Redocly CLI (config in redocly.yaml). Requires Node.js with npx.
func ContractLint() error {
	return sh.RunV("npx", "--yes", redoclyCLI, "lint", contractFile, "--config", redoclyConfig)
}

// CI executa fmt, lint, vuln e test em sequência.
func CI() {
	mg.SerialDeps(Fmt, Lint, Vuln, Test)
}

// Cert gera um certificado autoassinado (ECDSA P-256) para localhost e
// 127.0.0.1 em certs/. Não sobrescreve um par já existente.
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

// MigrateUp aplica todas as migrations pendentes (goose).
func MigrateUp() error {
	return goose("up")
}

// MigrateDown reverte a última migration aplicada (goose).
func MigrateDown() error {
	return goose("down")
}

// Seed cria o primeiro administrador a partir de SEED_ADMIN_LOGIN e
// SEED_ADMIN_PASSWORD. Running it again with the same login fails with a
// conflict.
func Seed() error {
	env, err := hostEnv()
	if err != nil {
		return err
	}
	return sh.RunWithV(env, "go", "run", "./cmd/server", "seed")
}

// DockerUp sobe aplicação e PostgreSQL e espera os healthchecks.
func DockerUp() error {
	if !fileExists(certFile) || !fileExists(keyFile) {
		return errors.New("certificado ausente: execute 'mage cert' antes")
	}
	env := map[string]string{}
	// O contêiner roda com o UID/GID do usuário atual para conseguir ler a
	// chave privada (0600) montada de certs/. Nunca como root.
	if uid, gid := os.Getuid(), os.Getgid(); uid > 0 && gid > 0 {
		env["APP_UID"] = strconv.Itoa(uid)
		env["APP_GID"] = strconv.Itoa(gid)
	}
	return sh.RunWithV(env, "docker", "compose", "up", "--detach", "--build", "--wait")
}

// DockerDown derruba os contêineres (os dados do volume são mantidos).
func DockerDown() error {
	return sh.RunV("docker", "compose", "down")
}

func goose(command string) error {
	env, err := hostEnv()
	if err != nil {
		return err
	}
	// A URL vai por variável de ambiente, não por argumento (não aparece no ps).
	env["GOOSE_DRIVER"] = "postgres"
	env["GOOSE_DBSTRING"] = env["DATABASE_URL"]
	env["GOOSE_MIGRATION_DIR"] = "migrations"
	return sh.RunWithV(env, "go", "run", "-tags", gooseNoDriver, gooseModule, command)
}

// hostEnv carrega o .env (sem sobrescrever variáveis já exportadas) e, se
// DATABASE_URL não estiver definida, monta-a a partir de POSTGRES_* apontando
// para o PostgreSQL publicado em 127.0.0.1.
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

// readDotEnv lê um arquivo KEY=VALUE simples. Linhas vazias e comentários
// (#) são ignorados; aspas simples ou duplas ao redor do valor são removidas.
// Um arquivo ausente resulta em mapa vazio.
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
	// Remove antes para que a permissão nova valha mesmo se o arquivo existia.
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
