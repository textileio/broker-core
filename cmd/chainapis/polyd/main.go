package main

import (
	"github.com/textileio/broker-core/cmd/chainapis/ethshared"
	"github.com/textileio/cli"
)

func main() {
	cli.CheckErrf("executing root cmd: %v", ethshared.BuildRootCmd("polyd", "POLY", "polygon").Execute())
}
