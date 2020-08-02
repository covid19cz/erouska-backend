package utils

import (
	"gopkg.in/go-playground/validator.v9"
	"math/rand"
	"time"
)

//SeededRand Seeded random
var SeededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

//Validate -_-
var Validate *validator.Validate

func init() {
	Validate = validator.New()
}

//GenerateEHrid Generate new eHrid.
func GenerateEHrid() string {
	// eLLLLLLNNN, L = letter N = number
	b := make([]byte, 10)
	b[0] = 'e'

	for i := 1; i <= 6; i++ {
		b[i] = byte(SeededRand.Intn(26) + 65)
	}

	for i := 7; i <= 9; i++ {
		b[i] = byte(SeededRand.Intn(10) + 48)
	}

	return string(b)
}

//GetTimeNow Gets current time (who yould guess)
func GetTimeNow() *time.Time {
	t := time.Now()

	return &t
}
