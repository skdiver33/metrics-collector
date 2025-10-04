package misc

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func GetRequestHash(body []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(body)
	dst := h.Sum(nil)
	hash := hex.EncodeToString(dst)
	return hash
}
