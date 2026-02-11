package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func HashJSON(jsonData any) string {
	data, _ := json.Marshal(jsonData)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
