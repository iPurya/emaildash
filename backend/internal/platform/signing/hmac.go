package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type HMACSigner struct{}

func NewHMACSigner() HMACSigner {
	return HMACSigner{}
}

func (HMACSigner) Sign(secret []byte, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func (s HMACSigner) Verify(secret []byte, timestamp, signature string, body []byte, maxSkew time.Duration) error {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	if skew := time.Since(time.Unix(ts, 0)); skew > maxSkew || skew < -maxSkew {
		return fmt.Errorf("stale signature")
	}
	expected := s.Sign(secret, timestamp, body)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(strings.TrimSpace(signature))) != 1 {
		return fmt.Errorf("invalid signature")
	}
	return nil
}
