package common

import "github.com/filecoin-project/go-address"

// StringifyAddrs stringifies a slice of address.Address into
// mainnet-prefixed strings.
func StringifyAddrs(addrs ...address.Address) []string {
	addrsStr := make([]string, len(addrs))
	for i := range addrs {
		addrsStr[i] = StringifyAddr(addrs[i])
	}
	return addrsStr
}

// StringifyAddr stringifies an address.Address into a mainnet-prefixed string.
// address.String() always returns as if the address is for testnet.
// We force replacing it to be mainnet.
func StringifyAddr(addr address.Address) string {
	return address.MainnetPrefix + addr.String()[len(address.MainnetPrefix):]
}
