package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"
)

func GenTlsCertificate(cn string) (tls.Certificate, error) {
	now := time.Now()

	// TODO: add randomization in template
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cn,
			Country:      []string{"US"},
			Organization: []string{"Cloudflare, Inc"},
		},
		SerialNumber:          big.NewInt(now.Unix()),
		NotBefore:             now.AddDate(0, 0, -4), // valid since N days ago
		NotAfter:              now.AddDate(0, 0, 90), // valid for N months
		BasicConstraintsValid: true,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		DNSNames:    []string{"localhost", cn},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// generate private key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(privKey.Public())
	if err != nil {
		return tls.Certificate{}, err
	}
	subjectKeyId := sha1.Sum(pubKeyBytes)
	template.SubjectKeyId = subjectKeyId[:]

	// generate certificate
	cert, err := x509.CreateCertificate(rand.Reader, template, template, privKey.Public(), privKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	var outCert tls.Certificate
	outCert.Certificate = append(outCert.Certificate, cert)
	outCert.PrivateKey = privKey

	return outCert, nil
}
