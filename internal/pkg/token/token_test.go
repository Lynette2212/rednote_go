package token

import (
	"testing"
	"time"

	"rednote/internal/pkg/config"

	"github.com/golang-jwt/jwt/v4"
)

func testSecret() {
	config.Instance = &config.Config{JwtSecret: "unit-test-secret"}
}

func TestGenerateAndParse(t *testing.T) {
	testSecret()
	raw, err := Generate(42)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserId != 42 || claims.ID == "" {
		t.Fatalf("claims 异常: %+v", claims)
	}
}

func TestParseRejectsAlgorithmConfusion(t *testing.T) {
	testSecret()
	claims := Claims{
		UserId: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	hs512, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte("unit-test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(hs512); err == nil {
		t.Fatal("HS512 token 不应通过校验")
	}

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(none); err == nil {
		t.Fatal("alg=none token 不应通过校验")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	testSecret()
	claims := Claims{
		UserId: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("unit-test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(raw); err == nil {
		t.Fatal("过期 token 不应通过校验")
	}
}
