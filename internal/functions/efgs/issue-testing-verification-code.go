package efgs

import (
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/secrets"
	"golang.org/x/net/context"
	"net/http"
	"os"
)

//IssueTestingVerificationCode Issues new VC for publishing keys.
func IssueTestingVerificationCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := logging.FromContext(ctx).Named("efgs.IssueTestingVerificationCode")

	_, enabled := os.LookupEnv("EFGS_TESTING_VC_ISSUE_ENABLED")
	if !enabled {
		logger.Error("Issuing testing verification codes is not enabled!")
		http.Error(w, "Not enabled", 503)
		return
	}

	secretClient := secrets.Client{}

	apikey, err := secretClient.Get("efgs-testing-vc-issue-apikey")
	if err != nil {
		logger.Warnf("Could not obtain api key: %v", err)
		http.Error(w, "Could not obtain api key", 500)
		return
	}

	publishConfig, err := loadPublishConfig(ctx)
	if err != nil {
		logger.Warnf("Could not load publish config: %v", err)
		http.Error(w, "Could not load config", 500)
		return
	}

	httpStatus, httpBody := issueTestingVerificationCode(ctx, r, publishConfig, string(apikey))

	http.Error(w, httpBody, httpStatus)
}

func issueTestingVerificationCode(ctx context.Context, r *http.Request, config *publishConfig, apikey string) (int, string) {
	logger := logging.FromContext(ctx).Named("efgs.issueTestingVerificationCode")

	providedAPIKeys := r.URL.Query()["apikey"]
	if len(providedAPIKeys) != 1 || providedAPIKeys[0] != apikey {
		return 401, "Bad api key"
	}

	// authenticated, go ahead

	vc, err := requestNewVC(ctx, config)
	if err != nil {
		logger.Warnf("Could not load publish config: %v", err)
		return 500, "Could not get VC"
	}

	return 200, vc
}
