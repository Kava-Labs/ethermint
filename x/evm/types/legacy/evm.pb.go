package types

import (
	sdkmath "cosmossdk.io/math"
	"github.com/evmos/ethermint/x/evm/types"
)

//
// LegacyParams and LegacyChainConfig are only used in
// the v3 store migration and fetching parameters for old blocks.
//
// Since the paramstore operates with Amino, we do not need full protobuf
// definitions since legacy param usage not propogated to methods that use
// protobuf serialization.  In addition, since these do not implement interfaces
// used in decoding, we do not need to register these types on any codecs.
//

// LegacyParams defines the EVM module parameters
type LegacyParams struct {
	// evm denom represents the token denomination used to run the EVM state
	// transitions.
	EvmDenom string `protobuf:"bytes,1,opt,name=evm_denom,json=evmDenom,proto3" json:"evm_denom,omitempty" yaml:"evm_denom"`
	// enable create toggles state transitions that use the vm.Create function
	EnableCreate bool `protobuf:"varint,2,opt,name=enable_create,json=enableCreate,proto3" json:"enable_create,omitempty" yaml:"enable_create"`
	// enable call toggles state transitions that use the vm.Call function
	EnableCall bool `protobuf:"varint,3,opt,name=enable_call,json=enableCall,proto3" json:"enable_call,omitempty" yaml:"enable_call"`
	// extra eips defines the additional EIPs for the vm.Config
	ExtraEIPs []int64 `protobuf:"varint,4,rep,packed,name=extra_eips,json=extraEips,proto3" json:"extra_eips,omitempty" yaml:"extra_eips"`
	// chain config defines the EVM chain configuration parameters
	ChainConfig LegacyChainConfig `protobuf:"bytes,5,opt,name=chain_config,json=chainConfig,proto3" json:"chain_config" yaml:"chain_config"`
	// list of allowed eip712 msgs and their types
	EIP712AllowedMsgs []types.EIP712AllowedMsg `protobuf:"bytes,6,rep,name=eip712_allowed_msgs,json=eip712AllowedMsgs,proto3" json:"eip712_allowed_msgs"`
}

// LegacyChainConfig defines the Ethereum ChainConfig parameters using *sdkmath.Int values
// instead of *big.Int.
type LegacyChainConfig struct {
	// Homestead switch block (nil no fork, 0 = already homestead)
	HomesteadBlock *sdkmath.Int `protobuf:"bytes,1,opt,name=homestead_block,json=homesteadBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"homestead_block,omitempty" yaml:"homestead_block"`
	// TheDAO hard-fork switch block (nil no fork)
	DAOForkBlock *sdkmath.Int `protobuf:"bytes,2,opt,name=dao_fork_block,json=daoForkBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"dao_fork_block,omitempty" yaml:"dao_fork_block"`
	// Whether the nodes supports or opposes the DAO hard-fork
	DAOForkSupport bool `protobuf:"varint,3,opt,name=dao_fork_support,json=daoForkSupport,proto3" json:"dao_fork_support,omitempty" yaml:"dao_fork_support"`
	// EIP150 implements the Gas price changes
	// (https://github.com/ethereum/EIPs/issues/150) EIP150 HF block (nil no fork)
	EIP150Block *sdkmath.Int `protobuf:"bytes,4,opt,name=eip150_block,json=eip150Block,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"eip150_block,omitempty" yaml:"eip150_block"`
	// EIP150 HF hash (needed for header only clients as only gas pricing changed)
	EIP150Hash string `protobuf:"bytes,5,opt,name=eip150_hash,json=eip150Hash,proto3" json:"eip150_hash,omitempty" yaml:"byzantium_block"`
	// EIP155Block HF block
	EIP155Block *sdkmath.Int `protobuf:"bytes,6,opt,name=eip155_block,json=eip155Block,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"eip155_block,omitempty" yaml:"eip155_block"`
	// EIP158 HF block
	EIP158Block *sdkmath.Int `protobuf:"bytes,7,opt,name=eip158_block,json=eip158Block,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"eip158_block,omitempty" yaml:"eip158_block"`
	// Byzantium switch block (nil no fork, 0 = already on byzantium)
	ByzantiumBlock *sdkmath.Int `protobuf:"bytes,8,opt,name=byzantium_block,json=byzantiumBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"byzantium_block,omitempty" yaml:"byzantium_block"`
	// Constantinople switch block (nil no fork, 0 = already activated)
	ConstantinopleBlock *sdkmath.Int `protobuf:"bytes,9,opt,name=constantinople_block,json=constantinopleBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"constantinople_block,omitempty" yaml:"constantinople_block"`
	// Petersburg switch block (nil same as Constantinople)
	PetersburgBlock *sdkmath.Int `protobuf:"bytes,10,opt,name=petersburg_block,json=petersburgBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"petersburg_block,omitempty" yaml:"petersburg_block"`
	// Istanbul switch block (nil no fork, 0 = already on istanbul)
	IstanbulBlock *sdkmath.Int `protobuf:"bytes,11,opt,name=istanbul_block,json=istanbulBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"istanbul_block,omitempty" yaml:"istanbul_block"`
	// Eip-2384 (bomb delay) switch block (nil no fork, 0 = already activated)
	MuirGlacierBlock *sdkmath.Int `protobuf:"bytes,12,opt,name=muir_glacier_block,json=muirGlacierBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"muir_glacier_block,omitempty" yaml:"muir_glacier_block"`
	// Berlin switch block (nil = no fork, 0 = already on berlin)
	BerlinBlock *sdkmath.Int `protobuf:"bytes,13,opt,name=berlin_block,json=berlinBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"berlin_block,omitempty" yaml:"berlin_block"`
	// London switch block (nil = no fork, 0 = already on london)
	LondonBlock *sdkmath.Int `protobuf:"bytes,17,opt,name=london_block,json=londonBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"london_block,omitempty" yaml:"london_block"`
	// Eip-4345 (bomb delay) switch block (nil = no fork, 0 = already activated)
	ArrowGlacierBlock *sdkmath.Int `protobuf:"bytes,18,opt,name=arrow_glacier_block,json=arrowGlacierBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"arrow_glacier_block,omitempty" yaml:"arrow_glacier_block"`
	// EIP-3675 (TheMerge) switch block (nil = no fork, 0 = already in merge proceedings)
	MergeForkBlock *sdkmath.Int `protobuf:"bytes,19,opt,name=merge_fork_block,json=mergeForkBlock,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"merge_fork_block,omitempty" yaml:"merge_fork_block"`
}
