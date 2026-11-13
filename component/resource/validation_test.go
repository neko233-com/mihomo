package resource

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBytesOnlyParses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provider")
	want := errors.New("bad provider")
	f := NewFetcher("test", 0, NewFileVehicle(path), nil, func(b []byte) (string, error) {
		if string(b) != "valid" {
			return "", want
		}
		return "parsed", nil
	}, func(string) { t.Fatal("validation updated the provider") })
	defer f.Close()
	if err := f.ValidateBytes([]byte("valid")); err != nil {
		t.Fatal(err)
	}
	if err := f.ValidateBytes([]byte("invalid")); !errors.Is(err, want) {
		t.Fatalf("invalid input: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("validation wrote a file: %v", err)
	}
}
