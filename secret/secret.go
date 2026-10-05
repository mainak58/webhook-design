package secret

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

var secretKey = "this is a secretKey"

func computeHMAC(data string) []byte {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func CreateHMACSHA256(data string) string {
	return hex.EncodeToString(computeHMAC(data))
}

func VerifyHMACSHA256(data, givenHex string) bool {
	given, err := hex.DecodeString(givenHex)
	if err != nil {
		return false
	}
	return hmac.Equal(computeHMAC(data), given)
}
