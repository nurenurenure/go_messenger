package certgen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

func GenerateCerts() error {
	// Генерируем приватный ключ
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Настраиваем параметры сертификата
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Срок действия 1 год

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"MyChat Corp"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Создаем сам сертификат
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Сохраняем сертификат в server.crt
	certOut, _ := os.Create("server.crt")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	// Сохраняем ключ в server.key
	keyOut, _ := os.OpenFile("server.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	privBytes, _ := x509.MarshalECPrivateKey(priv)
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	keyOut.Close()

	return nil
}
