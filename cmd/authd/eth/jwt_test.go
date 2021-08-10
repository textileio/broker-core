package ethjwt

import (
	"log"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt"
)

var privateKey string
var signingMethod *SigningMethodEth
var testData []struct {
	name        string
	tokenString string
	alg         string
	claims      map[string]interface{}
	valid       bool
}

func init() {
	signingMethod = &SigningMethodEth{"ETH"}
	jwt.RegisterSigningMethod(SigningMethod.Alg(), func() jwt.SigningMethod {
		return SigningMethod
	})

	pk, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	privateKeyBytes := crypto.FromECDSA(pk)
	privateKey = hexutil.Encode(privateKeyBytes)[2:]

	testData = []struct {
		name        string
		tokenString string
		alg         string
		claims      map[string]interface{}
		valid       bool
	}{
		{
			"ETH",
			generateToken(true),
			"ETH",
			map[string]interface{}{"jti": "foo", "sub": "bar"},
			true,
		},
		{
			"invalid key",
			generateToken(false),
			"ETH",
			map[string]interface{}{"jti": "foo", "sub": "bar"},
			false,
		},
	}
}

func TestSigningMethodEth_Alg(t *testing.T) {
	if SigningMethod.Alg() != "ETH" {
		t.Fatal("wrong alg")
	}
}

func TestSigningMethodEth_Sign(t *testing.T) {
	sk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range testData {
		if data.valid {
			parts := strings.Split(data.tokenString, ".")

			method := jwt.GetSigningMethod(data.alg)
			sig, err := method.Sign(strings.Join(parts[0:2], "."), sk)
			if err != nil {
				t.Errorf("[%v] error signing token: %v", data.name, err)
			}
			if sig != parts[2] {
				t.Errorf("[%v] incorrect signature.\nwas:\n%v\nexpecting:\n%v", data.name, sig, parts[2])
			}
		}
	}
}

// func TestSigningMethodEth_Verify(t *testing.T) {
// 	sk, err := crypto.HexToECDSA(privateKey)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	pk := sk.Public()
// 	publicKeyECDSA, ok := pk.(*ecdsa.PublicKey)
// 	if !ok {
// 		log.Fatal("error casting public key to ECDSA")
// 	}
// 	address := crypto.PubkeyToAddress(*publicKeyECDSA)

// 	for _, data := range testData {
// 		parts := strings.Split(data.tokenString, ".")

// 		method := jwt.GetSigningMethod(data.alg)

// 		signingString := strings.Join(parts[0:2], ".")

// 		err := method.Verify(signingString, parts[2], address)
// 		if data.valid && err != nil {
// 			t.Errorf("[%v] error while verifying key: %v", data.name, err)
// 		}
// 		if !data.valid && err == nil {
// 			t.Errorf("[%v] invalid key passed validation", data.name)
// 		}
// 	}
// }

func TestGenerateEthToken(t *testing.T) {
	sk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	claims := &jwt.StandardClaims{
		Id:      "foo",
		Subject: "bar",
	}
	_, err = jwt.NewWithClaims(SigningMethod, claims).SignedString(sk)
	if err != nil {
		t.Fatal(err)
	}
}

func generateToken(valid bool) string {
	if valid {
		sk, _ := crypto.HexToECDSA(privateKey)
		claims := &jwt.StandardClaims{
			Id:      "foo",
			Subject: "bar",
		}
		token, _ := jwt.NewWithClaims(SigningMethod, claims).SignedString(sk)
		return token
	}
	sk, _ := crypto.GenerateKey()
	claims := &jwt.StandardClaims{
		Id:      "foo",
		Subject: "bar",
	}
	token, _ := jwt.NewWithClaims(SigningMethod, claims).SignedString(sk)
	return token
}
