module github.com/covid19cz/erouska-backend

go 1.13

require (
	github.com/google/go-cmp v0.3.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sethvargo/go-signalcontext v0.1.0
	github.com/stretchr/testify v1.5.1 // indirect
	go.opencensus.io v0.22.4
	go.uber.org/zap v1.15.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/tools v0.0.0-20200723000907-a7c6fd066f6d // indirect
)

replace github.com/covid19cz/erouska-backend/pkg/httpserver v0.0.0 => ./pkg/httpserver

replace github.com/covid19cz/erouska-backend/internal/hello v0.0.0 => ../internal/hello
