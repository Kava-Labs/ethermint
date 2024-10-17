package v3

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/ethermint/x/evm/types"
	legacytypes "github.com/evmos/ethermint/x/evm/types/legacy"
)

// MigrateStore runs the state migrations that includes upstream consensus
// versions v2 to v5. Kava consensus version diverges from upstream at v2.
func MigrateStore(
	ctx sdk.Context,
	paramstore types.Subspace,
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
) error {
	// ensure a subspace not passed from keeper includes the legacy param key table
	if !paramstore.HasKeyTable() {
		paramstore = paramstore.WithKeyTable(legacytypes.ParamKeyTable())
	}

	// load existing legacy parameters
	var legacyParams legacytypes.LegacyParams
	paramstore.GetParamSetIfExists(ctx, &legacyParams)

	// -------------------------------------------------------------------------
	// Upstream v2 to v3
	// New GrayGlacierBlock and MergeNetsplitBlock in ChainConfig parameter.
	// Any new fields are disabled / nil. These should be nil if we leave them
	// out because of the default value, but set explicitly here to nil.

	newChainConfig := types.ChainConfig{
		HomesteadBlock:      legacyParams.ChainConfig.HomesteadBlock,
		DAOForkBlock:        legacyParams.ChainConfig.DAOForkBlock,
		DAOForkSupport:      legacyParams.ChainConfig.DAOForkSupport,
		EIP150Block:         legacyParams.ChainConfig.EIP150Block,
		EIP150Hash:          legacyParams.ChainConfig.EIP150Hash,
		EIP155Block:         legacyParams.ChainConfig.EIP155Block,
		EIP158Block:         legacyParams.ChainConfig.EIP158Block,
		ByzantiumBlock:      legacyParams.ChainConfig.ByzantiumBlock,
		ConstantinopleBlock: legacyParams.ChainConfig.ConstantinopleBlock,
		PetersburgBlock:     legacyParams.ChainConfig.PetersburgBlock,
		IstanbulBlock:       legacyParams.ChainConfig.IstanbulBlock,
		MuirGlacierBlock:    legacyParams.ChainConfig.MuirGlacierBlock,
		BerlinBlock:         legacyParams.ChainConfig.BerlinBlock,
		LondonBlock:         legacyParams.ChainConfig.LondonBlock,
		ArrowGlacierBlock:   legacyParams.ChainConfig.ArrowGlacierBlock,

		// This is an old field, but renamed from mergeForkBlock
		MergeNetsplitBlock: legacyParams.ChainConfig.MergeForkBlock,

		// New fields are nil
		GrayGlacierBlock: nil,
		ShanghaiBlock:    nil,
		CancunBlock:      nil,
	}

	// -------------------------------------------------------------------------
	// Upstream v3 to v4
	// Moves from deprecated Cosmos SDK params module to x/evm module state.

	// Params in store is currently empty
	store := ctx.KVStore(storeKey)

	newParams := types.Params{
		EvmDenom:            legacyParams.EvmDenom,
		EnableCreate:        legacyParams.EnableCreate,
		EnableCall:          legacyParams.EnableCall,
		ExtraEIPs:           legacyParams.ExtraEIPs,
		ChainConfig:         newChainConfig,
		EIP712AllowedMsgs:   legacyParams.EIP712AllowedMsgs,
		AllowUnprotectedTxs: false, // Upstream v1 to v2
	}

	if err := newParams.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&newParams)
	store.Set(types.KeyPrefixParams, bz)

	return nil
}
