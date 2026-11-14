package security

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
)

// NewTLSConfig carrega o par certificado/chave (falhando já na inicialização
// se forem inválidos) e retorna a configuração TLS do servidor:
//
//   - TLS 1.2 como versão mínima (TLS 1.3 é negociado quando possível);
//   - em TLS 1.2, apenas suítes ECDHE com AEAD (forward secrecy, sem CBC/RSA
//     estático). As suítes de TLS 1.3 não são configuráveis e já são seguras;
//   - curvas: padrão do Go, que inclui X25519MLKEM768 (pós-quântica híbrida).
func NewTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("security: carregar certificado TLS: %w", err)
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		CipherSuites: tls12CipherSuites(),
		Certificates: []tls.Certificate{cert},
	}, nil
}

// NewClientTLSConfig retorna a configuração TLS de cliente usada pelo
// healthcheck interno: mesma versão mínima e suítes do servidor, confiando
// apenas nas raízes informadas. A verificação de certificado nunca é
// desativada.
func NewClientTLSConfig(roots *x509.CertPool) (*tls.Config, error) {
	if roots == nil {
		return nil, errors.New("security: conjunto de raízes TLS vazio")
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		CipherSuites: tls12CipherSuites(),
		RootCAs:      roots,
	}, nil
}

func tls12CipherSuites() []uint16 {
	return []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}
}
