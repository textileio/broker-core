// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contractclient

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// BridgeProviderABI is the input ABI used to generate the binding from.
const BridgeProviderABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"AddDeposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"RelDeposit\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"addDeposit\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"apiEndpoint\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"deposits\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasDeposit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"listDepositees\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"providerProportion\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"relDeposit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"relDeposits\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sessionDivisor\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"a\",\"type\":\"string\"}],\"name\":\"setApiEndpoint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"p\",\"type\":\"uint256\"}],\"name\":\"setProviderProportion\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"m\",\"type\":\"uint256\"}],\"name\":\"setSessionDivisor\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// BridgeProvider is an auto generated Go binding around an Ethereum contract.
type BridgeProvider struct {
	BridgeProviderCaller     // Read-only binding to the contract
	BridgeProviderTransactor // Write-only binding to the contract
	BridgeProviderFilterer   // Log filterer for contract events
}

// BridgeProviderCaller is an auto generated read-only Go binding around an Ethereum contract.
type BridgeProviderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeProviderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BridgeProviderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeProviderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BridgeProviderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeProviderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BridgeProviderSession struct {
	Contract     *BridgeProvider   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeProviderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BridgeProviderCallerSession struct {
	Contract *BridgeProviderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// BridgeProviderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BridgeProviderTransactorSession struct {
	Contract     *BridgeProviderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// BridgeProviderRaw is an auto generated low-level Go binding around an Ethereum contract.
type BridgeProviderRaw struct {
	Contract *BridgeProvider // Generic contract binding to access the raw methods on
}

// BridgeProviderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BridgeProviderCallerRaw struct {
	Contract *BridgeProviderCaller // Generic read-only contract binding to access the raw methods on
}

// BridgeProviderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BridgeProviderTransactorRaw struct {
	Contract *BridgeProviderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBridgeProvider creates a new instance of BridgeProvider, bound to a specific deployed contract.
func NewBridgeProvider(address common.Address, backend bind.ContractBackend) (*BridgeProvider, error) {
	contract, err := bindBridgeProvider(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BridgeProvider{BridgeProviderCaller: BridgeProviderCaller{contract: contract}, BridgeProviderTransactor: BridgeProviderTransactor{contract: contract}, BridgeProviderFilterer: BridgeProviderFilterer{contract: contract}}, nil
}

// NewBridgeProviderCaller creates a new read-only instance of BridgeProvider, bound to a specific deployed contract.
func NewBridgeProviderCaller(address common.Address, caller bind.ContractCaller) (*BridgeProviderCaller, error) {
	contract, err := bindBridgeProvider(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeProviderCaller{contract: contract}, nil
}

// NewBridgeProviderTransactor creates a new write-only instance of BridgeProvider, bound to a specific deployed contract.
func NewBridgeProviderTransactor(address common.Address, transactor bind.ContractTransactor) (*BridgeProviderTransactor, error) {
	contract, err := bindBridgeProvider(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeProviderTransactor{contract: contract}, nil
}

// NewBridgeProviderFilterer creates a new log filterer instance of BridgeProvider, bound to a specific deployed contract.
func NewBridgeProviderFilterer(address common.Address, filterer bind.ContractFilterer) (*BridgeProviderFilterer, error) {
	contract, err := bindBridgeProvider(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BridgeProviderFilterer{contract: contract}, nil
}

// bindBridgeProvider binds a generic wrapper to an already deployed contract.
func bindBridgeProvider(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BridgeProviderABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeProvider *BridgeProviderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeProvider.Contract.BridgeProviderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeProvider *BridgeProviderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeProvider.Contract.BridgeProviderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeProvider *BridgeProviderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeProvider.Contract.BridgeProviderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeProvider *BridgeProviderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeProvider.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeProvider *BridgeProviderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeProvider.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeProvider *BridgeProviderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeProvider.Contract.contract.Transact(opts, method, params...)
}

// ApiEndpoint is a free data retrieval call binding the contract method 0xf25ecd5d.
//
// Solidity: function apiEndpoint() view returns(string)
func (_BridgeProvider *BridgeProviderCaller) ApiEndpoint(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _BridgeProvider.contract.Call(opts, &out, "apiEndpoint")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// ApiEndpoint is a free data retrieval call binding the contract method 0xf25ecd5d.
//
// Solidity: function apiEndpoint() view returns(string)
func (_BridgeProvider *BridgeProviderSession) ApiEndpoint() (string, error) {
	return _BridgeProvider.Contract.ApiEndpoint(&_BridgeProvider.CallOpts)
}

// ApiEndpoint is a free data retrieval call binding the contract method 0xf25ecd5d.
//
// Solidity: function apiEndpoint() view returns(string)
func (_BridgeProvider *BridgeProviderCallerSession) ApiEndpoint() (string, error) {
	return _BridgeProvider.Contract.ApiEndpoint(&_BridgeProvider.CallOpts)
}

// Deposits is a free data retrieval call binding the contract method 0xfc7e286d.
//
// Solidity: function deposits(address ) view returns(uint256 timestamp, address sender, uint256 value)
func (_BridgeProvider *BridgeProviderCaller) Deposits(opts *bind.CallOpts, arg0 common.Address) (struct {
	Timestamp *big.Int
	Sender    common.Address
	Value     *big.Int
}, error) {
	var out []interface{}
	err := _BridgeProvider.contract.Call(opts, &out, "deposits", arg0)

	outstruct := new(struct {
		Timestamp *big.Int
		Sender    common.Address
		Value     *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Timestamp = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Sender = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Value = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Deposits is a free data retrieval call binding the contract method 0xfc7e286d.
//
// Solidity: function deposits(address ) view returns(uint256 timestamp, address sender, uint256 value)
func (_BridgeProvider *BridgeProviderSession) Deposits(arg0 common.Address) (struct {
	Timestamp *big.Int
	Sender    common.Address
	Value     *big.Int
}, error) {
	return _BridgeProvider.Contract.Deposits(&_BridgeProvider.CallOpts, arg0)
}

// Deposits is a free data retrieval call binding the contract method 0xfc7e286d.
//
// Solidity: function deposits(address ) view returns(uint256 timestamp, address sender, uint256 value)
func (_BridgeProvider *BridgeProviderCallerSession) Deposits(arg0 common.Address) (struct {
	Timestamp *big.Int
	Sender    common.Address
	Value     *big.Int
}, error) {
	return _BridgeProvider.Contract.Deposits(&_BridgeProvider.CallOpts, arg0)
}

// HasDeposit is a free data retrieval call binding the contract method 0x1c48c7ec.
//
// Solidity: function hasDeposit(address account) view returns(bool)
func (_BridgeProvider *BridgeProviderCaller) HasDeposit(opts *bind.CallOpts, account common.Address) (bool, error) {
	var out []interface{}
	err := _BridgeProvider.contract.Call(opts, &out, "hasDeposit", account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasDeposit is a free data retrieval call binding the contract method 0x1c48c7ec.
//
// Solidity: function hasDeposit(address account) view returns(bool)
func (_BridgeProvider *BridgeProviderSession) HasDeposit(account common.Address) (bool, error) {
	return _BridgeProvider.Contract.HasDeposit(&_BridgeProvider.CallOpts, account)
}

// HasDeposit is a free data retrieval call binding the contract method 0x1c48c7ec.
//
// Solidity: function hasDeposit(address account) view returns(bool)
func (_BridgeProvider *BridgeProviderCallerSession) HasDeposit(account common.Address) (bool, error) {
	return _BridgeProvider.Contract.HasDeposit(&_BridgeProvider.CallOpts, account)
}

// ListDepositees is a free data retrieval call binding the contract method 0x2c40d143.
//
// Solidity: function listDepositees() view returns(address[])
func (_BridgeProvider *BridgeProviderCaller) ListDepositees(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _BridgeProvider.contract.Call(opts, &out, "listDepositees")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// ListDepositees is a free data retrieval call binding the contract method 0x2c40d143.
//
// Solidity: function listDepositees() view returns(address[])
func (_BridgeProvider *BridgeProviderSession) ListDepositees() ([]common.Address, error) {
	return _BridgeProvider.Contract.ListDepositees(&_BridgeProvider.CallOpts)
}

// ListDepositees is a free data retrieval call binding the contract method 0x2c40d143.
//
// Solidity: function listDepositees() view returns(address[])
func (_BridgeProvider *BridgeProviderCallerSession) ListDepositees() ([]common.Address, error) {
	return _BridgeProvider.Contract.ListDepositees(&_BridgeProvider.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BridgeProvider *BridgeProviderCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeProvider.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BridgeProvider *BridgeProviderSession) Owner() (common.Address, error) {
	return _BridgeProvider.Contract.Owner(&_BridgeProvider.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BridgeProvider *BridgeProviderCallerSession) Owner() (common.Address, error) {
	return _BridgeProvider.Contract.Owner(&_BridgeProvider.CallOpts)
}

// ProviderProportion is a free data retrieval call binding the contract method 0x15e00a77.
//
// Solidity: function providerProportion() view returns(uint256)
func (_BridgeProvider *BridgeProviderCaller) ProviderProportion(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeProvider.contract.Call(opts, &out, "providerProportion")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ProviderProportion is a free data retrieval call binding the contract method 0x15e00a77.
//
// Solidity: function providerProportion() view returns(uint256)
func (_BridgeProvider *BridgeProviderSession) ProviderProportion() (*big.Int, error) {
	return _BridgeProvider.Contract.ProviderProportion(&_BridgeProvider.CallOpts)
}

// ProviderProportion is a free data retrieval call binding the contract method 0x15e00a77.
//
// Solidity: function providerProportion() view returns(uint256)
func (_BridgeProvider *BridgeProviderCallerSession) ProviderProportion() (*big.Int, error) {
	return _BridgeProvider.Contract.ProviderProportion(&_BridgeProvider.CallOpts)
}

// SessionDivisor is a free data retrieval call binding the contract method 0xc5f056ba.
//
// Solidity: function sessionDivisor() view returns(uint256)
func (_BridgeProvider *BridgeProviderCaller) SessionDivisor(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _BridgeProvider.contract.Call(opts, &out, "sessionDivisor")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SessionDivisor is a free data retrieval call binding the contract method 0xc5f056ba.
//
// Solidity: function sessionDivisor() view returns(uint256)
func (_BridgeProvider *BridgeProviderSession) SessionDivisor() (*big.Int, error) {
	return _BridgeProvider.Contract.SessionDivisor(&_BridgeProvider.CallOpts)
}

// SessionDivisor is a free data retrieval call binding the contract method 0xc5f056ba.
//
// Solidity: function sessionDivisor() view returns(uint256)
func (_BridgeProvider *BridgeProviderCallerSession) SessionDivisor() (*big.Int, error) {
	return _BridgeProvider.Contract.SessionDivisor(&_BridgeProvider.CallOpts)
}

// AddDeposit is a paid mutator transaction binding the contract method 0xdea02892.
//
// Solidity: function addDeposit(address account) payable returns()
func (_BridgeProvider *BridgeProviderTransactor) AddDeposit(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "addDeposit", account)
}

// AddDeposit is a paid mutator transaction binding the contract method 0xdea02892.
//
// Solidity: function addDeposit(address account) payable returns()
func (_BridgeProvider *BridgeProviderSession) AddDeposit(account common.Address) (*types.Transaction, error) {
	return _BridgeProvider.Contract.AddDeposit(&_BridgeProvider.TransactOpts, account)
}

// AddDeposit is a paid mutator transaction binding the contract method 0xdea02892.
//
// Solidity: function addDeposit(address account) payable returns()
func (_BridgeProvider *BridgeProviderTransactorSession) AddDeposit(account common.Address) (*types.Transaction, error) {
	return _BridgeProvider.Contract.AddDeposit(&_BridgeProvider.TransactOpts, account)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_BridgeProvider *BridgeProviderTransactor) Initialize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "initialize")
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_BridgeProvider *BridgeProviderSession) Initialize() (*types.Transaction, error) {
	return _BridgeProvider.Contract.Initialize(&_BridgeProvider.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_BridgeProvider *BridgeProviderTransactorSession) Initialize() (*types.Transaction, error) {
	return _BridgeProvider.Contract.Initialize(&_BridgeProvider.TransactOpts)
}

// RelDeposit is a paid mutator transaction binding the contract method 0x712681cf.
//
// Solidity: function relDeposit(address account) returns()
func (_BridgeProvider *BridgeProviderTransactor) RelDeposit(opts *bind.TransactOpts, account common.Address) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "relDeposit", account)
}

// RelDeposit is a paid mutator transaction binding the contract method 0x712681cf.
//
// Solidity: function relDeposit(address account) returns()
func (_BridgeProvider *BridgeProviderSession) RelDeposit(account common.Address) (*types.Transaction, error) {
	return _BridgeProvider.Contract.RelDeposit(&_BridgeProvider.TransactOpts, account)
}

// RelDeposit is a paid mutator transaction binding the contract method 0x712681cf.
//
// Solidity: function relDeposit(address account) returns()
func (_BridgeProvider *BridgeProviderTransactorSession) RelDeposit(account common.Address) (*types.Transaction, error) {
	return _BridgeProvider.Contract.RelDeposit(&_BridgeProvider.TransactOpts, account)
}

// RelDeposits is a paid mutator transaction binding the contract method 0xd09d490e.
//
// Solidity: function relDeposits() returns()
func (_BridgeProvider *BridgeProviderTransactor) RelDeposits(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "relDeposits")
}

// RelDeposits is a paid mutator transaction binding the contract method 0xd09d490e.
//
// Solidity: function relDeposits() returns()
func (_BridgeProvider *BridgeProviderSession) RelDeposits() (*types.Transaction, error) {
	return _BridgeProvider.Contract.RelDeposits(&_BridgeProvider.TransactOpts)
}

// RelDeposits is a paid mutator transaction binding the contract method 0xd09d490e.
//
// Solidity: function relDeposits() returns()
func (_BridgeProvider *BridgeProviderTransactorSession) RelDeposits() (*types.Transaction, error) {
	return _BridgeProvider.Contract.RelDeposits(&_BridgeProvider.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_BridgeProvider *BridgeProviderTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_BridgeProvider *BridgeProviderSession) RenounceOwnership() (*types.Transaction, error) {
	return _BridgeProvider.Contract.RenounceOwnership(&_BridgeProvider.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_BridgeProvider *BridgeProviderTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _BridgeProvider.Contract.RenounceOwnership(&_BridgeProvider.TransactOpts)
}

// SetApiEndpoint is a paid mutator transaction binding the contract method 0x0ab0e54f.
//
// Solidity: function setApiEndpoint(string a) returns()
func (_BridgeProvider *BridgeProviderTransactor) SetApiEndpoint(opts *bind.TransactOpts, a string) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "setApiEndpoint", a)
}

// SetApiEndpoint is a paid mutator transaction binding the contract method 0x0ab0e54f.
//
// Solidity: function setApiEndpoint(string a) returns()
func (_BridgeProvider *BridgeProviderSession) SetApiEndpoint(a string) (*types.Transaction, error) {
	return _BridgeProvider.Contract.SetApiEndpoint(&_BridgeProvider.TransactOpts, a)
}

// SetApiEndpoint is a paid mutator transaction binding the contract method 0x0ab0e54f.
//
// Solidity: function setApiEndpoint(string a) returns()
func (_BridgeProvider *BridgeProviderTransactorSession) SetApiEndpoint(a string) (*types.Transaction, error) {
	return _BridgeProvider.Contract.SetApiEndpoint(&_BridgeProvider.TransactOpts, a)
}

// SetProviderProportion is a paid mutator transaction binding the contract method 0xbb10e226.
//
// Solidity: function setProviderProportion(uint256 p) returns()
func (_BridgeProvider *BridgeProviderTransactor) SetProviderProportion(opts *bind.TransactOpts, p *big.Int) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "setProviderProportion", p)
}

// SetProviderProportion is a paid mutator transaction binding the contract method 0xbb10e226.
//
// Solidity: function setProviderProportion(uint256 p) returns()
func (_BridgeProvider *BridgeProviderSession) SetProviderProportion(p *big.Int) (*types.Transaction, error) {
	return _BridgeProvider.Contract.SetProviderProportion(&_BridgeProvider.TransactOpts, p)
}

// SetProviderProportion is a paid mutator transaction binding the contract method 0xbb10e226.
//
// Solidity: function setProviderProportion(uint256 p) returns()
func (_BridgeProvider *BridgeProviderTransactorSession) SetProviderProportion(p *big.Int) (*types.Transaction, error) {
	return _BridgeProvider.Contract.SetProviderProportion(&_BridgeProvider.TransactOpts, p)
}

// SetSessionDivisor is a paid mutator transaction binding the contract method 0x38b6fff1.
//
// Solidity: function setSessionDivisor(uint256 m) returns()
func (_BridgeProvider *BridgeProviderTransactor) SetSessionDivisor(opts *bind.TransactOpts, m *big.Int) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "setSessionDivisor", m)
}

// SetSessionDivisor is a paid mutator transaction binding the contract method 0x38b6fff1.
//
// Solidity: function setSessionDivisor(uint256 m) returns()
func (_BridgeProvider *BridgeProviderSession) SetSessionDivisor(m *big.Int) (*types.Transaction, error) {
	return _BridgeProvider.Contract.SetSessionDivisor(&_BridgeProvider.TransactOpts, m)
}

// SetSessionDivisor is a paid mutator transaction binding the contract method 0x38b6fff1.
//
// Solidity: function setSessionDivisor(uint256 m) returns()
func (_BridgeProvider *BridgeProviderTransactorSession) SetSessionDivisor(m *big.Int) (*types.Transaction, error) {
	return _BridgeProvider.Contract.SetSessionDivisor(&_BridgeProvider.TransactOpts, m)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BridgeProvider *BridgeProviderTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _BridgeProvider.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BridgeProvider *BridgeProviderSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _BridgeProvider.Contract.TransferOwnership(&_BridgeProvider.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BridgeProvider *BridgeProviderTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _BridgeProvider.Contract.TransferOwnership(&_BridgeProvider.TransactOpts, newOwner)
}

// BridgeProviderAddDepositIterator is returned from FilterAddDeposit and is used to iterate over the raw logs and unpacked data for AddDeposit events raised by the BridgeProvider contract.
type BridgeProviderAddDepositIterator struct {
	Event *BridgeProviderAddDeposit // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *BridgeProviderAddDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeProviderAddDeposit)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(BridgeProviderAddDeposit)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *BridgeProviderAddDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeProviderAddDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeProviderAddDeposit represents a AddDeposit event raised by the BridgeProvider contract.
type BridgeProviderAddDeposit struct {
	Sender  common.Address
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterAddDeposit is a free log retrieval operation binding the contract event 0x0c7b2da39dfff790695e589229c5fd71ea2245de6cb792114a14009a0ad68c60.
//
// Solidity: event AddDeposit(address indexed sender, address indexed account, uint256 amount)
func (_BridgeProvider *BridgeProviderFilterer) FilterAddDeposit(opts *bind.FilterOpts, sender []common.Address, account []common.Address) (*BridgeProviderAddDepositIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _BridgeProvider.contract.FilterLogs(opts, "AddDeposit", senderRule, accountRule)
	if err != nil {
		return nil, err
	}
	return &BridgeProviderAddDepositIterator{contract: _BridgeProvider.contract, event: "AddDeposit", logs: logs, sub: sub}, nil
}

// WatchAddDeposit is a free log subscription operation binding the contract event 0x0c7b2da39dfff790695e589229c5fd71ea2245de6cb792114a14009a0ad68c60.
//
// Solidity: event AddDeposit(address indexed sender, address indexed account, uint256 amount)
func (_BridgeProvider *BridgeProviderFilterer) WatchAddDeposit(opts *bind.WatchOpts, sink chan<- *BridgeProviderAddDeposit, sender []common.Address, account []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _BridgeProvider.contract.WatchLogs(opts, "AddDeposit", senderRule, accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeProviderAddDeposit)
				if err := _BridgeProvider.contract.UnpackLog(event, "AddDeposit", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAddDeposit is a log parse operation binding the contract event 0x0c7b2da39dfff790695e589229c5fd71ea2245de6cb792114a14009a0ad68c60.
//
// Solidity: event AddDeposit(address indexed sender, address indexed account, uint256 amount)
func (_BridgeProvider *BridgeProviderFilterer) ParseAddDeposit(log types.Log) (*BridgeProviderAddDeposit, error) {
	event := new(BridgeProviderAddDeposit)
	if err := _BridgeProvider.contract.UnpackLog(event, "AddDeposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeProviderOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the BridgeProvider contract.
type BridgeProviderOwnershipTransferredIterator struct {
	Event *BridgeProviderOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *BridgeProviderOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeProviderOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(BridgeProviderOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *BridgeProviderOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeProviderOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeProviderOwnershipTransferred represents a OwnershipTransferred event raised by the BridgeProvider contract.
type BridgeProviderOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BridgeProvider *BridgeProviderFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*BridgeProviderOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BridgeProvider.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BridgeProviderOwnershipTransferredIterator{contract: _BridgeProvider.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BridgeProvider *BridgeProviderFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *BridgeProviderOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BridgeProvider.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeProviderOwnershipTransferred)
				if err := _BridgeProvider.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BridgeProvider *BridgeProviderFilterer) ParseOwnershipTransferred(log types.Log) (*BridgeProviderOwnershipTransferred, error) {
	event := new(BridgeProviderOwnershipTransferred)
	if err := _BridgeProvider.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeProviderRelDepositIterator is returned from FilterRelDeposit and is used to iterate over the raw logs and unpacked data for RelDeposit events raised by the BridgeProvider contract.
type BridgeProviderRelDepositIterator struct {
	Event *BridgeProviderRelDeposit // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *BridgeProviderRelDepositIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeProviderRelDeposit)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(BridgeProviderRelDeposit)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *BridgeProviderRelDepositIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeProviderRelDepositIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeProviderRelDeposit represents a RelDeposit event raised by the BridgeProvider contract.
type BridgeProviderRelDeposit struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRelDeposit is a free log retrieval operation binding the contract event 0x6d516f70e5f0a4474859e7b7b84a6fd56643b58b11981039137e4256f920f9eb.
//
// Solidity: event RelDeposit(address indexed account, uint256 amount)
func (_BridgeProvider *BridgeProviderFilterer) FilterRelDeposit(opts *bind.FilterOpts, account []common.Address) (*BridgeProviderRelDepositIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _BridgeProvider.contract.FilterLogs(opts, "RelDeposit", accountRule)
	if err != nil {
		return nil, err
	}
	return &BridgeProviderRelDepositIterator{contract: _BridgeProvider.contract, event: "RelDeposit", logs: logs, sub: sub}, nil
}

// WatchRelDeposit is a free log subscription operation binding the contract event 0x6d516f70e5f0a4474859e7b7b84a6fd56643b58b11981039137e4256f920f9eb.
//
// Solidity: event RelDeposit(address indexed account, uint256 amount)
func (_BridgeProvider *BridgeProviderFilterer) WatchRelDeposit(opts *bind.WatchOpts, sink chan<- *BridgeProviderRelDeposit, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _BridgeProvider.contract.WatchLogs(opts, "RelDeposit", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeProviderRelDeposit)
				if err := _BridgeProvider.contract.UnpackLog(event, "RelDeposit", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRelDeposit is a log parse operation binding the contract event 0x6d516f70e5f0a4474859e7b7b84a6fd56643b58b11981039137e4256f920f9eb.
//
// Solidity: event RelDeposit(address indexed account, uint256 amount)
func (_BridgeProvider *BridgeProviderFilterer) ParseRelDeposit(log types.Log) (*BridgeProviderRelDeposit, error) {
	event := new(BridgeProviderRelDeposit)
	if err := _BridgeProvider.contract.UnpackLog(event, "RelDeposit", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
