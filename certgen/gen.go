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
	//эллиптическая кривая NIST P-256
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	// Настраиваем параметры сертификата
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Срок действия 1 год
	//Серийный номер
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	//Шаблон сертификата
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"MyChat Corp"},
			CommonName:   "localhost",
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		//битовая маска разрешённых операций с ключом - можно использовать для шифрования сессионных ключей
		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		//сертификат используется для аутентификации сервера перед клиентом
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Создаем сам сертификат
	//&template — это самоподписанный сертификат.
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Сохраняем сертификат в server.crt
	certOut, _ := os.Create("server.crt")

	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	// Сохраняем ключ в server.key
	//O_WRONLY — только запись. O_CREATE — создать, если нет. O_TRUNC — обрезать до нуля, если существует.

	//Права 0600: ладелец: чтение + запись. Группа: нет доступа. Остальные: нет доступа.
	keyOut, _ := os.OpenFile("server.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	privBytes, _ := x509.MarshalECPrivateKey(priv)
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	keyOut.Close()

	return nil
}
