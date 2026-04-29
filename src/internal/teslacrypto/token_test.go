package teslacrypto

import "testing"

func TestDecryptAccessTokenReturnsEmptyForMalformedCiphertext(t *testing.T) {
	if got := DecryptAccessToken("bad", "key"); got != "" {
		t.Fatalf("expected empty token for malformed ciphertext, got %q", got)
	}
}

func TestGetCarRegionAPIDefaultsForMalformedJWT(t *testing.T) {
	if got := GetCarRegionAPI("header.payload.signature"); got != GlobalAPI {
		t.Fatalf("expected malformed token to default to global API, got %s", got)
	}
}
