module github.com/covid19cz/erouska-backend

go 1.13

require (
	cloud.google.com/go v0.65.0
	cloud.google.com/go/firestore v1.2.0
	cloud.google.com/go/pubsub v1.3.1
	firebase.google.com/go v3.13.0+incompatible
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/golang/gddo v0.0.0-20200715224205-051695c33a3f
	github.com/google/exposure-notifications-server v0.7.0
	github.com/google/go-cmp v0.5.2
	github.com/sethvargo/go-envconfig v0.3.1
	github.com/sethvargo/go-signalcontext v0.1.0
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	google.golang.org/genproto v0.0.0-20200901141002-b3bf27a9dbd1
	google.golang.org/grpc v1.31.1
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
)

replace github.com/covid19cz/erouska-backend/internal/httpserver v0.0.0 => ./pkg/httpserver

replace github.com/covid19cz/erouska-backend/internal/hello v0.0.0 => ../internal/hello
