module github.com/textileio/broker-core

go 1.16

require (
	cloud.google.com/go/pubsub v1.16.0
	cloud.google.com/go/storage v1.16.1
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/ethereum/go-ethereum v1.10.8
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-cbor-util v0.0.0-20191219014500-08c40a1e63a2
	github.com/filecoin-project/go-dagaggregator-unixfs v0.3.0
	github.com/filecoin-project/go-fil-commcid v0.1.0
	github.com/filecoin-project/go-fil-commp-hashhash v0.1.0
	github.com/filecoin-project/go-fil-markets v1.12.0
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210810190654-139e0e79e69e
	github.com/filecoin-project/lotus v1.11.3
	github.com/filecoin-project/specs-actors v0.9.14
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang-migrate/migrate/v4 v4.14.1
	github.com/google/uuid v1.3.0
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.4.6
	github.com/ipfs/go-ipfs-cmds v0.6.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-http-client v0.1.0
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-unixfs v0.2.6
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/ipld/go-car v0.3.1-0.20210601190600-f512dac51e8e
	github.com/jackc/pgconn v1.10.0
	github.com/jackc/pgx/v4 v4.13.0
	github.com/joho/godotenv v1.3.0
	github.com/jsign/go-filsigner v0.3.1
	github.com/lib/pq v1.10.2
	github.com/libp2p/go-libp2p v0.15.1
	github.com/libp2p/go-libp2p-circuit v0.4.0
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.9.0
	github.com/libp2p/go-libp2p-swarm v0.5.3
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multiaddr v0.4.0
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.16
	github.com/oklog/ulid/v2 v2.0.2
	github.com/ory/dockertest/v3 v3.7.0
	github.com/shirou/gopsutil v3.21.7+incompatible // indirect
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/textileio/bidbot v0.1.1-0.20211001192524-0dc443db44a5
	github.com/textileio/cli v1.0.1
	github.com/textileio/crypto v0.0.0-20210929130053-08edebc3361a
	github.com/textileio/go-auctions-client v0.0.0-20210914093526-52fac6d8b09f
	github.com/textileio/go-datastore-extensions v1.0.1
	github.com/textileio/go-libp2p-pubsub-rpc v0.0.5
	github.com/textileio/go-log/v2 v2.1.3-gke-2
	github.com/textileio/near-api-go v0.2.0
	github.com/tklauser/go-sysconf v0.3.8 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.22.0
	go.opentelemetry.io/contrib/instrumentation/runtime v0.21.0
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/prometheus v0.21.0
	go.opentelemetry.io/otel/metric v0.22.0
	go.opentelemetry.io/otel/sdk/export/metric v0.21.0
	go.opentelemetry.io/otel/sdk/metric v0.21.0
	go.uber.org/atomic v1.9.0 // indirect
	google.golang.org/api v0.56.0
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/kilic/bls12-381 => github.com/kilic/bls12-381 v0.0.0-20200820230200-6b2c19996391

replace github.com/ethereum/go-ethereum => github.com/textileio/go-ethereum v1.10.3-0.20210413172519-62e8b38d82b1

replace github.com/ipfs/go-ipns => github.com/ipfs/go-ipns v0.0.2

// add status code to HTTP request metrics, see https://github.com/open-telemetry/opentelemetry-go-contrib/pull/771
replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => github.com/nabokihms/opentelemetry-go-contrib/instrumentation/net/http/otelhttp v0.20.1-0.20210622062648-f4d780a56f54
