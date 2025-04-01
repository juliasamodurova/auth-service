package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func ReadPrivateKey() (*rsa.PrivateKey, error) {
	privateKeyBytes, err := os.ReadFile("private.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %v", err)
	}

	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing the private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		privateKey = privateKeyInterface.(*rsa.PrivateKey)
	}

	return privateKey, nil
}

func ReadPublicKey() (*rsa.PublicKey, error) {
	publicKeyBytes, err := os.ReadFile("public.pem")
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(publicKeyBytes)
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to parse public key")
	}

	return publicKey, nil
}
