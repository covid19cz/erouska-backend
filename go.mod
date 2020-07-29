module github.com/covid19cz/erouska-backend

go 1.13

require (
	cloud.google.com/go/firestore v1.2.0
	firebase.google.com/go v3.13.0+incompatible
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/golang/gddo v0.0.0-20200715224205-051695c33a3f
	github.com/google/go-cmp v0.4.0
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sethvargo/go-signalcontext v0.1.0
	github.com/stretchr/testify v1.5.1
	go.opencensus.io v0.22.4 // indirect
	go.uber.org/zap v1.15.0
	golang.org/x/tools v0.0.0-20200731060945-b5fad4ed8dd6 // indirect
	google.golang.org/api v0.29.0 // indirect
	google.golang.org/grpc v1.28.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
)

replace github.com/covid19cz/erouska-backend/internal/httpserver v0.0.0 => ./pkg/httpserver

replace github.com/covid19cz/erouska-backend/internal/hello v0.0.0 => ../internal/hello
