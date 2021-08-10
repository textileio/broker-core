package ethjwt

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt"
)

// SigningMethodEth implements the ETH signing method.
// Expects *ecdsa.PrivateKey for signing and common.Address for validation.
type SigningMethodEth struct {
	Name string
}

// SigningMethod is a specific instance for Ed25519.
var SigningMethod *SigningMethodEth

// PrefixedHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calculated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func PrefixedHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

func init() {
	SigningMethod = &SigningMethodEth{"ETH"}
	jwt.RegisterSigningMethod(SigningMethod.Alg(), func() jwt.SigningMethod {
		return SigningMethod
	})
}

// Alg returns the name of this signing method.
func (m *SigningMethodEth) Alg() string {
	return m.Name
}

// Verify implements the Verify method from SigningMethod.
// For this signing method, must be a common.Address.
func (m *SigningMethodEth) Verify(signingString, signature string, address interface{}) error {
	// Decode the signature
	sig, err := jwt.DecodeSegment(signature)
	if err != nil {
		return fmt.Errorf("decoding signature: %v", err)
	}

	if sig[64] != 27 && sig[64] != 28 {
		return fmt.Errorf("sig[64] is not 27 or 28")
	}
	sig[64] -= 27

	expectedAddr, ok := address.(common.Address)
	if !ok {
		return jwt.ErrInvalidKeyType
	}

	msg := []byte(signingString)
	sigPublicKeyECDSA, err := crypto.SigToPub(PrefixedHash(msg), sig)
	if err != nil {
		return fmt.Errorf("converting sig to pub key: %v", err)
	}
	recoveredAddr := crypto.PubkeyToAddress(*sigPublicKeyECDSA)

	matches := bytes.Equal(expectedAddr.Bytes(), recoveredAddr.Bytes())
	if !matches {
		return jwt.ErrSignatureInvalid
	}

	return nil
}

// Sign implements the Sign method from SigningMethod.
// For this signing method, must be a *ecdsa.PrivateKey structure.
func (m *SigningMethodEth) Sign(signingString string, privateKey interface{}) (string, error) {
	// validate type of key
	privateKeyPointer, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return "", jwt.ErrInvalidKey
	}

	sigBytes, err := crypto.Sign(PrefixedHash([]byte(signingString)), privateKeyPointer)
	if err != nil {
		return "", fmt.Errorf("signing: %v", err)
	}

	return jwt.EncodeSegment(sigBytes), nil
}
