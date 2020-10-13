package efgs

import (
	"context"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/binary"
	"encoding/pem"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"go.mozilla.org/pkcs7"
	"sort"
	"unsafe"
)

func diagnosisKeyToBytes(key *DiagnosisKey) []byte {
	var fieldSeparator byte = '.'
	var keyBytes []byte
	keyBytes = append(keyBytes, []byte(b64.StdEncoding.EncodeToString(key.KeyData))[:]...)
	keyBytes = append(keyBytes, fieldSeparator)
	keyBytes = append(keyBytes, intToByteArray(int32(key.RollingStartIntervalNumber))[:]...)
	keyBytes = append(keyBytes, fieldSeparator)
	keyBytes = append(keyBytes, intToByteArray(int32(key.RollingPeriod))[:]...)
	keyBytes = append(keyBytes, fieldSeparator)
	keyBytes = append(keyBytes, intToByteArray(key.TransmissionRiskLevel)[:]...)
	keyBytes = append(keyBytes, fieldSeparator)
	sortVisitedCountries(key.VisitedCountries)
	keyBytes = append(keyBytes, serializeVisitedCountries(key.VisitedCountries)[:]...)
	keyBytes = append(keyBytes, fieldSeparator)
	keyBytes = append(keyBytes, []byte(b64.StdEncoding.EncodeToString([]byte(key.Origin)))[:]...)
	keyBytes = append(keyBytes, fieldSeparator)
	keyBytes = append(keyBytes, intToByteArray(int32(key.ReportType.Number()))[:]...)
	keyBytes = append(keyBytes, fieldSeparator)
	keyBytes = append(keyBytes, intToByteArray(key.DaysSinceOnsetOfSymptoms)[:]...)
	keyBytes = append(keyBytes, fieldSeparator)
	return keyBytes
}

func batchToBytes(diagnosisKey *DiagnosisKeyBatch) []byte {
	var rawDiagnosisKey []byte
	for _, k := range diagnosisKey.Keys {
		rawDiagnosisKey = append(rawDiagnosisKey, diagnosisKeyToBytes(k)[:]...)
	}
	return rawDiagnosisKey
}

func intToByteArray(num int32) []byte {
	arr := make([]byte, int(unsafe.Sizeof(num)))
	binary.BigEndian.PutUint32(arr, uint32(num))
	return []byte(b64.StdEncoding.EncodeToString(arr))
}

func sortVisitedCountries(countries []string) {
	sort.SliceStable(countries, func(i, j int) bool {
		return b64.StdEncoding.EncodeToString([]byte(countries[j])) > b64.StdEncoding.EncodeToString([]byte(countries[i]))
	})
}

func serializeVisitedCountries(countries []string) []byte {
	var visitedCountries []byte
	for i := 0; i < len(countries); i++ {
		visitedCountries = append(visitedCountries, []byte(countries[i])[:]...)
		if i != (len(countries) - 1) {
			visitedCountries = append(visitedCountries, ',')
		}
	}

	return []byte(b64.StdEncoding.EncodeToString(visitedCountries))
}

func sortDiagnosisKey(keys []*DiagnosisKey) {
	sort.SliceStable(keys, func(i, j int) bool {
		return b64.StdEncoding.EncodeToString(diagnosisKeyToBytes(keys[j])) > b64.StdEncoding.EncodeToString(diagnosisKeyToBytes(keys[i]))
	})
}

func makeBatch(keys []*DiagnosisKey) DiagnosisKeyBatch {
	return DiagnosisKeyBatch{
		Keys: keys,
	}
}

func signBatch(ctx context.Context, diagnosisKey *DiagnosisKeyBatch) (string, error) {
	logger := logging.FromContext(ctx)

	nbbsPair, err := efgsutils.LoadX509KeyPair(ctx, efgsutils.NBBS)
	if err != nil {
		logger.Debugf("Error loading authentication certificate: %v", err)
		return "", err
	}

	certBlock, _ := pem.Decode(nbbsPair.Cert)
	keyBlock, _ := pem.Decode(nbbsPair.Key)

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		logger.Debugf("Certification parsing error: %v", err)
		return "", err
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		logger.Debugf("Private key parsing error: %v", err)
		return "", err
	}

	var rawDiagnosisKey []byte
	for _, k := range diagnosisKey.Keys {
		rawDiagnosisKey = append(rawDiagnosisKey, diagnosisKeyToBytes(k)[:]...)
	}

	signedBatch, err := pkcs7.NewSignedData(rawDiagnosisKey)
	if err != nil {
		return "", err
	}

	if err := signedBatch.AddSigner(cert, key, pkcs7.SignerInfoConfig{}); err != nil {
		return "", err
	}

	signedBatch.Detach()
	detachedSignature, err := signedBatch.Finish()

	logger.Infof("Batch signature in base64: %s\n", b64.StdEncoding.EncodeToString(detachedSignature))

	return b64.StdEncoding.EncodeToString(detachedSignature), nil
}
