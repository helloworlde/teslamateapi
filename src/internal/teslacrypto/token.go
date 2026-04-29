package teslacrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

type CarRegionAPI string

const (
	ChinaAPI  CarRegionAPI = "China"
	GlobalAPI CarRegionAPI = "Global"
)

// DecryptAccessToken 解密 TeslaMate private.tokens.access；解密失败时只返回空字符串，避免泄漏密文或异常细节。
func DecryptAccessToken(data string, encryptionKey string) (token string) {
	defer func() {
		if recover() != nil {
			token = ""
		}
	}()

	/*
	   From Adrian....
	   I had a look at how to decode the binary input without additional libraries. Below is sample code for Elixir. An important detail is that  "Additional Authenticated Data (AAD) " is required to decrypt the tokens. The AAD is a fixed string, in this case "AES256GCM”.
	   << _type::bytes-1, length::integer, _tag::bytes-size(length), iv::bytes-12, ciphertag::bytes-16, ciphertext::bytes >> = input
	   key = :crypto.hash(:sha256, key)
	   aad = "AES256GCM"
	   plaintext = :crypto.crypto_one_time_aead(:aes_256_gcm, key, iv, ciphertext, aad, ciphertag, false)

	   How the encrypted content looks like....
	   +----------------------------------------------------------+----------------------+
	   |                          HEADER                          |         BODY         |
	   +-------------------+---------------+----------------------+----------------------+
	   | Key Tag (n bytes) | IV (n bytes)  | Ciphertag (16 bytes) | Ciphertext (n bytes) |
	   +-------------------+---------------+----------------------+----------------------+
	   |                   |_________________________________
	   |                                                     |
	   +---------------+-----------------+-------------------+
	   | Type (1 byte) | Length (1 byte) | Key Tag (n bytes) |
	   +---------------+-----------------+-------------------+
	*/

	h := sha256.New()
	h.Write([]byte(encryptionKey))

	key := h.Sum(nil)
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	if len(data) < 31 {
		return ""
	}

	// second byte
	keyLen := int([]rune(data)[1])
	if keyLen < 0 || len(data) < 2+keyLen+12+16 {
		return ""
	}

	/*
	   With AES.GCM, 12-byte IV length is necessary for interoperability reasons.
	   See https://github.com/danielberkompas/cloak/issues/93
	   IV and nonce are often used interchangeably. Essentially though, an IV is a nonce with an additional requirement: it must be selected in a non-predictable way
	   https://medium.com/@fridakahsas/salt-nonces-and-ivs-whats-the-difference-d7a44724a447#:~:text=IV%20and%20nonce%20are%20often,an%20IV%20must%20be%20random.
	*/

	nonce := data[2+keyLen : 2+keyLen+12]

	aesgcm, err := cipher.NewGCMWithTagSize(block, 16)
	if err != nil {
		return ""
	}

	// https://stackoverflow.com/a/68353192
	// golang aes expects cipertag to append ciphertext....
	ciphertextTag := data[2+keyLen+12+16:] + data[2+keyLen+12:2+keyLen+12+16]

	// AES256GCM -- Additional Authenticated Data (AAD)
	plaintext, err := aesgcm.Open(nil, []byte(nonce), []byte(ciphertextTag), []byte("AES256GCM"))
	if err != nil {
		return ""
	}

	return string(plaintext)
}

// GetCarRegionAPI 根据 JWT iss 判断 Tesla Owner API 区域。
func GetCarRegionAPI(accessToken string) CarRegionAPI {
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
	if !ok || iss == "" {
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
