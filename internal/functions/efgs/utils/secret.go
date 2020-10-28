package utils

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"net/http"
)

//CertType Type of certificate to work with.
type CertType string

const (
	//NBBS National backend batch-signing certificate.
	NBBS CertType = "nbbs"
	//NBTLS National backend mTLS certificate.
	NBTLS CertType = "nbtls"
)

//X509KeyPair X509 certificate and private key pair.
type X509KeyPair struct {
	Cert []byte
	Key  []byte
}

//LoadX509KeyPair Loads certificate and key pair from Secrets Manager.
func LoadX509KeyPair(ctx context.Context, env Environment, certType CertType) (*X509KeyPair, error) {
	logger := logging.FromContext(ctx)
	secretsClient := secrets.Client{}

	certBytes, err := secretsClient.Get(fmt.Sprintf("efgs-%v-%v-cert", env, certType))
	if err != nil {
		logger.Fatalf("Error loading '%v' certificate", certType)
		return nil, err
	}

	keyBytes, err := secretsClient.Get(fmt.Sprintf("efgs-%v-%v-key", env, certType))
	if err != nil {
		logger.Fatalf("Error loading '%v' key", certType)
		return nil, err
	}

	return &X509KeyPair{
		Cert: certBytes,
		Key:  keyBytes,
	}, nil
}

//NewEFGSClient Creates new secured client for EFGS.
func NewEFGSClient(ctx context.Context, nbtlsPair *X509KeyPair) (*http.Client, error) {
	logger := logging.FromContext(ctx)

	tlsCert, err := tls.X509KeyPair(nbtlsPair.Cert, nbtlsPair.Key)
	if err != nil {
		logger.Fatalf("Error loading authentication certificate: %v", err)
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
			},
		},
	}, nil
}

//GetCertificateFingerprint Gets fingerprint of given cert.
func GetCertificateFingerprint(ctx context.Context, pair *X509KeyPair) (string, error) {
	logger := logging.FromContext(ctx)

	certBlock, _ := pem.Decode(pair.Cert)

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		logger.Fatalf("Error parsing cert: %v", err)
		return "", err
	}

	hash := sha256.Sum256(cert.Raw)

	return hex.EncodeToString(hash[:]), nil
}

//GetCertificateSubject Gets subject of given cert.
func GetCertificateSubject(ctx context.Context, pair *X509KeyPair) (string, error) {
	logger := logging.FromContext(ctx)

	certBlock, _ := pem.Decode(pair.Cert)

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		logger.Fatalf("Error parsing cert: %v", err)
	}

	return cert.Subject.ToRDNSequence().String(), nil
}
