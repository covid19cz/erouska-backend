package utils

import (
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	urlutils "net/url"
	"os"
	"strings"
)

//Environment Environment of EFGS.
type Environment string

//EfgsExtendedLogging Determines whether extended logging should be used - e.g. all raw EFGS requests and responses.
var EfgsExtendedLogging = false

func init() {
	v, exists := os.LookupEnv("EFGS_EXTENDED_LOGGING")
	if exists && v == "true" {
		EfgsExtendedLogging = true
	}
}

const (
	//EnvLocal Our local testing environment.
	EnvLocal Environment = "local"
	//EnvAcc EnvAcc env of EFGS.
	EnvAcc Environment = "acc"
)

//GetEfgsEnvironmentOrFail Gets EFGS environment from ENV variable and fails if it's not available.
func GetEfgsEnvironmentOrFail() Environment {
	efgsEnv, ok := os.LookupEnv("EFGS_ENV")
	if !ok {
		panic("EFGS_ENV must be set!")
	}

	switch strings.ToLower(efgsEnv) {
	case "local":
		return EnvLocal
	case "acc":
		return EnvAcc
	default:
		panic(fmt.Sprintf("Invalid value of EFGS_ENV: %v", efgsEnv))
	}
}

//GetEfgsURLOrFail Gets EFGS url from secrets and fails if it's not available.
func GetEfgsURLOrFail(env Environment) *urlutils.URL {
	secretsClient := secrets.Client{}
	efgsRootURL, err := secretsClient.Get(fmt.Sprintf("efgs-%v-url", env))

	if err != nil {
		panic(err)
	}

	url, err := urlutils.Parse(string(efgsRootURL))
	if err != nil {
		panic(err)
	}

	return url
}
