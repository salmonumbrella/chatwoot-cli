package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func newIdempotencyKey() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("cwcli_%d", time.Now().UnixNano())
	}
	return "cwcli_" + hex.EncodeToString(buf)
}
