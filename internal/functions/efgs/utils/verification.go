package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	keyserverapi "github.com/google/exposure-notifications-server/pkg/api/v1"
	"sort"
	"strings"
)

// CalculateExposureKeysHMAC calculates HMAC of given keys. Copied from Verification server code.
func CalculateExposureKeysHMAC(keys []keyserverapi.ExposureKey, hmacKey []byte) ([]byte, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("cannot calculate hmac on empty exposure keys")
	}
	// Sort by the key
	sort.Slice(keys, func(i int, j int) bool {
		return strings.Compare(keys[i].Key, keys[j].Key) <= 0
	})

	// Build the cleartext.
	perKeyText := make([]string, 0, len(keys))
	for _, ek := range keys {
		perKeyText = append(perKeyText,
			fmt.Sprintf("%s.%d.%d.%d", ek.Key, ek.IntervalNumber, ek.IntervalCount, ek.TransmissionRisk))
	}

	cleartext := strings.Join(perKeyText, ",")
	mac := hmac.New(sha256.New, hmacKey)
	if _, err := mac.Write([]byte(cleartext)); err != nil {
		return nil, fmt.Errorf("failed to write hmac: %w", err)
	}

	return mac.Sum(nil), nil
}
