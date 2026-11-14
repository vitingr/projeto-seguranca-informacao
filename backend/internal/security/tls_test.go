package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// writeSelfSignedCert grava um par certificado/chave temporário.
func writeSelfSignedCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	certFile, keyFile = filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

func TestNewTLSConfig(t *testing.T) {
	t.Parallel()

	certFile, keyFile := writeSelfSignedCert(t)
	cfg, err := NewTLSConfig(certFile, keyFile)
	if err != nil {
		t.Fatalf("NewTLSConfig() erro: %v", err)
	}
	if cfg.MinVersion < tls.VersionTLS12 {
		t.Errorf("MinVersion = %x, want >= TLS 1.2", cfg.MinVersion)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("Certificates = %d, want 1", len(cfg.Certificates))
	}
	insecure := map[uint16]bool{}
	for _, cs := range tls.InsecureCipherSuites() {
		insecure[cs.ID] = true
	}
	for _, id := range cfg.CipherSuites {
		if insecure[id] {
			t.Errorf("suíte insegura habilitada: %s", tls.CipherSuiteName(id))
		}
	}
}

func TestNewTLSConfigFailsFast(t *testing.T) {
	t.Parallel()

	certFile, keyFile := writeSelfSignedCert(t)
	tests := []struct {
		name, cert, key string
	}{
		{"arquivos inexistentes", "/nao/existe.crt", "/nao/existe.key"},
		{"cert e chave trocados", keyFile, certFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewTLSConfig(tt.cert, tt.key); err == nil {
				t.Fatal("esperava erro")
			}
		})
	}
}

func TestTLSHandshakeVersions(t *testing.T) {
	t.Parallel()

	certFile, keyFile := writeSelfSignedCert(t)
	srvCfg, err := NewTLSConfig(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes, err := os.ReadFile(certFile) //nolint:gosec // caminho gerado pelo próprio teste
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(pemBytes)

	tests := []struct {
		name    string
		version uint16
		wantErr bool
	}{
		{"TLS 1.0 recusado", tls.VersionTLS10, true},
		{"TLS 1.1 recusado", tls.VersionTLS11, true},
		{"TLS 1.2 aceito", tls.VersionTLS12, false},
		{"TLS 1.3 aceito", tls.VersionTLS13, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ln, err := tls.Listen("tcp", "127.0.0.1:0", srvCfg)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = ln.Close() }()
			go func() {
				c, acceptErr := ln.Accept()
				if acceptErr != nil {
					return
				}
				_ = c.(*tls.Conn).HandshakeContext(t.Context())
				_ = c.Close()
			}()

			clientCfg, err := NewClientTLSConfig(roots)
			if err != nil {
				t.Fatal(err)
			}
			clientCfg.MinVersion, clientCfg.MaxVersion = tt.version, tt.version //nolint:gosec // testa a recusa de versões antigas
			dialer := &tls.Dialer{Config: clientCfg}
			conn, err := dialer.DialContext(t.Context(), "tcp", ln.Addr().String())
			if (err != nil) != tt.wantErr {
				t.Fatalf("handshake erro = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			defer func() { _ = conn.Close() }()
			if v := conn.(*tls.Conn).ConnectionState().Version; !slices.Contains([]uint16{tls.VersionTLS12, tls.VersionTLS13}, v) {
				t.Errorf("versão negociada inesperada: %x", v)
			}
		})
	}
}

func TestNewClientTLSConfigRequiresRoots(t *testing.T) {
	t.Parallel()

	if _, err := NewClientTLSConfig(nil); err == nil {
		t.Fatal("esperava erro com raízes nil")
	}
}
