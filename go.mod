module github.com/evmos/ethermint

go 1.23.1

toolchain go1.23.2

require (
	//cosmossdk.io/core v0.12.1-0.20240725072823-6a2d039e1212
	cosmossdk.io/core v0.12.0
	//cosmossdk.io/core v0.11.1
	cosmossdk.io/errors v1.0.1
	cosmossdk.io/math v1.3.0
	//cosmossdk.io/simapp v0.0.0-20230608160436-666c345ad23d
	cosmossdk.io/simapp v0.0.0-20241008175849-325728a9fd6c
	cosmossdk.io/tools/rosetta v0.2.1
	github.com/armon/go-metrics v0.4.1
	github.com/btcsuite/btcd v0.24.2
	github.com/btcsuite/btcd/btcutil v1.1.6
	github.com/cometbft/cometbft v1.0.0-rc1.0.20240908111210-ab0be101882f
	//github.com/cometbft/cometbft-db v0.15.0
	github.com/cometbft/cometbft-db v0.11.0
	github.com/cosmos/cosmos-proto v1.0.0-beta.5
	github.com/cosmos/cosmos-sdk v0.53.0
	github.com/cosmos/go-bip39 v1.0.0
	github.com/cosmos/gogoproto v1.7.0
	github.com/cosmos/ibc-go/v8 v8.5.1
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/ethereum/go-ethereum v1.10.26
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/holiman/uint256 v1.2.1
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/onsi/ginkgo/v2 v2.9.1
	github.com/onsi/gomega v1.27.4
	github.com/pkg/errors v0.9.1
	github.com/rakyll/statik v0.1.7
	github.com/rs/cors v1.11.1
	github.com/spf13/cast v1.7.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/viper v1.19.0
	github.com/status-im/keycard-go v0.2.0
	github.com/stretchr/testify v1.9.0
	github.com/tyler-smith/go-bip39 v1.1.0
	golang.org/x/net v0.29.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240814211410-ddb44dafa142
	google.golang.org/grpc v1.67.1
	google.golang.org/protobuf v1.35.1
	sigs.k8s.io/yaml v1.4.0
)

require (
	cloud.google.com/go v0.115.1 // indirect
	cloud.google.com/go/compute v1.27.4 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	cloud.google.com/go/iam v1.1.13 // indirect
	cloud.google.com/go/storage v1.43.0 // indirect
	cosmossdk.io/api v0.7.6 // indirect
	//cosmossdk.io/core v1.0.0-alpha.4 // indirect
	cosmossdk.io/depinject v1.0.0 // indirect
	cosmossdk.io/log v1.4.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/ChainSafe/go-schnorrkel v1.0.0 // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/VictoriaMetrics/fastcache v1.6.0 // indirect
	github.com/aws/aws-sdk-go v1.55.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/bgentry/speakeasy v0.2.0 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cockroachdb/apd/v2 v2.0.2 // indirect
	github.com/cockroachdb/errors v1.11.3 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/coinbase/rosetta-sdk-go v0.7.9 // indirect
	github.com/confio/ics23/go v0.9.0 // indirect
	github.com/cosmos/btcutil v1.0.5 // indirect
	github.com/cosmos/gogogateway v1.2.0 // indirect
	github.com/cosmos/iavl v1.3.0 // indirect
	github.com/cosmos/ics23/go v0.11.0 // indirect
	github.com/cosmos/ledger-cosmos-go v0.13.3 // indirect
	github.com/cosmos/rosetta-sdk-go v0.10.0 // indirect
	github.com/creachadair/taskgroup v0.4.2 // indirect
	github.com/danieljoos/wincred v1.2.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/dgraph-io/badger/v2 v2.2007.4 // indirect
	github.com/dgraph-io/ristretto v0.1.2-0.20240116140435-c67e07994f91 // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/dlclark/regexp2 v1.4.1-0.20201116162257-a2a8dda75c91 // indirect
	github.com/dop251/goja v0.0.0-20220405120441-9037c2b61cbf // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.6.0 // indirect
	github.com/edsrzf/mmap-go v1.0.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gballet/go-libpcsclite v0.0.0-20190607065134-2772fd86a8ff // indirect
	github.com/getsentry/sentry-go v0.27.0 // indirect
	github.com/go-kit/kit v0.13.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang/glog v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/orderedcode v0.0.1 // indirect
	github.com/google/pprof v0.0.0-20230228050547-1710fef4ab10 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/gtank/merlin v0.1.1 // indirect
	github.com/gtank/ristretto255 v0.1.2 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-getter v1.7.6 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-safetemp v1.0.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hdevalence/ed25519consensus v0.2.0 // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/huandu/skiplist v1.2.1 // indirect
	github.com/huin/goupnp v1.0.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/linxGnu/grocksdb v1.9.3 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/manifoldco/promptui v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mimoo/StrobeGo v0.0.0-20210601165009-122bf33a46e0 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/petermattis/goid v0.0.0-20240813172612-4fcff4a6cae7 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.20.4 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.60.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rjeczalik/notify v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	github.com/sasha-s/go-deadlock v0.3.5 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d // indirect
	github.com/tendermint/go-amino v0.16.0 // indirect
	github.com/tidwall/btree v1.7.0 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/ulikunitz/xz v0.5.12 // indirect
	github.com/zondax/hid v0.9.2 // indirect
	github.com/zondax/ledger-go v0.14.3 // indirect
	go.etcd.io/bbolt v1.4.0-alpha.0.0.20240404170359-43604f3112c5 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8 // indirect
	golang.org/x/oauth2 v0.23.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/term v0.25.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	google.golang.org/api v0.192.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240930140551-af27646dc61f // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.6 // indirect
	pgregory.net/rapid v1.1.0 // indirect
)

replace (
	// Fix upstream GHSA-h395-qcrw-5vmq vulnerability.
	// TODO Remove it: https://github.com/cosmos/cosmos-sdk/issues/10409

	cosmossdk.io/simapp => ./simapp
	cosmossdk.io/simapp/params => /simapp/params
	// use cosmos keyring
	github.com/99designs/keyring => github.com/cosmos/keyring v1.2.0
	// Use cosmos-sdk fork with staking transfer events, and custom tally handler support
	//github.com/cosmos/cosmos-sdk => github.com/kava-labs/cosmos-sdk v0.47.10-kava.1

	github.com/cometbft/cometbft => github.com/kava-labs/cometbft v0.0.0-20241007151334-8345af773eb9
	//github.com/cometbft/cometbft-db => github.com/kava-labs/cometbft-db kava-patched-test b2740b2e4bed1112feb4157b154ce969759b52ce
	//github.com/cometbft/cometbft-db => github.com/kava-labs/cometbft-db v0.0.0-20241007145430-b2740b2e4bed
	// Use cosmos-sdk fork with backported fix for unsafe-reset-all, staking transfer events, and custom tally handler support
	//github.com/cosmos/cosmos-sdk => github.com/kava-labs/cosmos-sdk v0.47.10-iavl-v1-kava.1
	//github.com/cosmos/cosmos-sdk => github.com/kava-labs/cosmos-sdk v0.50.10-test-patch 5f9239e3147358ef034bfc4d19aacb34e5ea2064
	github.com/cosmos/cosmos-sdk => github.com/kava-labs/cosmos-sdk v0.0.0-20241007153930-5f9239e31473

	github.com/cosmos/cosmos-sdk/x/evidence => cosmossdk.io/x/evidence v0.1.1
	github.com/cosmos/cosmos-sdk/x/feegrant => cosmossdk.io/x/feegrant v0.1.1
	github.com/cosmos/cosmos-sdk/x/upgrade => cosmossdk.io/x/upgrade v0.1.4

	github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.9.0
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
)
