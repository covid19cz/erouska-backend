package efgs

import (
	"context"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/binary"
	"encoding/pem"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"go.mozilla.org/pkcs7"
	"sort"
	"time"
	"unsafe"
)

//ToDiagnosisKey Converts ExposureKey to DiagnosisKey
func ToDiagnosisKey(symptomsSince time.Time, key *keyserverapi.ExposureKey, origin string, visitedCountries []string) *efgsapi.DiagnosisKey {
	bytes, err := b64.StdEncoding.DecodeString(key.Key)
	if err != nil {
		panic(err) // this would be very, very bad!
	}

	dsos := getDSOS(symptomsSince, key.IntervalNumber)

	return &efgsapi.DiagnosisKey{
		KeyData:                    bytes,
		RollingStartIntervalNumber: uint32(key.IntervalNumber),
		RollingPeriod:              uint32(key.IntervalCount),
		TransmissionRiskLevel:      int32(key.TransmissionRisk),
		VisitedCountries:           visitedCountries,
		Origin:                     origin,
		ReportType:                 efgsapi.ReportType_CONFIRMED_TEST,
		DaysSinceOnsetOfSymptoms:   dsos,
	}
}

func getDSOS(symptomsSince time.Time, intervalNumber int32) int32 {
	return -(int32(symptomsSince.Truncate(24*time.Hour).Unix()/600) - intervalNumber) / 144
}

func diagnosisKeyToBytes(key *efgsapi.DiagnosisKey) []byte {
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

func batchToBytes(diagnosisKey *efgsapi.DiagnosisKeyBatch) []byte {
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

func sortDiagnosisKey(keys []*efgsapi.DiagnosisKey) {
	sort.SliceStable(keys, func(i, j int) bool {
		return b64.StdEncoding.EncodeToString(diagnosisKeyToBytes(keys[j])) > b64.StdEncoding.EncodeToString(diagnosisKeyToBytes(keys[i]))
	})
}

func makeBatch(keys []*efgsapi.DiagnosisKey) efgsapi.DiagnosisKeyBatch {
	return efgsapi.DiagnosisKeyBatch{
		Keys: keys,
	}
}

func signBatch(ctx context.Context, efgsEnv efgsutils.Environment, diagnosisKey *efgsapi.DiagnosisKeyBatch) (string, error) {
	logger := logging.FromContext(ctx).Named("efgs.signBatch")

	nbbsPair, err := efgsutils.LoadX509KeyPair(ctx, efgsEnv, efgsutils.NBBS)
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
	if err != nil {
		logger.Debugf("Could not sign batch", err)
		return "", err
	}

	if efgsutils.EfgsExtendedLogging {
		logger.Debugf("Batch signature in base64: %s\n", b64.StdEncoding.EncodeToString(detachedSignature))
	}

	return b64.StdEncoding.EncodeToString(detachedSignature), nil
}

func isRecent(k *efgsapi.DiagnosisKey, now time.Time, maxAge int) bool {
	age := now.Unix() - int64(k.RollingStartIntervalNumber*600)
	return age <= int64(maxAge)*24*int64(time.Hour.Seconds())
}

func splitKeys(keys []efgsapi.ExpKey, batchSize int, maxOverlapping int) (batches []efgsapi.ExpKeyBatch) {

	addNewChunk := func(key efgsapi.ExpKey) {
		batches = append(batches, efgsapi.ExpKeyBatch{key})
	}

	// Determines whether the chunk is suitable for the key.
	// 1) It must not have too much keys that overlap with the given one.
	// 2) It must not have any unaligned keys.
	isSuitable := func(chunk efgsapi.ExpKeyBatch, key efgsapi.ExpKey) bool {
		if len(chunk) >= batchSize {
			return false
		}

		keyIntervalNumber := key.IntervalNumber

		overlapping := 0

		for _, key := range chunk {
			if key.IntervalNumber == keyIntervalNumber {
				overlapping++
			}

			// If the key is unaligned but it's not overlapping key at the same moment...
			if key.IntervalNumber+key.IntervalCount > keyIntervalNumber && key.IntervalNumber != keyIntervalNumber {
				return false
			}
		}

		return overlapping < maxOverlapping
	}

	// ------

	// First, sort the keys.
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].IntervalNumber == keys[j].IntervalNumber {
			return keys[i].IntervalCount < keys[j].IntervalCount
		}
		return keys[i].IntervalNumber < keys[j].IntervalNumber
	})

	// Try to find suitable batch for every key. When none is found, new batch is created.
	for _, key := range keys {
		suitableBatchIndex := -1
		for i, batch := range batches {
			if isSuitable(batch, key) {
				suitableBatchIndex = i
				continue
			}
		}

		if suitableBatchIndex >= 0 {
			batches[suitableBatchIndex] = append(batches[suitableBatchIndex], key)
		} else {
			addNewChunk(key)
		}
	}

	return batches
}
