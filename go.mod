module github.com/textileio/broker-core

go 1.16

require (
	cloud.google.com/go/pubsub v1.13.0
	cloud.google.com/go/storage v1.16.0
	github.com/dustin/go-humanize v1.0.0
	github.com/ethereum/go-ethereum v1.10.6
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-cbor-util v0.0.0-20191219014500-08c40a1e63a2
	github.com/filecoin-project/go-dagaggregator-unixfs v0.2.0
	github.com/filecoin-project/go-fil-commcid v0.1.0
	github.com/filecoin-project/go-fil-commp-hashhash v0.1.0
	github.com/filecoin-project/go-fil-markets v1.5.0
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/lotus v1.11.0
	github.com/filecoin-project/specs-actors v0.9.14
	github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/google/uuid v1.3.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.6
	github.com/ipfs/go-ipfs-cmds v0.6.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-http-client v0.1.0
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-unixfs v0.2.6
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/ipld/go-car v0.1.1-0.20201119040415-11b6074b6d4d
	github.com/jackc/pgconn v1.10.0
	github.com/jackc/pgx/v4 v4.12.0
	github.com/joho/godotenv v1.3.0
	github.com/jsign/go-filsigner v0.2.0
	github.com/lib/pq v1.10.2
	github.com/libp2p/go-libp2p v0.14.3
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.15
	github.com/oklog/ulid/v2 v2.0.2
	github.com/ory/dockertest/v3 v3.7.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/textileio/bidbot v0.0.5-0.20210812152825-8b6499ad10ca
	github.com/textileio/go-datastore-extensions v1.0.1
	github.com/textileio/go-ds-badger3 v0.0.0-20210324034212-7b7fb3be3d1c
	github.com/textileio/go-libp2p-pubsub-rpc v0.0.2
	github.com/textileio/go-log/v2 v2.1.3-gke-1
	github.com/textileio/jwt-go-eddsa v0.2.1
	github.com/textileio/near-api-go v0.1.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.21.0
	go.opentelemetry.io/otel v1.0.0-RC1
	go.opentelemetry.io/otel/metric v0.21.0
	google.golang.org/api v0.50.0
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/kilic/bls12-381 => github.com/kilic/bls12-381 v0.0.0-20200820230200-6b2c19996391

replace github.com/ethereum/go-ethereum => github.com/textileio/go-ethereum v1.10.3-0.20210413172519-62e8b38d82b1

replace github.com/ipfs/go-ipns => github.com/ipfs/go-ipns v0.0.2
