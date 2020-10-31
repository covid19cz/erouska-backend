package database

import (
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/proxy"
	efgsapi "github.com/covid19cz/erouska-backend/internal/functions/efgs/api"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"net"
	"sync"
	"time"
)

//Database Singleton connection to EFGS database.
var Database Connection

type lazyConnection func() *pg.DB

//Connection Contains database connection pool.
type Connection struct {
	inner lazyConnection
}

//Create new (lazy) database connection pool. Credentials must be specified in secret manager.
func init() {
	connectToDatabase := func() *pg.DB {
		ctx := context.Background()
		logger := logging.FromContext(ctx).Named("efgs.database.connectToDatabase")
		secretsClient := secrets.Client{}

		logger.Debug("Initializing EFGS database connection")

		efgsDatabaseName, err := secretsClient.Get("efgs-database-name")
		if err != nil {
			panic(fmt.Sprintf("Connection to secret manager failed: %s", err))
		}
		efgsDatabasePassword, err := secretsClient.Get("efgs-database-password")
		if err != nil {
			panic(fmt.Sprintf("Connection to secret manager failed: %s", err))
		}
		efgsDatabaseUser, err := secretsClient.Get("efgs-database-login")
		if err != nil {
			panic(fmt.Sprintf("Connection to secret manager failed: %s", err))
		}
		efgsDatabaseConnectionName, err := secretsClient.Get("efgs-database-connection-name")
		if err != nil {
			panic(fmt.Sprintf("Connection to secret manager failed: %s", err))
		}

		connection := pg.Connect(&pg.Options{
			Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return proxy.Dial(string(efgsDatabaseConnectionName))
			},
			User:     string(efgsDatabaseUser),
			Password: string(efgsDatabasePassword),
			Database: string(efgsDatabaseName),
		})

		if err := createSchema(connection); err != nil {
			panic(fmt.Sprintf("Error while creating DB schema: %s", err))
		}

		logger.Debug("EFGS database initialized")

		return connection
	}

	// run the above just once, but lazy:

	var conn *pg.DB
	var once sync.Once

	initInner := func() *pg.DB {
		once.Do(func() {
			conn = connectToDatabase()
		})
		return conn
	}

	Database = Connection{
		inner: initInner,
	}
}

//PersistDiagnosisKeys Save array of DiagnosisKey to database
func (db Connection) PersistDiagnosisKeys(keys []*efgsapi.DiagnosisKey) error {
	connection := db.inner().Conn()
	defer connection.Close()

	_, err := connection.Model(&keys).Insert()
	if err != nil {
		return err
	}

	return nil
}

//GetDiagnosisKeys Get keys from the DB that are not yet in EFGS and are not older than dateFrom.
func (db Connection) GetDiagnosisKeys(dateFrom time.Time) ([]*efgsapi.DiagnosisKeyWrapper, error) {
	connection := db.inner().Conn()
	defer connection.Close()

	var keys []*efgsapi.DiagnosisKeyWrapper
	if err := connection.Model(&keys).Where("created_at >= ?", dateFrom.Format("2006-01-02")).Select(); err != nil {
		return nil, err
	}

	return keys, nil
}

//RemoveDiagnosisKeys Remove array of DiagnosisKeyWrapper from the DB.
func (db Connection) RemoveDiagnosisKeys(keys []*efgsapi.DiagnosisKeyWrapper) error {
	connection := db.inner().Conn()
	defer connection.Close()

	_, err := connection.Model(&keys).WherePK().Delete()
	if err != nil {
		return err
	}

	return nil
}

//UpdateKeys Updates key records in the DB.
func (db Connection) UpdateKeys(keys []*efgsapi.DiagnosisKeyWrapper) error {
	connection := db.inner().Conn()
	defer connection.Close()

	_, err := connection.Model(&keys).WherePK().Update()
	if err != nil {
		return err
	}

	return nil
}

//RemoveInvalidKeys Remove N times refused keys from the DB
func (db Connection) RemoveInvalidKeys(keys []*efgsapi.DiagnosisKeyWrapper) error {
	connection := db.inner().Conn()
	defer connection.Close()

	_, err := connection.Model(&keys).Where("retries >= 2").Delete()
	if err != nil {
		return err
	}

	return nil
}

func createSchema(cpool *pg.DB) error {
	connection := cpool.Conn()
	defer connection.Close()

	models := []interface{}{
		(*efgsapi.DiagnosisKeyWrapper)(nil),
	}

	for _, model := range models {
		err := connection.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
