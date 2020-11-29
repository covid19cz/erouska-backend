package redismutex

import (
	"context"
	"fmt"
	"github.com/covid19cz/erouska-backend/internal/logging"
	redisclient "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"os"
	"sync"
	"time"
)

var rs *Connection
var ctx context.Context

type lazyConnection func() *redsync.Redsync

//Connection Contains lazy Redsync connection
type Connection struct {
	inner lazyConnection
}

func init() {
	ctx = context.Background()

	connect := func() *redsync.Redsync {
		logger := logging.FromContext(ctx).Named("efgs.redis-mutex.connect")

		logger.Debug("Connecting to EFGS Redis")

		addr, ok := os.LookupEnv("EFGS_REDIS_ADDR")
		if !ok {
			panic("EFGS_REDIS_ADDR env missing")
		}

		client := redisclient.NewClient(&redisclient.Options{
			Addr: addr,
			DB:   1, // here it differs from normal Redis client!
		})

		_, err := client.Ping(ctx).Result()
		if err != nil {
			err := fmt.Errorf("Connection to Redis failed:%v", err)
			logger.Error(err)
			panic(err)
		}

		logger.Debugf("Connected to EFGS Redis at %v", addr)

		return redsync.New(goredis.NewPool(client))
	}

	var conn *redsync.Redsync
	var once sync.Once

	initInner := func() *redsync.Redsync {
		once.Do(func() {
			conn = connect()
		})
		return conn
	}

	rs = &Connection{
		inner: initInner,
	}
}

//MutexManager Mutex manager over Redis
type MutexManager interface {
	Lock(name string) (*redsync.Mutex, error)
}

//ClientImpl Real Redis mutex client
type ClientImpl struct{}

//Lock Creates locked mutex
func (r ClientImpl) Lock(name string) (*redsync.Mutex, error) {
	logger := logging.FromContext(ctx).Named("efgs.redis-mutex.Lock")

	mutex := rs.inner().NewMutex(name, redsync.WithExpiry(time.Hour)) // expiration of 1 hour is just to be sure

	logger.Debugf("Trying to acquire '%v' exclusive lock", name)

	err := mutex.Lock()
	if err != nil {
		return nil, err
	}

	return mutex, nil
}
