# erouska-backend
----
![tests](https://github.com/covid19cz/erouska-backend/workflows/tests/badge.svg)

## Quickstart
```
git clone https://github.com/covid19cz/erouska-backend
cd erouska-backend
make dep
make build
./bin/erouska &
curl -X GET localhost:8081/ -d@examples/request.json
Hello, Jaroslav!%
```

## Deployment
```
PROJECT_ID=<YOUR_PROJECT> ./scripts/deploy
```
