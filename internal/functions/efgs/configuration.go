package efgs

import (
	"context"
	"encoding/json"
	"fmt"
	efgsdatabase "github.com/covid19cz/erouska-backend/internal/functions/efgs/database"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/sethvargo/go-envconfig"
	"net/http"
	urlutils "net/url"
	"os"
	"strconv"
)

type uploadConfig struct {
	URL            *urlutils.URL
	Env            efgsutils.Environment
	NBTLSPair      *efgsutils.X509KeyPair
	Client         *http.Client
	Database       *efgsdatabase.Connection
	BatchSizeLimit int
	BatchTag       string
}

type publishConfig struct {
	VerificationServer *utils.VerificationServerConfig
	KeyServer          *utils.KeyServerConfig
	Client             *http.Client
	MaxBatchSize       int `env:"MAX_UPLOAD_KEYS,default=30"`
}

type downloadConfig struct {
	Env           efgsutils.Environment
	Client        *http.Client
	URL           *urlutils.URL
	NBTLSPair     *efgsutils.X509KeyPair
	HaidMappings  map[string]string
	PublishConfig *publishConfig
}

func loadUploadConfig(ctx context.Context) (*uploadConfig, error) {
	logger := logging.FromContext(ctx).Named("efgs.loadUploadConfig")

	efgsEnv := efgsutils.GetEfgsEnvironmentOrFail()

	url := efgsutils.GetEfgsURLOrFail(efgsEnv)
	url.Path = "diagnosiskeys/upload"

	var err error
	config := uploadConfig{
		URL:      url,
		Env:      efgsEnv,
		Database: &efgsdatabase.Database,
	}

	config.Env = efgsEnv

	config.NBTLSPair, err = efgsutils.LoadX509KeyPair(ctx, efgsEnv, efgsutils.NBTLS)
	if err != nil {
		logger.Debug("Error loading authentication certificate")
		return nil, err
	}

	config.Client, err = efgsutils.NewEFGSClient(ctx, config.NBTLSPair)
	if err != nil {
		logger.Debug("Could not create EFGS client")
		return nil, err
	}

	maxBatchSize, isSet := os.LookupEnv("EFGS_UPLOAD_BATCH_SIZE")
	if !isSet {
		return nil, fmt.Errorf("EFGS_UPLOAD_BATCH_SIZE must be set")
	}

	batchSizeLimit, err := strconv.Atoi(maxBatchSize)
	if err != nil {
		return nil, fmt.Errorf("Error converting batch size to int: %s", err)
	}

	config.BatchSizeLimit = batchSizeLimit

	return &config, nil
}

func loadPublishConfig(ctx context.Context) (*publishConfig, error) {
	logger := logging.FromContext(ctx).Named("efgs.loadPublishConfig")

	var config publishConfig
	if err := envconfig.Process(ctx, &config); err != nil {
		return nil, err
	}

	keyServerConfig, err := utils.LoadKeyServerConfig(ctx)
	if err != nil {
		logger.Fatalf("Could not load key server config: %v", err)
		return nil, err
	}

	verificationServerConfig, err := utils.LoadVerificationServerConfig(ctx)
	if err != nil {
		logger.Fatalf("Could not load verification server config: %v", err)
		return nil, err
	}

	config.KeyServer = keyServerConfig
	config.VerificationServer = verificationServerConfig
	config.Client = &http.Client{}

	return &config, nil
}

func loadDownloadConfig(ctx context.Context) (*downloadConfig, error) {
	logger := logging.FromContext(ctx).Named("efgs.download-batch.loadDownloadConfig")

	env := efgsutils.GetEfgsEnvironmentOrFail()

	var config downloadConfig

	secretsClient := secrets.Client{}
	bytes, err := secretsClient.Get("efgs-haid-mappings")
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(bytes, &config.HaidMappings); err != nil {
		return nil, err
	}

	nbtlsPair, err := efgsutils.LoadX509KeyPair(ctx, env, efgsutils.NBTLS)
	if err != nil {
		logger.Debugf("Error loading authentication certificate: %v", err)
		return nil, err
	}

	url := efgsutils.GetEfgsURLOrFail(env)

	client, err := efgsutils.NewEFGSClient(ctx, nbtlsPair)
	if err != nil {
		logger.Debugf("Could not create EFGS client: %v", err)
		return nil, err
	}

	config.Env = env
	config.URL = url
	config.NBTLSPair = nbtlsPair
	config.Client = client

	publishConfig, err := loadPublishConfig(ctx)
	if err != nil {
		logger.Debugf("Could not load publish config: %v", err)
		return nil, err
	}

	config.PublishConfig = publishConfig

	return &config, nil
}
