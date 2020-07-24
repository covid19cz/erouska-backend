// Package hello provides some logic
package hello

import (
	"fmt"
)

// SayHello says hello, obviously, duh
func SayHello(name string) string {

	if name == "" {
		return "Hello, world!"
	}

	return fmt.Sprintf("Hello, %s!", name)
}
