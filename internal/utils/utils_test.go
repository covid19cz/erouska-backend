package utils

import (
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestGenerateEHrid(t *testing.T) {
	for i := 0; i < 100; i++ {
		var ehrid = GenerateEHrid()

		match, err := regexp.MatchString(`e[A-Z]{6}[0-9]{3}`, ehrid)
		assert.Nil(t, err, "Failed: %v", ehrid)
		assert.True(t, match, "Failed: %v", ehrid)
	}

}
