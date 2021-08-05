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
// Expects *crypto.Ed25519PrivateKey for signing and *crypto.Ed25519PublicKey
// for validation.
type SigningMethodEth struct {
	Name string
}

// SigningMethod is a specific instance for Ed25519.
var SigningMethod *SigningMethodEth

// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L404
// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calculated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
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
	var err error

	// Decode the signature
	var sig []byte
	if sig, err = jwt.DecodeSegment(signature); err != nil {
		return err
	}

	// if sig[64] != 27 && sig[64] != 28 {
	// 	return jwt.ErrSignatureInvalid
	// }
	// sig[64] -= 27

	expectedAddr, ok := address.(common.Address)
	if !ok {
		return jwt.ErrInvalidKeyType
	}

	msg := []byte(signingString)
	sigPublicKeyECDSA, err := crypto.SigToPub(signHash(msg), sig)
	if err != nil {
		return err
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
	// // validate type of key
	var privateKeyPointer *ecdsa.PrivateKey
	var ok bool
	privateKeyPointer, ok = privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return "", jwt.ErrInvalidKey
	}

	sigBytes, err := crypto.Sign(signHash([]byte(signingString)), privateKeyPointer)

	if err != nil {
		return "", err
	}
	return jwt.EncodeSegment(sigBytes), nil
}
