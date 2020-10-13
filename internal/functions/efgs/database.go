package efgs

import (
	"context"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"net"

	"github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/proxy"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

//Singleton database struct.
var database Database

//Database Contains database connection pool.
type Database struct {
	connection *pg.DB
}

//Create new database connection pool. Credentials must be specified in secret manager.
func init() {
	ctx := context.Background()
	logger := logging.FromContext(ctx)
	secretsClient := secrets.Client{}

	efgsDatabaseName, err := secretsClient.Get("efgs-database-name")
	if err != nil {
		logger.Fatalf("Connection to secret manager failed: %s", err)
		return
	}
	efgsDatabasePassword, err := secretsClient.Get("efgs-database-password")
	if err != nil {
		logger.Fatalf("Connection to secret manager failed: %s", err)
		return
	}
	efgsDatabaseUser, err := secretsClient.Get("efgs-database-login")
	if err != nil {
		logger.Fatalf("Connection to secret manager failed: %s", err)
		return
	}
	efgsDatabaseConnectionName, err := secretsClient.Get("efgs-database-connection-name")
	if err != nil {
		logger.Fatalf("Connection to secret manager failed: %s", err)
		return
	}

	database.connection = pg.Connect(&pg.Options{
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return proxy.Dial(string(efgsDatabaseConnectionName))
		},
		User:     string(efgsDatabaseUser),
		Password: string(efgsDatabasePassword),
		Database: string(efgsDatabaseName),
	})

	if err := database.createSchema(); err != nil {
		logger.Fatalf("Error while creating DB schema: %s", err)
		return
	}
}

//PersistDiagnosisKeys Save array of DiagnosisKey to database
func (db Database) PersistDiagnosisKeys(keys []*DiagnosisKey) error {
	connection := db.connection.Conn()
	defer connection.Close()

	_, err := connection.Model(keys).Insert()
	if err != nil {
		return err
	}

	return nil
}

//GetDiagnosisKeys Get keys from database that are not yet in EFGS and are older than dateTo and newer than dateFrom.
func (db Database) GetDiagnosisKeys(dateFrom string) ([]*DiagnosisKeyWrapper, error) {
	connection := db.connection.Conn()
	defer connection.Close()

	var keys []*DiagnosisKeyWrapper
	if err := connection.Model(&keys).Where("created_at >= ?", dateFrom).Select(); err != nil {
		return nil, err
	}
	return keys, nil
}

//RemoveDiagnosisKey Remove array of DiagnosisKeyWrapper from database.
func (db Database) RemoveDiagnosisKey(keys []*DiagnosisKeyWrapper) error {
	connection := db.connection.Conn()
	defer connection.Close()

	_, err := connection.Model(&keys).WherePK().Delete()
	if err != nil {
		return err
	}

	return nil
}

func (db Database) createSchema() error {
	connection := db.connection.Conn()
	defer connection.Close()

	models := []interface{}{
		(*DiagnosisKeyWrapper)(nil),
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
