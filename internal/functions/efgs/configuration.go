package efgs

import (
	"context"
	"encoding/json"
	"fmt"
	efgsdatabase "github.com/covid19cz/erouska-backend/internal/functions/efgs/database"
	"github.com/covid19cz/erouska-backend/internal/functions/efgs/redis"
	"github.com/covid19cz/erouska-backend/internal/functions/efgs/redismutex"
	efgsutils "github.com/covid19cz/erouska-backend/internal/functions/efgs/utils"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/pubsub"
	"github.com/covid19cz/erouska-backend/internal/realtimedb"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"github.com/covid19cz/erouska-backend/internal/utils"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"github.com/sethvargo/go-envconfig"
	"net/http"
	urlutils "net/url"
	"os"
	"strconv"
)

type uploadConfig struct {
	URL              *urlutils.URL
	Env              efgsutils.Environment
	NBTLSPair        *efgsutils.X509KeyPair
	Client           *http.Client
	Database         *efgsdatabase.Connection
	RealtimeDBClient *realtimedb.Client
	BatchSizeLimit   int
	KeyValidityDays  int
	BatchTag         string
}

type publishConfig struct {
	VerificationServer *utils.VerificationServerConfig
	KeyServer          *utils.KeyServerConfig
	Client             *http.Client
	MaxKeysOnPublish   int `env:"MAX_KEYS_ON_PUBLISH,default=30"`
}

type downloadConfig struct {
	Env                               efgsutils.Environment
	Client                            *http.Client
	URL                               *urlutils.URL
	NBTLSPair                         *efgsutils.X509KeyPair
	HaidMappings                      map[string]string
	PubSubClient                      pubsub.EventPublisher
	RedisClient                       redis.Client
	MutexManager                      redismutex.MutexManager
	RealtimeDBClient                  *realtimedb.Client
	MaxKeysOnPublish                  int `env:"MAX_KEYS_ON_PUBLISH,default=30"`
	MaxIntervalAge                    int `env:"MAX_INTERVAL_AGE_ON_PUBLISH,default=15"`
	MaxSameStartIntervalKeys          int `env:"MAX_SAME_START_INTERVAL_KEYS,default=15"`
	MaxDownloadYesterdaysKeysPartSize int `env:"MAX_YESTERDAYS_KEYS_PART_SIZE"`
}

func loadUploadConfig(ctx context.Context) (*uploadConfig, error) {
	logger := logging.FromContext(ctx).Named("efgs.loadUploadConfig")

	efgsEnv := efgsutils.GetEfgsEnvironmentOrFail()

	url := efgsutils.GetEfgsURLOrFail(efgsEnv)
	url.Path = "diagnosiskeys/upload"

	var err error
	config := uploadConfig{
		URL:              url,
		Env:              efgsEnv,
		Database:         &efgsdatabase.Database,
		RealtimeDBClient: &realtimedb.Client{},
	}

	config.Env = efgsEnv

	config.NBTLSPair, err = efgsutils.LoadX509KeyPair(ctx, efgsEnv, efgsutils.NBTLS)
	if err != nil {
		logger.Debug("Error loading authentication certificate")
		return nil, err
	}

	efgsClient, err := efgsutils.NewEFGSClient(ctx, config.NBTLSPair)
	if err != nil {
		logger.Debug("Could not create EFGS client")
		return nil, err
	}

	clientLogger := logging.FromContext(ctx).Named("efgs.efgs-client")
	config.Client = httputils.NewThrottlingAwareClient(efgsClient, clientLogger.Debugf)

	maxBatchSize, isSet := os.LookupEnv("EFGS_UPLOAD_BATCH_SIZE")
	if !isSet {
		return nil, fmt.Errorf("EFGS_UPLOAD_BATCH_SIZE must be set")
	}

	batchSizeLimit, err := strconv.Atoi(maxBatchSize)
	if err != nil {
		return nil, fmt.Errorf("Error converting batch size to int: %s", err)
	}

	config.BatchSizeLimit = batchSizeLimit

	keyExpiration, isSet := os.LookupEnv("EFGS_EXPOSURE_KEYS_EXPIRATION")
	if !isSet {
		return nil, fmt.Errorf("EFGS_EXPOSURE_KEYS_EXPIRATION must be set")
	}

	keyValidityDays, err := strconv.Atoi(keyExpiration)
	if err != nil {
		return nil, fmt.Errorf("Error converting key expiration to int: %s", err)
	}

	config.KeyValidityDays = keyValidityDays

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

	clientLogger := logging.FromContext(ctx).Named("efgs.publish-client")
	config.Client = httputils.NewThrottlingAwareClient(&http.Client{}, clientLogger.Debugf)

	return &config, nil
}

func loadDownloadConfig(ctx context.Context) (*downloadConfig, error) {
	logger := logging.FromContext(ctx).Named("efgs.download-batch.loadDownloadConfig")

	env := efgsutils.GetEfgsEnvironmentOrFail()

	var config downloadConfig
	if err := envconfig.Process(ctx, &config); err != nil {
		return nil, err
	}

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

	efgsClient, err := efgsutils.NewEFGSClient(ctx, nbtlsPair)
	if err != nil {
		logger.Debugf("Could not create EFGS client: %v", err)
		return nil, err
	}

	clientLogger := logging.FromContext(ctx).Named("efgs.efgs-client")
	config.Client = httputils.NewThrottlingAwareClient(efgsClient, clientLogger.Debugf)

	config.Env = env
	config.URL = url
	config.NBTLSPair = nbtlsPair
	config.PubSubClient = pubsub.Client{}
	config.MutexManager = redismutex.ClientImpl{}
	config.RedisClient = redis.ClientImpl{}
	config.RealtimeDBClient = &realtimedb.Client{}

	return &config, nil
}
