module github.com/covid19cz/erouska-backend

go 1.13

require (
	cloud.google.com/go/firestore v1.2.0
	firebase.google.com/go v3.13.0+incompatible
	github.com/google/go-cmp v0.4.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sethvargo/go-signalcontext v0.1.0
	github.com/stretchr/testify v1.5.1 // indirect
	go.opencensus.io v0.22.4 // indirect
	go.uber.org/zap v1.15.0
	golang.org/x/tools v0.0.0-20200723000907-a7c6fd066f6d // indirect
	google.golang.org/api v0.29.0 // indirect
)

replace github.com/covid19cz/erouska-backend/pkg/httpserver v0.0.0 => ./pkg/httpserver

replace github.com/covid19cz/erouska-backend/pkg/firestore v0.0.0 => ./pkg/firestore

replace github.com/covid19cz/erouska-backend/internal/hello v0.0.0 => ../internal/hello
