package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"time"
)

func CreateCertificate(serverName string) ([]byte, crypto.PrivateKey, tls.Config) {
	template := &x509.Certificate{
		SerialNumber: &big.Int{},
		Subject:      pkix.Name{CommonName: serverName},
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		DNSNames:     []string{serverName},
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert, _ := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	tlsCfg := tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert},
				PrivateKey:  priv,
			},
		},
	}
	return cert, priv, tlsCfg
}

func GetQueryParam(r *http.Request, name string) string {
	params := r.URL.Query()[name]
	if len(params) == 0 {
		return ""
	}
	return params[0]
}
