package auth

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	customToken     = flag.String("ct", "", "custom token")
	projectAPIToken = flag.String("at", "", "project API token - if not provided, value from env variable PROJECTAPIKEY is used")
)

type verificationResponse struct {
	IDToken string `json:"idToken"`
}

// getIDToken verify customToken and return idToken
func getIDToken(customToken string, projectAPIKey string) string {
	requestBody, err := json.Marshal(map[string]string{
		"token":             customToken,
		"returnSecureToken": "true",
	})

	if err != nil {
		log.Fatalln(err)
		return ""
	}

	resp, err := http.Post(
		"https://www.googleapis.com/identitytoolkit/v3/relyingparty/verifyCustomToken?key="+projectAPIKey,
		"application/json",
		bytes.NewBuffer(requestBody))

	if err != nil {
		log.Fatalf("Verifiaction request err: %s\n", err)
		return ""
	}

	var r verificationResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error while reading response body: %s\n", err)
		return ""
	}

	if err := json.Unmarshal(body, &r); err != nil {
		log.Fatalf("Response mismatch: %s\n", err)
		return ""
	}

	return r.IDToken
}

func printIDToken(customToken string, projectAPIKey string) {
	token := getIDToken(customToken, projectAPIKey)
	if token == "" {
		flag.PrintDefaults()
		return
	}
	fmt.Println(token)
}

func main() {
	flag.Parse()

	if *customToken == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *projectAPIToken == "" {
		*projectAPIToken = os.Getenv("PROJECTAPIKEY")
	}

	printIDToken(*customToken, *projectAPIToken)
}
