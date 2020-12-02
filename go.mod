module github.com/covid19cz/erouska-backend

go 1.13

require (
	cloud.google.com/go v0.71.0
	cloud.google.com/go/firestore v1.3.0
	cloud.google.com/go/pubsub v1.3.1
	firebase.google.com/go v3.13.0+incompatible
	github.com/GoogleCloudPlatform/cloudsql-proxy v1.18.0
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-pg/pg/v10 v10.7.0
	github.com/go-redis/redis/v8 v8.4.0
	github.com/go-redsync/redsync/v4 v4.0.3
	github.com/golang/gddo v0.0.0-20200715224205-051695c33a3f
	github.com/golang/protobuf v1.4.3
	github.com/google/exposure-notifications-server v0.16.0
	github.com/google/exposure-notifications-verification-server v0.16.0
	github.com/google/go-cmp v0.5.3
	github.com/hashicorp/go-retryablehttp v0.6.8
	github.com/leanovate/gopter v0.2.9
	github.com/sethvargo/go-envconfig v0.3.2
	github.com/sethvargo/go-signalcontext v0.1.0
	github.com/stretchr/stew v0.0.0-20130812190256-80ef0842b48b
	github.com/stretchr/testify v1.6.1
	go.mozilla.org/pkcs7 v0.0.0-20200128120323-432b2356ecb1
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	google.golang.org/api v0.35.0
	google.golang.org/genproto v0.0.0-20201110150050-8816d57aaa9a
	google.golang.org/grpc v1.33.2
	google.golang.org/protobuf v1.25.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0
)

replace github.com/covid19cz/erouska-backend/internal/httpserver v0.0.0 => ./pkg/httpserver

replace github.com/covid19cz/erouska-backend/internal/hello v0.0.0 => ../internal/hello
