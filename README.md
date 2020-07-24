# erouska-backend
----

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

## Deployment to Google Cloud Functions
```
gcloud alpha functions  deploy HelloHTTP --runtime go113 --trigger-http --memory=128 --allow-unauthenticated --region=europe-west1
```
