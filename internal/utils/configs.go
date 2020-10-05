package utils

import (
	"context"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"github.com/sethvargo/go-envconfig"
	urlutils "net/url"
)

//KeyServerConfig Configuration of KeyServer.
type KeyServerConfig struct {
	URL string `env:"KEY_SERVER_URL, required"`
}

//VerificationServerConfig Configuration of Verification server.
type VerificationServerConfig struct {
	AdminURL  string `env:"VERIFICATION_SERVER_ADMIN_URL, required"`
	DeviceURL string `env:"VERIFICATION_SERVER_DEVICE_URL, required"`
	AdminKey  string
	DeviceKey string
}

//LoadKeyServerConfig Load KeyServer config.
func LoadKeyServerConfig(ctx context.Context) (*KeyServerConfig, error) {
	logger := logging.FromContext(ctx)

	var keyServerConfig KeyServerConfig
	if err := envconfig.Process(ctx, &keyServerConfig); err != nil {
		logger.Debugf("Could not load KeyServerConfig: %v", err)
		return nil, err
	}

	return &keyServerConfig, nil
}

//LoadVerificationServerConfig Load Verification server config.
func LoadVerificationServerConfig(ctx context.Context) (*VerificationServerConfig, error) {
	logger := logging.FromContext(ctx)

	var verificationServerConfig VerificationServerConfig
	if err := envconfig.Process(ctx, &verificationServerConfig); err != nil {
		logger.Debugf("Could not load VerificationServerConfig: %v", err)
		return nil, err
	}

	// load the rest from secrets manager; requires special access rights

	secretsClient := secrets.Client{}

	bytes, err := secretsClient.Get("verificationserver-admin-key")
	if err != nil {
		logger.Debugf("Could not load VerificationServerConfig: %v", err)
		return nil, err
	}

	verificationServerConfig.AdminKey = string(bytes)

	bytes, err = secretsClient.Get("verificationserver-device-key")
	if err != nil {
		logger.Debugf("Could not load VerificationServerConfig: %v", err)
		return nil, err
	}

	verificationServerConfig.DeviceKey = string(bytes)

	return &verificationServerConfig, nil
}

//GetURL Gets configured url with given path set. It does URL verification but it also ensures that a valid URL comes
//out of it, no matter if the original one (passed to ENV) included some path or trailing slash etc.
func (c *KeyServerConfig) GetURL(path string) string {
	url, err := urlutils.Parse(c.URL)
	if err != nil {
		panic(err)
	}

	url.Path = path
	return url.String()
}

//GetDeviceURL Gets configured device url with given path set. It does URL verification but it also ensures that a valid URL comes
//out of it, no matter if the original one (passed to ENV) included some path or trailing slash etc.
func (c *VerificationServerConfig) GetDeviceURL(path string) string {
	url, err := urlutils.Parse(c.DeviceURL)
	if err != nil {
		panic(err)
	}

	url.Path = path
	return url.String()
}

//GetAdminURL Gets configured admin url with given path set. It does URL verification but it also ensures that a valid URL comes
//out of it, no matter if the original one (passed to ENV) included some path or trailing slash etc.
func (c *VerificationServerConfig) GetAdminURL(path string) string {
	url, err := urlutils.Parse(c.AdminURL)
	if err != nil {
		panic(err)
	}

	url.Path = path
	return url.String()
}
