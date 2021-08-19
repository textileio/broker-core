package nearjwt

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"

	"github.com/golang-jwt/jwt"
)

// SigningMethodNear implements the NEAR signing method.
// Expects ed25519.Ed25519PrivateKey for signing and ed25519.Ed25519PublicKey
// for validation.
type SigningMethodNear struct {
	Name string
}

// SigningMethod is a specific instance of the NEAR signing method.
var SigningMethod *SigningMethodNear

func init() {
	SigningMethod = &SigningMethodNear{"NEAR"}
	jwt.RegisterSigningMethod(SigningMethod.Alg(), func() jwt.SigningMethod {
		return SigningMethod
	})
}

// Alg returns the name of this signing method.
func (m *SigningMethodNear) Alg() string {
	return m.Name
}

// Verify implements the Verify method from SigningMethod.
// For this signing method, must be a ed25519.PublicKey structure.
func (m *SigningMethodNear) Verify(signingString, signature string, key interface{}) error {
	var err error

	// Decode the signature
	var sig []byte
	if sig, err = jwt.DecodeSegment(signature); err != nil {
		return err
	}

	var ed25519Key ed25519.PublicKey
	var ok bool

	if ed25519Key, ok = key.(ed25519.PublicKey); !ok {
		return jwt.ErrInvalidKeyType
	}

	h := sha256.New()
	_, err = h.Write([]byte(signingString))
	if err != nil {
		return err
	}
	signingString = string(h.Sum(nil))

	// verify the signature
	valid := ed25519.Verify(ed25519Key, []byte(signingString), sig)
	if !valid {
		return jwt.ErrSignatureInvalid
	}

	return nil
}

// Sign implements the Sign method from SigningMethod.
// For this signing method, must be a ed25519.Ed25519PrivateKey structure.
func (m *SigningMethodNear) Sign(signingString string, key interface{}) (string, error) {
	var ed25519Key ed25519.PrivateKey
	var ok bool

	// validate type of key
	if ed25519Key, ok = key.(ed25519.PrivateKey); !ok {
		return "", jwt.ErrInvalidKey
	}

	h := sha256.New()
	_, err := h.Write([]byte(signingString))
	if err != nil {
		return "", err
	}
	signingString = string(h.Sum(nil))

	sigBytes, err := ed25519Key.Sign(rand.Reader, []byte(signingString), crypto.Hash(0))
	if err != nil {
		return "", err
	}
	return jwt.EncodeSegment(sigBytes), nil
}
