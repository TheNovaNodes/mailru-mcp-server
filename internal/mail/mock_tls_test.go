package mail

import (
	"net"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// setupMockTLS creates a CA and a server certificate.
// It returns a *tls.Config for the server, and a cleanup function.
// It also sets SSL_CERT_FILE so the default HTTP client/TLS dialer trusts the CA.
func setupMockTLS(t *testing.T) (*tls.Config, func()) {
	t.Helper()
	srvPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate server private key: %v", err)
	}

	srvTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Mock Server"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	srvDER, err := x509.CreateCertificate(rand.Reader, &srvTemplate, globalMockCA, &srvPriv.PublicKey, globalMockCAPriv)
	if err != nil {
		t.Fatalf("failed to create server certificate: %v", err)
	}

	srvCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srvDER})
	srvPrivPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(srvPriv)})

	tlsCert, err := tls.X509KeyPair(srvCertPEM, srvPrivPEM)
	if err != nil {
		t.Fatalf("failed to parse server key pair: %v", err)
	}

	srvConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	return srvConfig, func() {}
}
