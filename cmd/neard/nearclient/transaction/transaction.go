package transaction

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/near/borsh-go"
	"github.com/textileio/broker-core/cmd/neard/nearclient/keys"
)

// Signature asdf.
type Signature struct {
	KeyType uint8
	Data    [64]byte
}

// SignedTransaction asdf.
type SignedTransaction struct {
	Transaction Transaction
	Signature   Signature
}

// Transaction asdf.
type Transaction struct {
	SignerID   string
	PublicKey  PublicKey
	Nonce      uint64
	ReceiverID string
	BlockHash  [32]byte
	Actions    []Action
}

// PublicKey asdf.
type PublicKey struct {
	KeyType uint8
	Data    [32]byte
}

// AccessKey asdf.
type AccessKey struct {
	Nonce      uint64
	Permission AccessKeyPermission
}

// AccessKeyPermission asdf.
type AccessKeyPermission struct {
	Enum         borsh.Enum `borsh_enum:"true"`
	FunctionCall FunctionCallPermission
	FullAccess   FullAccessPermission
}

// FunctionCallPermission asdf.
type FunctionCallPermission struct {
	Allowance   *big.Int
	ReceiverID  string
	MethodNames []string
}

// FullAccessPermission asdf.
type FullAccessPermission struct{}

// Action asdf.
type Action struct {
	Enum           borsh.Enum `borsh_enum:"true"`
	CreateAccount  CreateAccount
	DeployContract DeployContract
	FunctionCall   FunctionCall
	Transfer       Transfer
	Stake          Stake
	AddKey         AddKey
	DeleteKey      DeleteKey
	DeleteAccount  DeleteAccount
}

// CreateAccount asdf.
type CreateAccount struct{}

// DeployContract asdf.
type DeployContract struct {
	Code []byte
}

// FunctionCall asdf.
type FunctionCall struct {
	MethodName string
	Args       []byte
	Gas        uint64
	Deposit    big.Int
}

// Transfer asdf.
type Transfer struct {
	Deposit big.Int
}

// Stake sadf.
type Stake struct {
	Stake     big.Int
	PublicKey PublicKey
}

// AddKey asdf.
type AddKey struct {
	PublicKey PublicKey
	AccessKey AccessKey
}

// DeleteKey asdf.
type DeleteKey struct {
	PublicKey PublicKey
}

// DeleteAccount asdf.
type DeleteAccount struct {
	BeneficiaryID string
}

func CreateAccountAction() Action {
	return Action{Enum: 0, CreateAccount: CreateAccount{}}
}

func DeployContractAction(code []byte) Action {
	return Action{Enum: 1, DeployContract: DeployContract{Code: code}}
}

func FunctionCallAction(methodName string, args []byte, gas uint64, deposit big.Int) Action {
	return Action{
		Enum: 2,
		FunctionCall: FunctionCall{
			MethodName: methodName,
			Args:       args,
			Gas:        gas,
			Deposit:    deposit,
		},
	}
}

func TransferAction(deposit big.Int) Action {
	return Action{Enum: 3, Transfer: Transfer{Deposit: deposit}}
}

func StakeAction(stake big.Int, publicKey keys.PublicKey) Action {
	// TODO: make keys.PublicKey the serializable model.
	var dataArr [32]byte
	copy(dataArr[:], publicKey.Data)
	return Action{
		Enum: 4,
		Stake: Stake{
			Stake: stake,
			PublicKey: PublicKey{
				KeyType: uint8(publicKey.Type),
				Data:    dataArr,
			},
		},
	}
}

func AddKeyAction(publicKey keys.PublicKey, accessKey AccessKey) Action {
	// TODO: make keys.PublicKey the serializable model.
	// TODO: better way of specifying AccessKey.
	var dataArr [32]byte
	copy(dataArr[:], publicKey.Data)
	return Action{
		Enum: 5,
		AddKey: AddKey{
			PublicKey: PublicKey{
				KeyType: uint8(publicKey.Type),
				Data:    dataArr,
			},
			AccessKey: accessKey,
		},
	}
}

func DeleteKeyAction(publicKey keys.PublicKey) Action {
	// TODO: make keys.PublicKey the serializable model.
	var dataArr [32]byte
	copy(dataArr[:], publicKey.Data)
	return Action{
		Enum: 6,
		DeleteKey: DeleteKey{
			PublicKey: PublicKey{
				KeyType: uint8(publicKey.Type),
				Data:    dataArr,
			},
		},
	}
}

func DeleteAccountAction(beneficiaryID string) Action {
	return Action{
		Enum: 7,
		DeleteAccount: DeleteAccount{
			BeneficiaryID: beneficiaryID,
		},
	}
}

// NewTransaction creates a new Transaction.
func NewTransaction(
	signerID string,
	publicKey PublicKey,
	nonce uint64,
	receiverID string,
	blockHash []byte,
	actions []Action,
) *Transaction {
	var blockHashArr [32]byte
	copy(blockHashArr[:], blockHash)
	return &Transaction{
		SignerID:   signerID,
		PublicKey:  publicKey,
		Nonce:      nonce,
		ReceiverID: receiverID,
		BlockHash:  blockHashArr,
		Actions:    actions,
	}
}

// SignTransaction serializes and signs a Transaction using the provided signer.
func SignTransaction(
	transaction Transaction,
	signer keys.KeyPair,
	accountID string,
	networkID string,
) ([]byte, *SignedTransaction, error) {
	message, err := borsh.Serialize(transaction)
	if err != nil {
		return nil, nil, fmt.Errorf("serializing transaction: %v", err)
	}
	hash := sha256.Sum256(message)
	sig, err := signer.Sign(hash[:])
	if err != nil {
		return nil, nil, fmt.Errorf("signing hash: %v", err)
	}
	var data [64]byte
	copy(data[:], sig)
	st := &SignedTransaction{
		Transaction: transaction,
		Signature: Signature{
			KeyType: transaction.PublicKey.KeyType,
			Data:    data,
		},
	}
	return hash[:], st, nil
}
