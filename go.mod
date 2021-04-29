module github.com/textileio/broker-core

go 1.16

require (
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/ethereum/go-ethereum v1.10.2
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-cbor-util v0.0.0-20191219014500-08c40a1e63a2
	github.com/filecoin-project/go-fil-commcid v0.1.0
	github.com/filecoin-project/go-fil-commp-hashhash v0.1.0
	github.com/filecoin-project/go-fil-markets v1.2.5
	github.com/filecoin-project/go-state-types v0.1.0
	github.com/filecoin-project/lotus v1.8.0
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/google/uuid v1.2.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hsanjuan/ipfs-lite v1.1.19
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-http-client v0.1.0
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/ipfs/go-merkledag v0.3.2
	github.com/ipfs/go-unixfs v0.2.4
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/ipld/go-car v0.1.1-0.20201119040415-11b6074b6d4d
	github.com/libp2p/go-libp2p v0.13.0
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-libp2p-peerstore v0.2.6
	github.com/libp2p/go-libp2p-protocol v0.1.0
	github.com/libp2p/go-libp2p-pubsub v0.4.2-0.20210212194758-6c1addf493eb
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.15
	github.com/multiformats/go-varint v0.0.6
	github.com/near/borsh-go v0.3.0
	github.com/ockam-network/did v0.1.3
	github.com/oklog/ulid/v2 v2.0.2
	github.com/ory/dockertest/v3 v3.6.3
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/textileio/go-datastore-extensions v1.0.1
	github.com/textileio/go-ds-badger3 v0.0.0-20210324034212-7b7fb3be3d1c
	github.com/textileio/go-ds-mongo v0.1.4
	github.com/textileio/jwt-go-eddsa v0.2.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.19.0
	go.opentelemetry.io/contrib/instrumentation/runtime v0.19.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.19.0
	go.opentelemetry.io/otel/metric v0.19.0
	go.uber.org/zap v1.16.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
)

replace github.com/ethereum/go-ethereum => github.com/textileio/go-ethereum v1.10.3-0.20210413172519-62e8b38d82b1

replace github.com/filecoin-project/filecoin-ffi => ../lotus/extern/filecoin-ffi
