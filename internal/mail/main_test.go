
package mail

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

var globalMockCAFile string
var globalMockCA *x509.Certificate
var globalMockCAPriv *rsa.PrivateKey

func TestMain(m *testing.M) {
	caPriv, _ := rsa.GenerateKey(rand.Reader, 2048)
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{Organization: []string{"Mock CA"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:         true,
	}
	caDER, _ := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPriv.PublicKey, caPriv)
	caFile, _ := os.CreateTemp("", "mock-ca-*.pem")
	pem.Encode(caFile, &pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caFile.Close()
	
	globalMockCAFile = caFile.Name()
	globalMockCA = &caTemplate
	globalMockCAPriv = caPriv
	
	os.Setenv("SSL_CERT_FILE", globalMockCAFile)
	
	code := m.Run()
	
	os.Remove(globalMockCAFile)
	os.Exit(code)
}
