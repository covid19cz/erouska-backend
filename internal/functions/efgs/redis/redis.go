package redis

import (
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/logging"
	redisclient "github.com/go-redis/redis/v8"
	"os"
	"sync"
	"time"
)

var redisClient *Connection
var ctx context.Context

type lazyConnection func() *redisclient.Client

//Connection Contains lazy Redis connection
type Connection struct {
	inner lazyConnection
}

func init() {
	ctx = context.Background()

	connect := func() *redisclient.Client {
		logger := logging.FromContext(ctx).Named("efgs.redis.connect")

		logger.Debug("Connecting to EFGS Redis")

		addr, ok := os.LookupEnv("EFGS_REDIS_ADDR")
		if !ok {
			panic("EFGS_REDIS_ADDR env missing")
		}

		client := redisclient.NewClient(&redisclient.Options{
			Addr: addr,
			DB:   0,
		})

		_, err := client.Ping(ctx).Result()
		if err != nil {
			err := fmt.Errorf("Connection to Redis failed:%v", err)
			logger.Error(err)
			panic(err)
		}

		logger.Debugf("Connected to EFGS Redis at %v", addr)

		return client
	}

	var conn *redisclient.Client
	var once sync.Once

	initInner := func() *redisclient.Client {
		once.Do(func() {
			conn = connect()
		})
		return conn
	}

	redisClient = &Connection{
		inner: initInner,
	}
}

//Client Redis client abstraction
type Client interface {
	Get(key string) (string, error)
	Set(key string, value interface{}, ttl time.Duration) error
}

//ClientImpl Real Redis client
type ClientImpl struct{}

//Get Get value from Redis
func (r ClientImpl) Get(key string) (string, error) {
	return redisClient.inner().Get(ctx, key).Result()
}

//Set Set value to Redis. TLL value 0 means forever.
func (r ClientImpl) Set(key string, value interface{}, ttl time.Duration) error {
	return redisClient.inner().Set(ctx, key, value, ttl).Err()
}
