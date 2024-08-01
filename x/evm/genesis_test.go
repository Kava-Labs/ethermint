package evm_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/simapp"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	precompile_modules "github.com/ethereum/go-ethereum/precompile/modules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm"
	"github.com/evmos/ethermint/x/evm/types"
)

// TestInitGenesis performs various tests of the x/evm InitGenesis function.
//
// Each test case has a name and a function to generate a test fixture.
// The test fixture is given a complete and fresh app with context and returns
// an updated context, a genesis state to test against, the mocked registered
// precompiles, an expectation function, and a panic value (if expected).
//
// The expectFunc has a closure of the context, application state, state,
// and registered precompiles and is called after InitGenesis.  Therefore,
// it may use any of this information to define it's expectations and is
// able to verify the application state.
//
// The expected panic value should be nil if no panic is expected, and
// is checked by value, so it must be a string, error, or exact type given
// to the panic.
func TestInitGenesis(t *testing.T) {
	type testFixture struct {
		ctx         sdk.Context
		state       *types.GenesisState
		precompiles []precompile_modules.Module
		expectFunc  func()
		expectPanic any
	}

	testCases := []struct {
		name       string
		genFixture func(*testing.T, sdk.Context, *app.EthermintApp) testFixture
	}{
		{
			name: "Default genesis does not panic",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				state := types.DefaultGenesisState()

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: nil,
				}
			},
		},
		{
			name: "The chain id is set from the context",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				expectFunc := func() {
					ctxChainID, err := ethermint.ParseChainID(ctx.ChainID())
					require.NoError(t, err)

					require.NotNil(t, tApp.EvmKeeper.ChainID(), "expected keeper chain id to be set")
					assert.True(t, tApp.EvmKeeper.ChainID().Cmp(ctxChainID) == 0, "expected keeper chain id to match context")
				}

				return testFixture{
					ctx:         ctx,
					state:       types.DefaultGenesisState(),
					precompiles: nil,
					expectFunc:  expectFunc,
					expectPanic: nil,
				}
			},
		},
		{
			name: "An invalid chain id panics",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				ctx = ctx.WithChainID("ethermint-1")

				_, err := ethermint.ParseChainID("ethermint-1")
				require.Error(t, err)

				return testFixture{
					ctx:         ctx,
					state:       types.DefaultGenesisState(),
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: err,
				}
			},
		},
		{
			name: "Parameters are set",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				state := types.DefaultGenesisState()

				// Ensure parameters change to gain confidence entire param set is stored
				state.Params.EvmDenom = state.Params.EvmDenom + "/test"
				state.Params.EnableCall = !state.Params.EnableCall
				state.Params.EnableCreate = !state.Params.EnableCreate

				expectFunc := func() {
					assert.Equal(t, state.Params, tApp.EvmKeeper.GetParams(ctx), "expected stored params to match genesis params")
				}

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  expectFunc,
					expectPanic: nil,
				}
			},
		},
		{
			name: "Invalid parameters cause a panic",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				state := types.DefaultGenesisState()

				// evm denom must always be set
				state.Params.EvmDenom = ""

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: errors.New("error setting params invalid denom: "),
				}
			},
		},
		{
			name: "Panics if the evm module account is not already set",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				// Init genesis checks for the module accounts address existance in
				// the module account list of permissions (what GetModuleAddress checks).
				//
				// If this is not set in app.go, then we will see a panic.  Here
				// we delete the entry to mimic the behavior of incorrect app setup.
				delete(tApp.AccountKeeper.GetModulePermissions(), types.ModuleName)

				return testFixture{
					ctx:         ctx,
					state:       types.DefaultGenesisState(),
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: "the EVM module account has not been set",
				}
			},
		},
		{
			name: "Panics when a genesis account references an account not does not exist",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				// generate a random address that will not collide with any existing state or accounts
				address := generateRandomAddress(t)

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
				})

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: fmt.Errorf("account not found for address %s", address),
				}
			},
		},
		{
			name: "Panics when a genesis account references a non ethereum account",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())

				acc := authtypes.NewBaseAccountWithAddress(accAddr)
				acc.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, acc)

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: fmt.Errorf("account %s must be an EthAccount interface, got %T", address, acc),
				}
			},
		},
		{
			name: "Panics when there is a code hash mismatch between auth and evm accounts",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				code := []byte{0x01, 0x02, 0x03}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)

				incorrectCodeHash := crypto.Keccak256Hash([]byte("incorrect code"))

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    codeHex,
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    incorrectCodeHash.String(),
				}
				acc.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				s := "the evm state code doesn't match with the codehash\n"
				expectedPanic := fmt.Sprintf("%s account: %s , evm state codehash: %v, ethAccount codehash: %v, evm state code: %s\n", s, address, codeHash, incorrectCodeHash, codeHex)

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: expectedPanic,
				}
			},
		},
		{
			name: "Panics when there is a code hash mismatch and matching genesis account contains no code",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				someCodeHash := crypto.Keccak256Hash([]byte("an outdated codehash from a delete error"))

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    "", // this does not panic
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    someCodeHash.String(),
				}
				acc.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				s := "the evm state code doesn't match with the codehash\n"
				expectedPanic := fmt.Sprintf("%s account: %s , evm state codehash: %v, ethAccount codehash: %v, evm state code: %s\n", s, address, common.BytesToHash(types.EmptyCodeHash), acc.GetCodeHash(), "")

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: expectedPanic,
				}
			},
		},
		{
			name: "Panics when code is set and code hash is empty",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				code := []byte{0x01, 0x02, 0x03}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    codeHex,
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    "", // we do not allow empty code hash when code is set
				}
				acc.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				s := "the evm state code doesn't match with the codehash\n"
				expectedPanic := fmt.Sprintf("%s account: %s , evm state codehash: %v, ethAccount codehash: %v, evm state code: %s\n", s, address, codeHash, acc.GetCodeHash(), codeHex)

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: expectedPanic,
				}
			},
		},
		{
			name: "Genesis account code is stored by hash in the keeper state",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				code := []byte{0x01, 0x02, 0x03}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    codeHex,
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    codeHash.String(),
				}
				acc.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				expectFunc := func() {
					storedCode := tApp.EvmKeeper.GetCode(ctx, codeHash)

					require.NotNil(t, storedCode, "expected code to be stored by hash in keeper")
					require.Equal(t, code, storedCode, "expected stored code to match hex decoded code in genesis account")
				}

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  expectFunc,
					expectPanic: nil,
				}
			},
		},
		{
			name: "Genesis account storage keys are decoded and stored as bytes in keeper",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				code := []byte{0x01, 0x02, 0x03}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)

				rawStorage := [][2][]byte{
					{common.BytesToHash([]byte{0x01}).Bytes(), common.BytesToHash([]byte{0x02}).Bytes()},
					{common.BytesToHash([]byte{0x03}).Bytes(), common.BytesToHash([]byte{0x04}).Bytes()},
					{common.BytesToHash([]byte{0x04}).Bytes(), common.BytesToHash([]byte{0x05}).Bytes()},
				}

				storage := []types.State{}
				for _, rs := range rawStorage {
					storage = append(storage, types.State{
						Key:   common.Bytes2Hex(rs[0]),
						Value: common.Bytes2Hex(rs[1]),
					})
				}

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    codeHex,
					Storage: storage,
				})

				evmAddr := common.HexToAddress(address)
				accAddr := sdk.AccAddress(evmAddr.Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    codeHash.String(),
				}
				acc.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				expectFunc := func() {
					for _, rs := range rawStorage {
						expectedValue := tApp.EvmKeeper.GetState(ctx, evmAddr, common.BytesToHash(rs[0]))
						assert.Equalf(t, common.BytesToHash(rs[1]), expectedValue, "expected value at account %s and key %s to match expected value", evmAddr, common.Bytes2Hex(rs[0]))
					}
				}

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  expectFunc,
					expectPanic: nil,
				}
			},
		},
		{
			name: "Panics when enabled precompiles are not sorted ascending",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				state := types.DefaultGenesisState()

				addr1 := common.BytesToAddress([]byte{0x02})
				addr2 := common.BytesToAddress([]byte{0x01})

				state.Params.EnabledPrecompiles = []string{addr1.String(), addr2.String()}

				registeredPrecompiles := []precompile_modules.Module{
					{Address: addr1},
					{Address: addr2},
				}

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: registeredPrecompiles,
					expectFunc:  func() {},
					expectPanic: fmt.Errorf("error setting params enabled precompiles are not sorted, %s > %s", addr1.String(), addr2.String()),
				}
			},
		},
		{
			name: "Panics when enabled precompiles are not unique",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				state := types.DefaultGenesisState()

				addr1 := common.BytesToAddress([]byte{0x01})
				state.Params.EnabledPrecompiles = []string{addr1.String(), addr1.String()}

				registeredPrecompiles := []precompile_modules.Module{
					{Address: addr1},
				}

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: registeredPrecompiles,
					expectFunc:  func() {},
					expectPanic: fmt.Errorf("error setting params enabled precompiles are not unique, %s is duplicated", addr1.String()),
				}
			},
		},
		{
			name: "Panics when enabled precompiles exists but is not registered",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				state := types.DefaultGenesisState()

				addr1 := common.BytesToAddress([]byte{0x01})
				addr2 := common.BytesToAddress([]byte{0x02})

				state.Params.EnabledPrecompiles = []string{addr1.String(), addr2.String()}

				registeredPrecompiles := []precompile_modules.Module{
					{Address: addr1},
				}

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: registeredPrecompiles,
					expectFunc:  func() {},
					expectPanic: fmt.Errorf("precompile %s is enabled but not registered", addr2.String()),
				}
			},
		},
		{
			name: "Valid enabled precompiles are set in params",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				state := types.DefaultGenesisState()

				addr1 := common.BytesToAddress([]byte{0x01})
				addr2 := common.BytesToAddress([]byte{0x02})

				state.Params.EnabledPrecompiles = []string{addr1.String(), addr2.String()}

				registeredPrecompiles := []precompile_modules.Module{
					{Address: addr1},
					{Address: addr2},
				}

				code := []byte{0x01}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)
				state.Accounts = append(state.Accounts,
					types.GenesisAccount{
						Address: addr1.String(),
						Code:    codeHex,
					},
					types.GenesisAccount{
						Address: addr2.String(),
						Code:    codeHex,
					},
				)

				accAddr1 := sdk.AccAddress(addr1.Bytes())
				acc1 := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr1),
					CodeHash:    codeHash.String(),
				}
				acc1.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, &acc1)

				accAddr2 := sdk.AccAddress(addr2.Bytes())
				acc2 := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr2),
					CodeHash:    codeHash.String(),
				}
				acc2.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, &acc2)

				expectFunc := func() {
					assert.Equal(t,
						state.Params.EnabledPrecompiles,
						tApp.EvmKeeper.GetParams(ctx).EnabledPrecompiles,
						"expected enabled precompiles to be set in state",
					)
				}

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: registeredPrecompiles,
					expectFunc:  expectFunc,
					expectPanic: nil,
				}
			},
		},
		{
			name: "Panics when genesis account has a nonce equal to zero",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				code := []byte{0x01, 0x02, 0x03}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    codeHex,
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    codeHash.String(),
				}
				acc.Sequence = uint64(0) // Not allowed for contracts
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: fmt.Errorf("account %s must have a positive nonce", address),
				}
			},
		},
		{
			name: "Allows a genesis account with a nonce greater than 1",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				code := []byte{0x01, 0x02, 0x03}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    codeHex,
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    codeHash.String(),
				}
				acc.Sequence = uint64(1000)
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: nil,
				}
			},
		},
		{
			name: "Panics when genesis account has a public key set",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				privkey, err := ethsecp256k1.GenerateKey()
				require.NoError(t, err)
				address := common.BytesToAddress(privkey.PubKey().Address()).String()

				code := []byte{0x01, 0x02, 0x03}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)

				state := types.DefaultGenesisState()
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    codeHex,
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    codeHash.String(),
				}
				acc.Sequence = uint64(1)
				pubkey, err := codectypes.NewAnyWithValue(privkey.PubKey())
				require.NoError(t, err)
				acc.PubKey = pubkey
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: nil,
					expectFunc:  nil,
					expectPanic: fmt.Errorf("account %s must not have a public key set", address),
				}
			},
		},
		{
			name: "Panics when enabled precompile does not have a genesis account with code 0x01",
			genFixture: func(t *testing.T, ctx sdk.Context, tApp *app.EthermintApp) testFixture {
				address := generateRandomAddress(t)

				code := []byte{0x02}
				codeHash := crypto.Keccak256Hash(code)
				codeHex := common.Bytes2Hex(code)

				state := types.DefaultGenesisState()
				state.Params.EnabledPrecompiles = []string{address}
				state.Accounts = append(state.Accounts, types.GenesisAccount{
					Address: address,
					Code:    codeHex,
				})

				accAddr := sdk.AccAddress(common.HexToAddress(address).Bytes())
				acc := ethermint.EthAccount{
					BaseAccount: authtypes.NewBaseAccountWithAddress(accAddr),
					CodeHash:    codeHash.String(),
				}
				acc.Sequence = uint64(1)
				tApp.AccountKeeper.SetAccount(ctx, &acc)

				registeredPrecompiles := []precompile_modules.Module{{Address: common.HexToAddress(address)}}

				return testFixture{
					ctx:         ctx,
					state:       state,
					precompiles: registeredPrecompiles,
					expectFunc:  nil,
					expectPanic: fmt.Errorf("enabled precompile %s must have code set to 0x01, got 0x%s", address, codeHex),
				}
			},
		},
	}

	for _, tc := range testCases {
		// For each test case, create a sdk context and app for each test, run init genesis, check panics,
		// then run any expectations included in the test case.
		t.Run(tc.name, func(t *testing.T) {
			// Create a context with test app to instantiate keepers to pass to init genesis.
			ctx, tApp := setupApp()

			// Get the genesis state to import, the registered precompiled to validate against,
			// and the function that will make assertions about the state after init genesis has run.
			tf := tc.genFixture(t, ctx, tApp)

			// Perform init genesis and validate we never provide validator updates
			testFunc := func() {
				validatorUpdates := evm.InitGenesis(tf.ctx, tApp.EvmKeeper, tApp.AccountKeeper, *tf.state, tf.precompiles)
				require.Equal(t, 0, len(validatorUpdates), "expected no validator updates in all init genesis scenarios")
			}

			// Check panic or no panic -- these are required expectations.
			if tf.expectPanic == nil {
				require.NotPanics(t, testFunc, "expected init genesis to not panic")
			} else {
				// It's important here to test full panic assertions to ensure that our test is
				// raising a panic for the correct account address, code hash, etc.
				switch expectedPanicValue := tf.expectPanic.(type) {
				case error:
					require.PanicsWithError(t, expectedPanicValue.Error(), testFunc, "expected init genesis to panic with correct error")
				default:
					require.PanicsWithValue(t, expectedPanicValue, testFunc, "expected init genesis to panic with correct value")
				}
			}

			// Run test specific assertions
			if tf.expectFunc != nil {
				tf.expectFunc()
			}
		})
	}
}

// setupApp creates a app and context with an in-memory database for testing
func setupApp() (sdk.Context, *app.EthermintApp) {
	isCheckTx := false

	tApp := app.Setup(isCheckTx, func(_ *app.EthermintApp, genesis simapp.GenesisState) simapp.GenesisState {
		return genesis
	})
	ctx := tApp.BaseApp.NewContext(isCheckTx, tmproto.Header{Height: 1, Time: time.Now().UTC(), ChainID: "ethermint_9000-1"})

	return ctx, tApp
}

// generateRandomAddress generates a cryptographically secure random 0x address
func generateRandomAddress(t *testing.T) string {
	privkey, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)

	return common.BytesToAddress(privkey.PubKey().Address()).String()
}
