# erouska 2.0 - backend
----
![tests](https://github.com/covid19cz/erouska-backend/workflows/tests/badge.svg)

## Quickstart / running locally
```
git clone https://github.com/covid19cz/erouska-backend
cd erouska-backend
make dep
make build
./bin/erouska &
curl -X POST localhost:8081/ -d@examples/request.json
Hello, Jaroslav!%
```

## Deployment
```
PROJECT_ID=<YOUR_GCP_PROJECT> ./scripts/deploy
```

## Environment variables
```
# for ci/testing set FIREBASE_URL to "NOOP"
export FIREBASE_URL=NOOP
```
