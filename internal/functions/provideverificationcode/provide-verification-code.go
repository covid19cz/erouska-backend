package provideverificationcode

import (
	"context"
	"fmt"
	rpccode "google.golang.org/genproto/googleapis/rpc/code"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/avast/retry-go"
	"github.com/covid19cz/erouska-backend/internal/constants"
	"github.com/covid19cz/erouska-backend/internal/firebase/structs"
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/covid19cz/erouska-backend/internal/store"
	"github.com/covid19cz/erouska-backend/internal/utils"
	"github.com/covid19cz/erouska-backend/internal/utils/errors"
	httputils "github.com/covid19cz/erouska-backend/internal/utils/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const needsRetry = "needs_retry"

type verificationCodeRequest struct {
	VsMetadata map[string]interface{} `json:"vsMetadata" validate:"required"`
}

type verificationCodeResponse struct {
	VerificationCode string `json:"verificationCode"`
}

//ProvideVerificationCode Creates and returns new verification code, saving it into DB.
func ProvideVerificationCode(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	logger := logging.FromContext(ctx)
	client := store.Client{}

	var request verificationCodeRequest

	if !httputils.DecodeJSONOrReportError(w, r, &request) {
		return
	}

	logger.Debugf("Handling ProvideVerificationCode request: %+v", request)

	metadata := structs.VerificationCodeMetadata{
		VsMetadata: request.VsMetadata,
		IssuedAt:   utils.GetTimeNow().Unix(),
	}

	vc, err := newVerificationCode(ctx, client, utils.GenerateVerificationCode, metadata)

	if err != nil {
		logger.Warnf("Cannot handle request due to unknown error: %+v", err.Error())
		httputils.SendErrorResponse(w, r, rpccode.Code_INTERNAL, "Unknown error")
		return
	}

	response := verificationCodeResponse{
		VerificationCode: vc,
	}

	httputils.SendResponse(w, r, response)
}

func newVerificationCode(ctx context.Context, store store.Storer, generateVC func() string, metadata structs.VerificationCodeMetadata) (string, error) {
	logger := logging.FromContext(ctx)

	var vc string

	err := retry.Do(
		func() error {
			vc = generateVC()
			var doc = store.Doc(constants.CollectionVerificationCodes, vc)

			logger.Debugf("Trying VC: %v", vc)

			return store.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
				_, err := tx.Get(doc)

				if err == nil {
					// doc found, need retry
					return &errors.CustomError{Msg: needsRetry}
				}

				if status.Code(err) != codes.NotFound {
					return fmt.Errorf("Error while querying Firestore: %v", err)
				}
				// not found, great!

				logger.Infof("Generated new VC %v, saving metadata %+v", vc, metadata)

				return tx.Set(doc, metadata)
			})
		},
		retry.RetryIf(func(err error) bool {
			return err.Error() == needsRetry
		}),
	)

	return vc, err
}
