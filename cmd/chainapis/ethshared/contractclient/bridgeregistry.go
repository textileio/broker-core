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

// BridgeRegistryABI is the input ABI used to generate the binding from.
const BridgeRegistryABI = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"AddProvider\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"RemoveProvider\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"addProvider\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"listProviders\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"removeProvider\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

// BridgeRegistry is an auto generated Go binding around an Ethereum contract.
type BridgeRegistry struct {
	BridgeRegistryCaller     // Read-only binding to the contract
	BridgeRegistryTransactor // Write-only binding to the contract
	BridgeRegistryFilterer   // Log filterer for contract events
}

// BridgeRegistryCaller is an auto generated read-only Go binding around an Ethereum contract.
type BridgeRegistryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeRegistryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type BridgeRegistryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeRegistryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type BridgeRegistryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// BridgeRegistrySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type BridgeRegistrySession struct {
	Contract     *BridgeRegistry   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// BridgeRegistryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type BridgeRegistryCallerSession struct {
	Contract *BridgeRegistryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// BridgeRegistryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type BridgeRegistryTransactorSession struct {
	Contract     *BridgeRegistryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// BridgeRegistryRaw is an auto generated low-level Go binding around an Ethereum contract.
type BridgeRegistryRaw struct {
	Contract *BridgeRegistry // Generic contract binding to access the raw methods on
}

// BridgeRegistryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type BridgeRegistryCallerRaw struct {
	Contract *BridgeRegistryCaller // Generic read-only contract binding to access the raw methods on
}

// BridgeRegistryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type BridgeRegistryTransactorRaw struct {
	Contract *BridgeRegistryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewBridgeRegistry creates a new instance of BridgeRegistry, bound to a specific deployed contract.
func NewBridgeRegistry(address common.Address, backend bind.ContractBackend) (*BridgeRegistry, error) {
	contract, err := bindBridgeRegistry(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &BridgeRegistry{BridgeRegistryCaller: BridgeRegistryCaller{contract: contract}, BridgeRegistryTransactor: BridgeRegistryTransactor{contract: contract}, BridgeRegistryFilterer: BridgeRegistryFilterer{contract: contract}}, nil
}

// NewBridgeRegistryCaller creates a new read-only instance of BridgeRegistry, bound to a specific deployed contract.
func NewBridgeRegistryCaller(address common.Address, caller bind.ContractCaller) (*BridgeRegistryCaller, error) {
	contract, err := bindBridgeRegistry(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeRegistryCaller{contract: contract}, nil
}

// NewBridgeRegistryTransactor creates a new write-only instance of BridgeRegistry, bound to a specific deployed contract.
func NewBridgeRegistryTransactor(address common.Address, transactor bind.ContractTransactor) (*BridgeRegistryTransactor, error) {
	contract, err := bindBridgeRegistry(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &BridgeRegistryTransactor{contract: contract}, nil
}

// NewBridgeRegistryFilterer creates a new log filterer instance of BridgeRegistry, bound to a specific deployed contract.
func NewBridgeRegistryFilterer(address common.Address, filterer bind.ContractFilterer) (*BridgeRegistryFilterer, error) {
	contract, err := bindBridgeRegistry(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &BridgeRegistryFilterer{contract: contract}, nil
}

// bindBridgeRegistry binds a generic wrapper to an already deployed contract.
func bindBridgeRegistry(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(BridgeRegistryABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeRegistry *BridgeRegistryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeRegistry.Contract.BridgeRegistryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeRegistry *BridgeRegistryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.BridgeRegistryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeRegistry *BridgeRegistryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.BridgeRegistryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_BridgeRegistry *BridgeRegistryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _BridgeRegistry.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_BridgeRegistry *BridgeRegistryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_BridgeRegistry *BridgeRegistryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.contract.Transact(opts, method, params...)
}

// ListProviders is a free data retrieval call binding the contract method 0x2bd395a0.
//
// Solidity: function listProviders() view returns(address[])
func (_BridgeRegistry *BridgeRegistryCaller) ListProviders(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _BridgeRegistry.contract.Call(opts, &out, "listProviders")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// ListProviders is a free data retrieval call binding the contract method 0x2bd395a0.
//
// Solidity: function listProviders() view returns(address[])
func (_BridgeRegistry *BridgeRegistrySession) ListProviders() ([]common.Address, error) {
	return _BridgeRegistry.Contract.ListProviders(&_BridgeRegistry.CallOpts)
}

// ListProviders is a free data retrieval call binding the contract method 0x2bd395a0.
//
// Solidity: function listProviders() view returns(address[])
func (_BridgeRegistry *BridgeRegistryCallerSession) ListProviders() ([]common.Address, error) {
	return _BridgeRegistry.Contract.ListProviders(&_BridgeRegistry.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BridgeRegistry *BridgeRegistryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _BridgeRegistry.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BridgeRegistry *BridgeRegistrySession) Owner() (common.Address, error) {
	return _BridgeRegistry.Contract.Owner(&_BridgeRegistry.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_BridgeRegistry *BridgeRegistryCallerSession) Owner() (common.Address, error) {
	return _BridgeRegistry.Contract.Owner(&_BridgeRegistry.CallOpts)
}

// AddProvider is a paid mutator transaction binding the contract method 0x46e2577a.
//
// Solidity: function addProvider(address provider) returns()
func (_BridgeRegistry *BridgeRegistryTransactor) AddProvider(opts *bind.TransactOpts, provider common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.contract.Transact(opts, "addProvider", provider)
}

// AddProvider is a paid mutator transaction binding the contract method 0x46e2577a.
//
// Solidity: function addProvider(address provider) returns()
func (_BridgeRegistry *BridgeRegistrySession) AddProvider(provider common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.AddProvider(&_BridgeRegistry.TransactOpts, provider)
}

// AddProvider is a paid mutator transaction binding the contract method 0x46e2577a.
//
// Solidity: function addProvider(address provider) returns()
func (_BridgeRegistry *BridgeRegistryTransactorSession) AddProvider(provider common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.AddProvider(&_BridgeRegistry.TransactOpts, provider)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_BridgeRegistry *BridgeRegistryTransactor) Initialize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeRegistry.contract.Transact(opts, "initialize")
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_BridgeRegistry *BridgeRegistrySession) Initialize() (*types.Transaction, error) {
	return _BridgeRegistry.Contract.Initialize(&_BridgeRegistry.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_BridgeRegistry *BridgeRegistryTransactorSession) Initialize() (*types.Transaction, error) {
	return _BridgeRegistry.Contract.Initialize(&_BridgeRegistry.TransactOpts)
}

// RemoveProvider is a paid mutator transaction binding the contract method 0x8a355a57.
//
// Solidity: function removeProvider(address provider) returns()
func (_BridgeRegistry *BridgeRegistryTransactor) RemoveProvider(opts *bind.TransactOpts, provider common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.contract.Transact(opts, "removeProvider", provider)
}

// RemoveProvider is a paid mutator transaction binding the contract method 0x8a355a57.
//
// Solidity: function removeProvider(address provider) returns()
func (_BridgeRegistry *BridgeRegistrySession) RemoveProvider(provider common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.RemoveProvider(&_BridgeRegistry.TransactOpts, provider)
}

// RemoveProvider is a paid mutator transaction binding the contract method 0x8a355a57.
//
// Solidity: function removeProvider(address provider) returns()
func (_BridgeRegistry *BridgeRegistryTransactorSession) RemoveProvider(provider common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.RemoveProvider(&_BridgeRegistry.TransactOpts, provider)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_BridgeRegistry *BridgeRegistryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _BridgeRegistry.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_BridgeRegistry *BridgeRegistrySession) RenounceOwnership() (*types.Transaction, error) {
	return _BridgeRegistry.Contract.RenounceOwnership(&_BridgeRegistry.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_BridgeRegistry *BridgeRegistryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _BridgeRegistry.Contract.RenounceOwnership(&_BridgeRegistry.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BridgeRegistry *BridgeRegistryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BridgeRegistry *BridgeRegistrySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.TransferOwnership(&_BridgeRegistry.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_BridgeRegistry *BridgeRegistryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _BridgeRegistry.Contract.TransferOwnership(&_BridgeRegistry.TransactOpts, newOwner)
}

// BridgeRegistryAddProviderIterator is returned from FilterAddProvider and is used to iterate over the raw logs and unpacked data for AddProvider events raised by the BridgeRegistry contract.
type BridgeRegistryAddProviderIterator struct {
	Event *BridgeRegistryAddProvider // Event containing the contract specifics and raw log

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
func (it *BridgeRegistryAddProviderIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeRegistryAddProvider)
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
		it.Event = new(BridgeRegistryAddProvider)
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
func (it *BridgeRegistryAddProviderIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeRegistryAddProviderIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeRegistryAddProvider represents a AddProvider event raised by the BridgeRegistry contract.
type BridgeRegistryAddProvider struct {
	Provider common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterAddProvider is a free log retrieval operation binding the contract event 0x635a554d7028e977037c71e4fefb0d02f19e160c13f01f813a18d362b4605c6d.
//
// Solidity: event AddProvider(address indexed provider)
func (_BridgeRegistry *BridgeRegistryFilterer) FilterAddProvider(opts *bind.FilterOpts, provider []common.Address) (*BridgeRegistryAddProviderIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _BridgeRegistry.contract.FilterLogs(opts, "AddProvider", providerRule)
	if err != nil {
		return nil, err
	}
	return &BridgeRegistryAddProviderIterator{contract: _BridgeRegistry.contract, event: "AddProvider", logs: logs, sub: sub}, nil
}

// WatchAddProvider is a free log subscription operation binding the contract event 0x635a554d7028e977037c71e4fefb0d02f19e160c13f01f813a18d362b4605c6d.
//
// Solidity: event AddProvider(address indexed provider)
func (_BridgeRegistry *BridgeRegistryFilterer) WatchAddProvider(opts *bind.WatchOpts, sink chan<- *BridgeRegistryAddProvider, provider []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _BridgeRegistry.contract.WatchLogs(opts, "AddProvider", providerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeRegistryAddProvider)
				if err := _BridgeRegistry.contract.UnpackLog(event, "AddProvider", log); err != nil {
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

// ParseAddProvider is a log parse operation binding the contract event 0x635a554d7028e977037c71e4fefb0d02f19e160c13f01f813a18d362b4605c6d.
//
// Solidity: event AddProvider(address indexed provider)
func (_BridgeRegistry *BridgeRegistryFilterer) ParseAddProvider(log types.Log) (*BridgeRegistryAddProvider, error) {
	event := new(BridgeRegistryAddProvider)
	if err := _BridgeRegistry.contract.UnpackLog(event, "AddProvider", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeRegistryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the BridgeRegistry contract.
type BridgeRegistryOwnershipTransferredIterator struct {
	Event *BridgeRegistryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *BridgeRegistryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeRegistryOwnershipTransferred)
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
		it.Event = new(BridgeRegistryOwnershipTransferred)
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
func (it *BridgeRegistryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeRegistryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeRegistryOwnershipTransferred represents a OwnershipTransferred event raised by the BridgeRegistry contract.
type BridgeRegistryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BridgeRegistry *BridgeRegistryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*BridgeRegistryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BridgeRegistry.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &BridgeRegistryOwnershipTransferredIterator{contract: _BridgeRegistry.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_BridgeRegistry *BridgeRegistryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *BridgeRegistryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _BridgeRegistry.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeRegistryOwnershipTransferred)
				if err := _BridgeRegistry.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_BridgeRegistry *BridgeRegistryFilterer) ParseOwnershipTransferred(log types.Log) (*BridgeRegistryOwnershipTransferred, error) {
	event := new(BridgeRegistryOwnershipTransferred)
	if err := _BridgeRegistry.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// BridgeRegistryRemoveProviderIterator is returned from FilterRemoveProvider and is used to iterate over the raw logs and unpacked data for RemoveProvider events raised by the BridgeRegistry contract.
type BridgeRegistryRemoveProviderIterator struct {
	Event *BridgeRegistryRemoveProvider // Event containing the contract specifics and raw log

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
func (it *BridgeRegistryRemoveProviderIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(BridgeRegistryRemoveProvider)
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
		it.Event = new(BridgeRegistryRemoveProvider)
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
func (it *BridgeRegistryRemoveProviderIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *BridgeRegistryRemoveProviderIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// BridgeRegistryRemoveProvider represents a RemoveProvider event raised by the BridgeRegistry contract.
type BridgeRegistryRemoveProvider struct {
	Provider common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterRemoveProvider is a free log retrieval operation binding the contract event 0x8ab468b9f8f57e82f33d9a1742c03768ff84410a4911e8647bfa641826876904.
//
// Solidity: event RemoveProvider(address indexed provider)
func (_BridgeRegistry *BridgeRegistryFilterer) FilterRemoveProvider(opts *bind.FilterOpts, provider []common.Address) (*BridgeRegistryRemoveProviderIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _BridgeRegistry.contract.FilterLogs(opts, "RemoveProvider", providerRule)
	if err != nil {
		return nil, err
	}
	return &BridgeRegistryRemoveProviderIterator{contract: _BridgeRegistry.contract, event: "RemoveProvider", logs: logs, sub: sub}, nil
}

// WatchRemoveProvider is a free log subscription operation binding the contract event 0x8ab468b9f8f57e82f33d9a1742c03768ff84410a4911e8647bfa641826876904.
//
// Solidity: event RemoveProvider(address indexed provider)
func (_BridgeRegistry *BridgeRegistryFilterer) WatchRemoveProvider(opts *bind.WatchOpts, sink chan<- *BridgeRegistryRemoveProvider, provider []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}

	logs, sub, err := _BridgeRegistry.contract.WatchLogs(opts, "RemoveProvider", providerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(BridgeRegistryRemoveProvider)
				if err := _BridgeRegistry.contract.UnpackLog(event, "RemoveProvider", log); err != nil {
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

// ParseRemoveProvider is a log parse operation binding the contract event 0x8ab468b9f8f57e82f33d9a1742c03768ff84410a4911e8647bfa641826876904.
//
// Solidity: event RemoveProvider(address indexed provider)
func (_BridgeRegistry *BridgeRegistryFilterer) ParseRemoveProvider(log types.Log) (*BridgeRegistryRemoveProvider, error) {
	event := new(BridgeRegistryRemoveProvider)
	if err := _BridgeRegistry.contract.UnpackLog(event, "RemoveProvider", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
