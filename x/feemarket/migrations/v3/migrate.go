package v4

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/ethermint/x/feemarket/types"
)

// MigrateStore migrates the x/feemarket module state from the consensus version 2 to version 3
func MigrateStore(
	ctx sdk.Context,
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
) error {
	var (
		store  = ctx.KVStore(storeKey)
		params = types.DefaultParams()
	)
	params.NoBaseFee = true

	if err := params.Validate(); err != nil {
		return err
	}

	bz, err := cdc.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(types.ParamsKey, bz)

	return nil
}
