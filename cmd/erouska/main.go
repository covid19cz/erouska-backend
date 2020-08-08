package main

import (
	"github.com/covid19cz/erouska-backend/internal/logging"
	"github.com/sethvargo/go-signalcontext"
)

func main() {

	ctx, done := signalcontext.OnInterrupt()
	defer done()

	_ = logging.FromContext(ctx)

}
