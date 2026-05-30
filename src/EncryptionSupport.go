package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type CarRegionAPI string

const (
	ChinaAPI  CarRegionAPI = "China"
	GlobalAPI CarRegionAPI = "Global"
)

// decryptAccessToken decrypts a TeslaMate-encrypted token blob from
// `private.tokens`. Layout (bytes — NOT runes; the previous []rune indexing
// silently corrupted blobs whose first byte was ≥ 0x80):
//
//	[type(1)] [keyLen(1)] [keyTag(keyLen)] [iv(12)] [ciphertag(16)] [ciphertext(n)]
//
// AAD is the fixed string "AES256GCM". Errors are returned (not panicked)
// so a malformed blob or wrong ENCRYPTION_KEY surfaces as 500, not a
// crashed gin worker.
func decryptAccessToken(data, encryptionKey string) (string, error) {
	raw := []byte(data)

	h := sha256.New()
	h.Write([]byte(encryptionKey))
	key := h.Sum(nil)
	if gin.IsDebugging() {
		log.Printf("[debug] decryptAccessToken - Key: %x", key)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("decryptAccessToken: aes.NewCipher: %w", err)
	}

	if len(raw) < 2 {
		return "", errors.New("decryptAccessToken: encrypted blob too short for header")
	}
	keyType := int(raw[0])
	keyLen := int(raw[1])

	// 2(header) + keyLen + 12(iv) + 16(tag) is the minimum frame; ciphertext
	// is allowed to be empty (an empty token would still parse) but a frame
	// shorter than the minimum is malformed.
	minLen := 2 + keyLen + 12 + 16
	if len(raw) < minLen {
		return "", fmt.Errorf("decryptAccessToken: encrypted blob length %d shorter than expected %d", len(raw), minLen)
	}

	keyTag := raw[2 : 2+keyLen]
	nonce := raw[2+keyLen : 2+keyLen+12]
	ciphertag := raw[2+keyLen+12 : 2+keyLen+12+16]
	ciphertext := raw[2+keyLen+12+16:]

	if gin.IsDebugging() {
		log.Printf("[debug] decryptAccessToken - Type: %d", keyType)
		log.Printf("[debug] decryptAccessToken - Length: %d", keyLen)
		log.Printf("[debug] decryptAccessToken - Key Tag: %s", keyTag)
		log.Printf("[debug] decryptAccessToken - IV (hex): %x", nonce)
		log.Printf("[debug] decryptAccessToken - Ciphertag (hex): %x", ciphertag)
	}

	aesgcm, err := cipher.NewGCMWithTagSize(block, 16)
	if err != nil {
		return "", fmt.Errorf("decryptAccessToken: cipher.NewGCMWithTagSize: %w", err)
	}

	// crypto/cipher's GCM expects the auth tag appended to the ciphertext.
	combined := make([]byte, 0, len(ciphertext)+len(ciphertag))
	combined = append(combined, ciphertext...)
	combined = append(combined, ciphertag...)

	plaintext, err := aesgcm.Open(nil, nonce, combined, []byte("AES256GCM"))
	if err != nil {
		return "", fmt.Errorf("decryptAccessToken: aesgcm.Open (likely wrong ENCRYPTION_KEY or corrupt blob): %w", err)
	}
	return string(plaintext), nil
}

// getCarRegionAPI function to get URL from iis in accessToken
func getCarRegionAPI(accessToken string) CarRegionAPI {
	payload := strings.Split(accessToken, ".")
	if len(payload) != 3 {
		return GlobalAPI
	}
	decodedStr, err := base64.RawStdEncoding.DecodeString(payload[1])
	if err != nil {
		return GlobalAPI
	}
	var result map[string]interface{}
	if err = json.Unmarshal(decodedStr, &result); err != nil {
		return GlobalAPI
	}
	iss, ok := result["iss"].(string)
	if !ok {
		return GlobalAPI
	}
	issUrl, err := url.Parse(iss)
	if err != nil {
		return GlobalAPI
	}
	if strings.HasSuffix(issUrl.Host, ".cn") {
		return ChinaAPI
	}
	return GlobalAPI
}
