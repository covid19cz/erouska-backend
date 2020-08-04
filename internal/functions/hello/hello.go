// Package hello provides some logic
package hello

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
)

func sayHello(name string) string {

	if name == "" {
		return "Hello, world!"
	}

	return fmt.Sprintf("Hello, %s!", name)
}

// Hello says hello to given name
func Hello(w http.ResponseWriter, r *http.Request) {
	var d struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		fmt.Fprint(w, sayHello(""))
		return
	}
	if d.Name == "" {
		fmt.Fprint(w, sayHello(""))
		return
	}

	fmt.Fprint(w, sayHello(html.EscapeString(d.Name)))
}
