package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

var _ PrecompileKeeper = (*DefaultPrecompileKeeper)(nil)

// DefaultPrecompileKeeper defines an empty PrecompileKeeper that returns an
// empty list of precompile addresses. This effectively disables all stateful
// precompiles used for testing within Ethermint as the x/precompile module
// lives in the Kava repository.
type DefaultPrecompileKeeper struct{}

// GetPrecompileAddresses returns an empty list of precompile addresses.
func (DefaultPrecompileKeeper) GetPrecompileAddresses(
	_ sdk.Context,
) []common.Address {
	return nil
}
