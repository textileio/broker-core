package secp

import (
	"github.com/filecoin-project/go-crypto"
	crypto2 "github.com/filecoin-project/go-state-types/crypto"
	"github.com/minio/blake2b-simd"
)

// ToPublic returns the public key associated with the private key.
func ToPublic(pk []byte) ([]byte, error) {
	return crypto.PublicKey(pk), nil
}

// Sign creates a signature for the message.
func Sign(pk []byte, msg []byte) (*crypto2.Signature, error) {
	b2sum := blake2b.Sum256(msg)
	sig, err := crypto.Sign(pk, b2sum[:])
	if err != nil {
		return nil, err
	}

	return &crypto2.Signature{
		Type: crypto2.SigTypeSecp256k1,
		Data: sig,
	}, nil
}
