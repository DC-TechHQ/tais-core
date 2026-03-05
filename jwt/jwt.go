package jwt

import (
	"fmt"
	"net"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType distinguishes staff from citizen tokens.
type TokenType string

const (
	TypeStaff   TokenType = "staff"
	TypeCitizen TokenType = "citizen"
)

// Config holds the JWT signing secret and default TTL.
type Config struct {
	Secret     string
	TTLSeconds int
}

// Claims is the canonical JWT payload used across all TAIS services.
// ip_net is only populated for staff tokens (first 3 octets of the staff IP).
type Claims struct {
	Sub   uint      `json:"sub"`              // user / citizen ID
	Type  TokenType `json:"type"`             // "staff" | "citizen"
	IpNet string    `json:"ip_net,omitempty"` // e.g. "10.200.1" — staff only
	JTI   string    `json:"jti"`              // unique token ID (blacklist key)
	jwt.RegisteredClaims
}

// Parse validates the signed token string and returns the embedded Claims.
// Returns an error for expired, malformed, or invalidly-signed tokens.
func Parse(tokenStr string, cfg Config) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt: parse: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt: invalid token")
	}
	return claims, nil
}

// Sign creates and signs a new JWT token for the given claims.
// ExpiresAt is set automatically based on cfg.TTLSeconds.
func Sign(claims *Claims, cfg Config) (string, error) {
	ttl := cfg.TTLSeconds
	if ttl <= 0 {
		ttl = 3600
	}
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(ttl) * time.Second)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

// CheckIPNet validates that the request IP belongs to the /24 subnet stored in
// the token's ip_net claim.  For citizen tokens (ip_net == ""), always returns true.
func CheckIPNet(claims *Claims, requestIP string) bool {
	if claims.IpNet == "" {
		return true
	}
	cidr := claims.IpNet + ".0/24"
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(requestIP)
	if ip == nil {
		return false
	}
	return network.Contains(ip)
}
