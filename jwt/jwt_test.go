package jwt_test

import (
	"testing"
	"time"

	"github.com/DC-TechHQ/tais-core/jwt"
	gojwt "github.com/golang-jwt/jwt/v5"
)

var testCfg = jwt.Config{Secret: "test-secret-key-32-bytes-long!!"}

// sign is a local helper to create signed tokens for testing.
// In production, signing is done by tais-auth — tais-core only parses.
func sign(claims *jwt.Claims, secret string) string {
	claims.RegisteredClaims = gojwt.RegisteredClaims{
		ExpiresAt: gojwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  gojwt.NewNumericDate(time.Now()),
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

func TestParse_Valid(t *testing.T) {
	claims := &jwt.Claims{
		Sub:   42,
		Type:  jwt.TypeStaff,
		IpNet: "10.200.1",
		JTI:   "unique-jti-001",
	}

	token := sign(claims, testCfg.Secret)

	parsed, err := jwt.Parse(token, testCfg)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.Sub != 42 {
		t.Errorf("Sub: got %d, want 42", parsed.Sub)
	}
	if parsed.Type != jwt.TypeStaff {
		t.Errorf("Type: got %q, want %q", parsed.Type, jwt.TypeStaff)
	}
	if parsed.JTI != "unique-jti-001" {
		t.Errorf("JTI: got %q, want %q", parsed.JTI, "unique-jti-001")
	}
}

func TestParse_WrongSecret(t *testing.T) {
	claims := &jwt.Claims{Sub: 1, JTI: "jti-1"}
	token := sign(claims, testCfg.Secret)

	wrongCfg := jwt.Config{Secret: "wrong-secret-key-32-bytes-long!!"}
	if _, err := jwt.Parse(token, wrongCfg); err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestParse_MalformedToken(t *testing.T) {
	if _, err := jwt.Parse("not.a.valid.token", testCfg); err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestParse_ExpiredToken(t *testing.T) {
	claims := &jwt.Claims{Sub: 1, JTI: "jti-expired"}
	claims.RegisteredClaims = gojwt.RegisteredClaims{
		ExpiresAt: gojwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testCfg.Secret))

	if _, err := jwt.Parse(signed, testCfg); err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestCheckIPNet_StaffValid(t *testing.T) {
	claims := &jwt.Claims{IpNet: "10.200.1"}
	if !jwt.CheckIPNet(claims, "10.200.1.55") {
		t.Error("expected valid IP to pass subnet check")
	}
}

func TestCheckIPNet_StaffInvalid(t *testing.T) {
	claims := &jwt.Claims{IpNet: "10.200.1"}
	if jwt.CheckIPNet(claims, "10.200.2.55") {
		t.Error("expected IP from different subnet to fail check")
	}
}

func TestCheckIPNet_CitizenAlwaysValid(t *testing.T) {
	// Citizens have no ip_net binding.
	claims := &jwt.Claims{IpNet: "", Type: jwt.TypeCitizen}
	if !jwt.CheckIPNet(claims, "203.0.113.42") {
		t.Error("citizen token should always pass IP check")
	}
}
