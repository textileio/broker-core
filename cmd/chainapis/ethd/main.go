package main

import (
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared"
)

func main() {
	common.CheckErrf("executing root cmd: %v", ethshared.BuildRootCmd("ethd", "ETH", "ethereum").Execute())
}
