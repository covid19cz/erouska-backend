package hello

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSayHello(t *testing.T) {

	tables := []struct {
		x string
		y string
	}{
		{"marek", "Hello, marek!"},
		{"", "Hello, world!"},
		{"Jenda", "Hello, Jenda!"},
	}

	for _, table := range tables {
		greeting := SayHello(table.x)

		diff := cmp.Diff(greeting, table.y)
		if diff != "" {
			t.Fatalf("greeting mismatch (-want +got):\n%v", diff)
		}
	}
}
