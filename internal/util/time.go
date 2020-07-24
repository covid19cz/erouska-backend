package utils

import (
	"time"
)

// GetTimeNow returns timestamp two weeks ago
func GetTimeNow() *time.Time {
	t := time.Now()

	return &t
}
