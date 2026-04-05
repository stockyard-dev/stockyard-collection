package server

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"
)

const publicKeyHex = "3af8f9593b3331c27994f1eeacf111c727ff6015016b0af44ed3ca6934d40b13"

type Limits struct {
	MaxItems int    `json:"max_items"`
	MaxCategories int    `json:"max_categories"`
	Search      bool   `json:"search"`
	Export      bool   `json:"export"`
	Tier        string `json:"tier"`
}

func FreeLimits() Limits {
	return Limits{MaxItems: 25, MaxCategories: 5, Search: false, Export: false, Tier: "free"}
}

func ProLimits() Limits {
	return Limits{MaxItems: 0, MaxCategories: 0, Search: true, Export: true, Tier: "pro"}
}

func DefaultLimits() Limits {
	key := os.Getenv("STOCKYARD_LICENSE_KEY")
	if key == "" {
		log.Printf("[license] No license key — running on free tier")
		log.Printf("[license] Set STOCKYARD_LICENSE_KEY to unlock Pro features")
		return FreeLimits()
	}
	if validateLicenseKey(key, "collection") {
		log.Printf("[license] Valid Pro license — all features unlocked")
		return ProLimits()
	}
	log.Printf("[license] Invalid license key — running on free tier")
	return FreeLimits()
}

func LimitReached(limit, current int) bool {
	return limit > 0 && current >= limit
}

func validateLicenseKey(key, product string) bool {
	if !strings.HasPrefix(key, "SY-") {
		return false
	}
	parts := strings.SplitN(key[3:], ".", 2)
	if len(parts) != 2 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	pub, err := hexDecode(publicKeyHex)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), payload, sig) {
		return false
	}
	var claims struct {
		Product string `json:"p"`
		Expires int64  `json:"x"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return false
	}
	if claims.Expires > 0 && time.Now().Unix() > claims.Expires {
		return false
	}
	return claims.Product == "*" || claims.Product == "stockyard" || claims.Product == product
}

func hexDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, os.ErrInvalid
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		h, l := hexVal(s[i]), hexVal(s[i+1])
		if h == 255 || l == 255 {
			return nil, os.ErrInvalid
		}
		b[i/2] = h<<4 | l
	}
	return b, nil
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 255
}
