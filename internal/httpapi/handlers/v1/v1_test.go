package v1

import (
	"errors"
	"strings"
	"testing"
)

func TestReadAllLimited(t *testing.T) {
	got, err := readAllLimited(strings.NewReader("abc"), 3)
	if err != nil {
		t.Fatalf("readAllLimited returned error: %v", err)
	}
	if string(got) != "abc" {
		t.Fatalf("readAllLimited = %q", string(got))
	}

	_, err = readAllLimited(strings.NewReader("abcd"), 3)
	if !errors.Is(err, errBodyTooLarge) {
		t.Fatalf("readAllLimited oversized error = %v", err)
	}
}
