package efgs

import (
	"fmt"
	efgsdatabase "github.com/covid19cz/erouska-backend/internal/functions/efgs/database"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"net/http"
	"os"
	"strconv"
	"time"
)

//CleanupDatabase Remove old (more than EFGS_EXPOSURE_KEYS_EXPIRATION days) keys from database.
func CleanupDatabase(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx).Named("efgs.CleanupDatabase")
	keyExpiration, isSet := os.LookupEnv("EFGS_EXPOSURE_KEYS_EXPIRATION")
	if !isSet {
		err := fmt.Errorf("EFGS_EXPOSURE_KEYS_EXPIRATION must be set")
		logger.Error(err)
		sendErrorResponse(w, err)
		return
	}

	keyValidityDays, err := strconv.Atoi(keyExpiration)
	if err != nil {
		logger.Errorf("Error converting key validity days to int: %s", err)
		sendErrorResponse(w, err)
		return
	}
	dateFrom := time.Now().AddDate(0, 0, -keyValidityDays).Format("2006-01-02")

	if err := efgsdatabase.Database.RemoveOldKeys(dateFrom); err != nil {
		logger.Errorf("Cleanup failed: %s", err)
		sendErrorResponse(w, err)
		return
	}

	logger.Infof("Database is cleared")
}
