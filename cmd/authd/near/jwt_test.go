package nearjwt

import (
	"crypto/ed25519"
	"crypto/rand"
	"log"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt"
)

var privateKey ed25519.PrivateKey
var signingMethod *SigningMethodNear
var testData []struct {
	name        string
	tokenString string
	alg         string
	claims      map[string]interface{}
	valid       bool
}

func init() {
	signingMethod = &SigningMethodNear{"NEAR"}
	jwt.RegisterSigningMethod(SigningMethod.Alg(), func() jwt.SigningMethod {
		return SigningMethod
	})

	var err error
	_, privateKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	testData = []struct {
		name        string
		tokenString string
		alg         string
		claims      map[string]interface{}
		valid       bool
	}{
		{
			"valid key",
			generateToken(true),
			"NEAR",
			map[string]interface{}{"jti": "foo", "sub": "bar"},
			true,
		},
		{
			"invalid key",
			generateToken(false),
			"NEAR",
			map[string]interface{}{"jti": "foo", "sub": "bar"},
			false,
		},
	}
}

func TestSigningMethodEth_Alg(t *testing.T) {
	if SigningMethod.Alg() != "NEAR" {
		t.Fatal("wrong alg")
	}
}

func TestSigningMethodEth_Sign(t *testing.T) {
	for _, data := range testData {
		if data.valid {
			parts := strings.Split(data.tokenString, ".")

			method := jwt.GetSigningMethod(data.alg)
			sig, err := method.Sign(strings.Join(parts[0:2], "."), privateKey)
			if err != nil {
				t.Errorf("[%v] error signing token: %v", data.name, err)
			}
			if sig != parts[2] {
				t.Errorf("[%v] incorrect signature.\nwas:\n%v\nexpecting:\n%v", data.name, sig, parts[2])
			}
		}
	}
}

func TestSigningMethodEth_Verify(t *testing.T) {
	pk := privateKey.Public()
	publicKey, ok := pk.(ed25519.PublicKey)
	if !ok {
		log.Fatal("error casting public key to Ed25519")
	}

	for _, data := range testData {
		parts := strings.Split(data.tokenString, ".")

		method := jwt.GetSigningMethod(data.alg)

		signingString := strings.Join(parts[0:2], ".")

		err := method.Verify(signingString, parts[2], publicKey)
		if data.valid && err != nil {
			t.Errorf("[%v] error while verifying key: %v", data.name, err)
		}
		if !data.valid && err == nil {
			t.Errorf("[%v] invalid key passed validation", data.name)
		}
	}
}

func TestGenerateEthToken(t *testing.T) {
	claims := &jwt.StandardClaims{
		Id:      "foo",
		Subject: "bar",
	}
	_, err := jwt.NewWithClaims(SigningMethod, claims).SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
}

func generateToken(valid bool) string {
	if valid {
		claims := &jwt.StandardClaims{
			Id:      "foo",
			Subject: "bar",
		}
		token, _ := jwt.NewWithClaims(SigningMethod, claims).SignedString(privateKey)
		return token
	}
	_, sk, _ := ed25519.GenerateKey(rand.Reader)
	claims := &jwt.StandardClaims{
		Id:      "foo",
		Subject: "bar",
	}
	token, _ := jwt.NewWithClaims(SigningMethod, claims).SignedString(sk)
	return token
}
