package identity

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/crypto"
	mbase "github.com/multiformats/go-multibase"
)

// NearIdentity wraps crypto.PrivKey, overwriting GetPublic with thread.PubKey.
type NearIdentity struct {
	crypto.PrivKey
}

// NewNearIdentity returns a new NearIdentity.
func NewNearIdentity(key crypto.PrivKey) *NearIdentity {
	return &NearIdentity{PrivKey: key}
}

func (i *NearIdentity) MarshalBinary() ([]byte, error) {
	return crypto.MarshalPrivateKey(i.PrivKey)
}

func (i *NearIdentity) UnmarshalBinary(bytes []byte) (err error) {
	i.PrivKey, err = crypto.UnmarshalPrivateKey(bytes)
	if err != nil {
		return err
	}
	return err
}

func (i *NearIdentity) String() string {
	bytes, err := crypto.MarshalPrivateKey(i.PrivKey)
	if err != nil {
		panic(fmt.Errorf("marshal privkey: %v", err))
	}
	str, err := mbase.Encode(mbase.Base32, bytes)
	if err != nil {
		panic(fmt.Errorf("multibase encoding privkey: %v", err))
	}
	return str
}

func (i *NearIdentity) UnmarshalString(str string) error {
	_, bytes, err := mbase.Decode(str)
	if err != nil {
		return err
	}
	i.PrivKey, err = crypto.UnmarshalPrivateKey(bytes)
	return err
}

func (i *NearIdentity) Sign(_ context.Context, msg []byte) ([]byte, error) {
	return i.PrivKey.Sign(msg)
}

func (i *NearIdentity) GetPublic() crypto.PubKey {
	return i.PrivKey.GetPublic()
}

func (i *NearIdentity) Equals(id NearIdentity) bool {
	return i.PrivKey.Equals(id.PrivKey)
}
