package statedb_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	xevm "github.com/evmos/ethermint/x/evm"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/testutil"
	"github.com/evmos/ethermint/x/evm/types"
)

var (
	address  common.Address = common.BigToAddress(big.NewInt(101))
	address2 common.Address = common.BigToAddress(big.NewInt(102))
	address3 common.Address = common.BigToAddress(big.NewInt(103))

	blockHash     common.Hash      = common.BigToHash(big.NewInt(9999))
	emptyTxConfig statedb.TxConfig = statedb.NewEmptyTxConfig(blockHash)
	emptyCodeHash                  = crypto.Keccak256(nil)
)

type HybridStateDBTestSuite struct {
	testutil.TestSuite
}

func TestHybridStateDBTestSuite(t *testing.T) {
	suite.Run(t, &HybridStateDBTestSuite{})
}

func (suite *HybridStateDBTestSuite) TestGetBalance_External() {
	// GetBalance() should account for balance changes from sdk.Context and
	// external factors - e.g. precompiles that modify user account balances.
	keeper := suite.App.EvmKeeper
	params := keeper.GetParams(suite.Ctx)

	tests := []struct {
		name     string
		maleate  func(db *statedb.StateDB)
		expected *big.Int
	}{
		{
			"no external balance changes",
			func(db *statedb.StateDB) {
				amount := big.NewInt(100)
				db.AddBalance(address, amount)
			},
			big.NewInt(100),
		},
		{
			"bank balance add",
			func(db *statedb.StateDB) {
				amount := big.NewInt(100)
				db.AddBalance(address, amount)

				// Add some external balance changes, this could be a call to a precompile
				// that transfers funds.
				externalTransferAmount := big.NewInt(50)
				suite.MintCoinsForAccount(
					db.Context(),
					sdk.AccAddress(address.Bytes()),
					sdk.NewCoins(sdk.NewCoin(params.EvmDenom, sdk.NewIntFromBigInt(externalTransferAmount))),
				)
			},
			big.NewInt(150),
		},
		{
			"bank balance sub",
			func(db *statedb.StateDB) {
				amount := big.NewInt(100)
				db.AddBalance(address, amount)

				externalTransferAmount := big.NewInt(50)
				err := suite.App.BankKeeper.SendCoins(
					db.Context(),
					sdk.AccAddress(address.Bytes()),
					sdk.AccAddress(address2.Bytes()),
					sdk.NewCoins(sdk.NewCoin(params.EvmDenom, sdk.NewIntFromBigInt(externalTransferAmount))),
				)
				suite.Require().NoError(err)
			},
			big.NewInt(50),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()
			keeper := suite.App.EvmKeeper

			db := statedb.New(suite.Ctx, keeper, emptyTxConfig)
			tt.maleate(db)

			suite.Require().NoError(db.Commit())

			// New db after Commit()
			db = statedb.New(suite.Ctx, keeper, emptyTxConfig)

			suite.Require().Equal(
				tt.expected,
				db.GetBalance(address),
				"GetBalance should account for both internal and external balance changes",
			)
		})
	}
}

func (suite *HybridStateDBTestSuite) TestExist() {
	// When precompiles create accounts, it should be visible to StateDB.
	db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)

	suite.Require().False(db.Exist(address), "account shouldn't be created or found")

	// Create account externally
	suite.MintCoinsForAccount(
		db.Context(), // Use the statedb context!
		sdk.AccAddress(address.Bytes()),
		sdk.NewCoins(sdk.NewCoin(types.DefaultEVMDenom, sdk.NewInt(100))),
	)

	// Account should be visible to StateDB - ensures the correct ctx is used
	// in the StateDB
	suite.Require().True(db.Exist(address), "account should be created and found")
}

func (suite *HybridStateDBTestSuite) TestForEachStorage_Committed() {
	db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
	// Set some storage
	db.SetState(address, common.BytesToHash([]byte("key")), common.BytesToHash([]byte("value")))

	// Commit changes
	suite.Require().NoError(db.Commit())

	// Create a new StateDB
	db = statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)

	// Iterate over storage
	var keys []common.Hash
	db.ForEachStorage(address, func(key, value common.Hash) bool {
		keys = append(keys, key)
		return false
	})

	suite.Require().Len(keys, 1, "expected 1 key")
	suite.Require().Equal(common.BytesToHash([]byte("key")), keys[0], "expected key to be found")
}

func (suite *HybridStateDBTestSuite) TestForEachStorage_Dirty() {
	suite.T().Skip("TODO: identify if ForEachStorage should return new keys")

	db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
	// Set some storage
	db.SetState(address, common.BytesToHash([]byte("key")), common.BytesToHash([]byte("value")))

	// Iterate over storage
	var keys []common.Hash
	db.ForEachStorage(address, func(key, value common.Hash) bool {
		keys = append(keys, key)
		return false
	})

	suite.Require().Len(keys, 1, "expected 1 key")
	suite.Require().Equal(common.BytesToHash([]byte("key")), keys[0], "expected key to be found")
}

func (suite *HybridStateDBTestSuite) TestAccount() {
	key1 := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(2))
	key2 := common.BigToHash(big.NewInt(3))
	value2 := common.BigToHash(big.NewInt(4))
	testCases := []struct {
		name     string
		malleate func(*statedb.StateDB)
	}{
		{"non-exist account", func(db *statedb.StateDB) {
			suite.Require().Equal(false, db.Exist(address))
			suite.Require().Equal(true, db.Empty(address))
			suite.Require().Equal(big.NewInt(0), db.GetBalance(address))
			suite.Require().Equal([]byte(nil), db.GetCode(address))
			suite.Require().Equal(common.Hash{}, db.GetCodeHash(address))
			suite.Require().Equal(uint64(0), db.GetNonce(address))
		}},
		{"empty account", func(db *statedb.StateDB) {
			db.CreateAccount(address)
			suite.Require().NoError(db.Commit())

			keeper := db.Keeper().(*keeper.Keeper)
			acct := keeper.GetAccount(suite.Ctx, address)
			states := suite.GetAllAccountStorage(suite.Ctx, address)

			suite.Require().Equal(statedb.NewEmptyAccount(), acct)
			suite.Require().Empty(states)
			suite.Require().False(acct.IsContract())

			db = statedb.New(suite.Ctx, keeper, emptyTxConfig)
			suite.Require().Equal(true, db.Exist(address))
			suite.Require().Equal(true, db.Empty(address))
			suite.Require().Equal(big.NewInt(0), db.GetBalance(address))
			suite.Require().Equal([]byte(nil), db.GetCode(address))
			suite.Require().Equal(common.BytesToHash(emptyCodeHash), db.GetCodeHash(address))
			suite.Require().Equal(uint64(0), db.GetNonce(address))
		}},
		{"suicide", func(db *statedb.StateDB) {
			// non-exist account.
			suite.Require().False(db.Suicide(address))
			suite.Require().False(db.HasSuicided(address))

			// create a contract account
			db.CreateAccount(address)
			db.SetCode(address, []byte("hello world"))
			db.AddBalance(address, big.NewInt(100))
			db.SetState(address, key1, value1)
			db.SetState(address, key2, value2)
			suite.Require().NoError(db.Commit())

			// suicide
			db = statedb.New(suite.Ctx, db.Keeper(), emptyTxConfig)
			suite.Require().False(db.HasSuicided(address))
			suite.Require().True(db.Suicide(address))

			// check dirty state
			suite.Require().True(db.HasSuicided(address))
			// balance is cleared
			suite.Require().Equal(big.NewInt(0), db.GetBalance(address))
			// but code and state are still accessible in dirty state
			suite.Require().Equal(value1, db.GetState(address, key1))
			suite.Require().Equal([]byte("hello world"), db.GetCode(address))

			suite.Require().NoError(db.Commit())

			// not accessible from StateDB anymore
			db = statedb.New(suite.Ctx, db.Keeper(), emptyTxConfig)
			suite.Require().False(db.Exist(address))

			// and cleared in keeper too
			keeper := db.Keeper().(*keeper.Keeper)
			acc := keeper.GetAccount(suite.Ctx, address)
			states := suite.GetAllAccountStorage(suite.Ctx, address)

			suite.Require().Empty(acc)
			suite.Require().Empty(states)
		}},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			keeper := suite.App.EvmKeeper
			db := statedb.New(suite.Ctx, keeper, emptyTxConfig)
			tc.malleate(db)
		})
	}
}

func (suite *HybridStateDBTestSuite) TestAccountOverride() {
	keeper := suite.App.EvmKeeper
	db := statedb.New(suite.Ctx, keeper, emptyTxConfig)
	// test balance carry over when overwritten
	amount := big.NewInt(1)

	// init an EOA account, account overriden only happens on EOA account.
	db.AddBalance(address, amount)
	db.SetNonce(address, 1)

	// override
	db.CreateAccount(address)

	// check balance is not lost
	suite.Require().Equal(amount, db.GetBalance(address))
	// but nonce is reset
	suite.Require().Equal(uint64(0), db.GetNonce(address))
}

func (suite *HybridStateDBTestSuite) TestDBError() {
	testCases := []struct {
		name        string
		malleate    func(*statedb.StateDB)
		errContains string
	}{
		{
			"negative balance",
			func(db *statedb.StateDB) {
				db.SubBalance(address, big.NewInt(1))
			},
			"insufficient funds",
		},
		{
			"multiple errors persist first error",
			func(db *statedb.StateDB) {
				db.SubBalance(address, big.NewInt(200))
				db.SubBalance(address2, big.NewInt(500))
			},
			"insufficient funds",
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
			tc.malleate(db)

			err := db.Commit()

			suite.Require().Error(err)
			suite.Require().ErrorContains(err, tc.errContains)
		})
	}
}

func (suite *HybridStateDBTestSuite) TestBalance() {
	// NOTE: no need to test overflow/underflow, that is guaranteed by evm implementation.
	testCases := []struct {
		name       string
		malleate   func(*statedb.StateDB)
		expBalance *big.Int
	}{
		{"add balance", func(db *statedb.StateDB) {
			db.AddBalance(address, big.NewInt(10))
		}, big.NewInt(10)},
		{"sub balance", func(db *statedb.StateDB) {
			db.AddBalance(address, big.NewInt(10))
			// get dirty balance
			suite.Require().Equal(big.NewInt(10), db.GetBalance(address))
			db.SubBalance(address, big.NewInt(2))
		}, big.NewInt(8)},
		{"add zero balance", func(db *statedb.StateDB) {
			db.AddBalance(address, big.NewInt(0))
		}, big.NewInt(0)},
		{"sub zero balance", func(db *statedb.StateDB) {
			db.SubBalance(address, big.NewInt(0))
		}, big.NewInt(0)},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			keeper := suite.App.EvmKeeper
			db := statedb.New(suite.Ctx, keeper, emptyTxConfig)
			tc.malleate(db)

			// check dirty state
			suite.Require().Equal(tc.expBalance, db.GetBalance(address))
			suite.Require().NoError(db.Commit())
			// check committed balance too
			suite.Require().Equal(tc.expBalance, keeper.GetBalance(suite.Ctx, address))
		})
	}
}

func (suite *HybridStateDBTestSuite) TestState() {
	key1 := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(1))
	testCases := []struct {
		name      string
		malleate  func(*statedb.StateDB)
		expStates map[common.Hash]common.Hash
	}{
		{"empty state", func(db *statedb.StateDB) {
		}, nil},

		{"set empty value deletes", func(db *statedb.StateDB) {
			db.SetState(address, key1, common.Hash{})
		}, map[common.Hash]common.Hash{}},

		{"noop state change - empty", func(db *statedb.StateDB) {
			db.SetState(address, key1, value1)
			db.SetState(address, key1, common.Hash{})
		}, map[common.Hash]common.Hash{}},

		{"noop state change - non-empty", func(db *statedb.StateDB) {
			// Start with non-empty committed state
			db.SetState(address, key1, value1)
			suite.Require().NoError(db.Commit())

			db.SetState(address, key1, common.Hash{})
			db.SetState(address, key1, value1)
		}, map[common.Hash]common.Hash{
			// Shouldn't be modified - Commit() may still write it again though
			key1: value1,
		}},

		{"set state", func(db *statedb.StateDB) {
			// check empty initial state
			suite.Require().Equal(common.Hash{}, db.GetState(address, key1))
			suite.Require().Equal(common.Hash{}, db.GetCommittedState(address, key1))

			// set state
			db.SetState(address, key1, value1)
			// query dirty state
			suite.Require().Equal(value1, db.GetState(address, key1))
			// check committed state is still not exist
			suite.Require().Equal(common.Hash{}, db.GetCommittedState(address, key1))

			// set same value again, should be noop
			db.SetState(address, key1, value1)
			suite.Require().Equal(value1, db.GetState(address, key1))
		}, map[common.Hash]common.Hash{
			key1: value1,
		}},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			keeper := suite.App.EvmKeeper
			db := statedb.New(suite.Ctx, keeper, emptyTxConfig)
			tc.malleate(db)
			suite.Require().NoError(db.Commit())

			// check committed states in keeper
			states := suite.GetAllAccountStorage(suite.Ctx, address)
			if len(tc.expStates) > 0 {
				suite.Require().Equal(tc.expStates, states)
			} else {
				suite.Require().Empty(states)
			}

			// check ForEachStorage
			db = statedb.New(suite.Ctx, keeper, emptyTxConfig)
			collected := CollectContractStorage(db)
			if len(tc.expStates) > 0 {
				suite.Require().Equal(tc.expStates, collected)
			} else {
				suite.Require().Empty(collected)
			}
		})
	}
}

func (suite *HybridStateDBTestSuite) TestCode() {
	code := []byte("hello world")
	codeHash := crypto.Keccak256Hash(code)

	testCases := []struct {
		name        string
		malleate    func(*statedb.StateDB)
		expCode     []byte
		expCodeHash common.Hash
	}{
		{"non-exist account", func(*statedb.StateDB) {}, nil, common.Hash{}},
		{"empty account", func(db *statedb.StateDB) {
			db.CreateAccount(address)
		}, nil, common.BytesToHash(emptyCodeHash)},
		{"set code", func(db *statedb.StateDB) {
			db.SetCode(address, code)
		}, code, codeHash},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			keeper := suite.App.EvmKeeper
			db := statedb.New(suite.Ctx, keeper, emptyTxConfig)
			tc.malleate(db)

			// check dirty state
			suite.Require().Equal(tc.expCode, db.GetCode(address))
			suite.Require().Equal(len(tc.expCode), db.GetCodeSize(address))
			suite.Require().Equal(tc.expCodeHash, db.GetCodeHash(address))

			suite.Require().NoError(db.Commit())

			// check again
			db = statedb.New(suite.Ctx, keeper, emptyTxConfig)
			suite.Require().Equal(tc.expCode, db.GetCode(address))
			suite.Require().Equal(len(tc.expCode), db.GetCodeSize(address))
			suite.Require().Equal(tc.expCodeHash, db.GetCodeHash(address))
		})
	}
}

func (suite *HybridStateDBTestSuite) TestRevertSnapshot() {
	v1 := common.BigToHash(big.NewInt(1))
	v2 := common.BigToHash(big.NewInt(2))
	v3 := common.BigToHash(big.NewInt(3))
	testCases := []struct {
		name     string
		malleate func(*statedb.StateDB)
	}{
		{"set state", func(db *statedb.StateDB) {
			db.SetState(address, v1, v3)
		}},
		{"set nonce", func(db *statedb.StateDB) {
			db.SetNonce(address, 10)
		}},
		{"change balance", func(db *statedb.StateDB) {
			db.AddBalance(address, big.NewInt(10))
			db.SubBalance(address, big.NewInt(5))
		}},
		{"override account", func(db *statedb.StateDB) {
			db.CreateAccount(address)
		}},
		{"set code", func(db *statedb.StateDB) {
			db.SetCode(address, []byte("hello world"))
		}},
		{"suicide", func(db *statedb.StateDB) {
			db.SetState(address, v1, v2)
			db.SetCode(address, []byte("hello world"))
			suite.Require().True(db.Suicide(address))
		}},
		{"add log", func(db *statedb.StateDB) {
			db.AddLog(&ethtypes.Log{
				Address: address,
			})
		}},
		{"add refund", func(db *statedb.StateDB) {
			db.AddRefund(10)
			db.SubRefund(5)
		}},
		{"access list", func(db *statedb.StateDB) {
			db.AddAddressToAccessList(address)
			db.AddSlotToAccessList(address, v1)
		}},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.App.AccountKeeper.GetModuleAccount(suite.Ctx, types.ModuleName)

			ctx := suite.Ctx
			keeper := suite.App.EvmKeeper

			{
				// do some arbitrary changes to the storage
				db := statedb.New(ctx, keeper, emptyTxConfig)
				db.SetNonce(address, 1)
				db.AddBalance(address, big.NewInt(100))
				db.SetCode(address, []byte("hello world"))
				db.SetState(address, v1, v2)
				db.SetNonce(address2, 1)
				suite.Require().NoError(db.Commit())
			}

			originalState := xevm.ExportGenesis(suite.Ctx, keeper, suite.App.AccountKeeper)

			// run test
			db := statedb.New(ctx, keeper, emptyTxConfig)
			rev := db.Snapshot()
			tc.malleate(db)
			db.RevertToSnapshot(rev)

			// check empty states after revert
			suite.Require().Zero(db.GetRefund())
			suite.Require().Empty(db.Logs())

			suite.Require().NoError(db.Commit())

			revertState := xevm.ExportGenesis(suite.Ctx, keeper, suite.App.AccountKeeper)

			// check keeper state should stay the same
			suite.Require().Equal(originalState, revertState)
		})
	}
}

func (suite *HybridStateDBTestSuite) TestNestedSnapshot() {
	key := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(1))
	value2 := common.BigToHash(big.NewInt(2))

	db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)

	rev1 := db.Snapshot()
	db.SetState(address, key, value1)

	rev2 := db.Snapshot()
	db.SetState(address, key, value2)
	suite.Require().Equal(value2, db.GetState(address, key))

	db.RevertToSnapshot(rev2)
	suite.Require().Equal(value1, db.GetState(address, key))

	db.RevertToSnapshot(rev1)
	suite.Require().Equal(common.Hash{}, db.GetState(address, key))
}

func (suite *HybridStateDBTestSuite) TestBalanceSnapshots() {
	db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
	// take a snapshot
	snapshot := db.Snapshot()

	// add balance
	db.AddBalance(address, big.NewInt(10))
	suite.Require().Equal(big.NewInt(10), db.GetBalance(address))

	// revert to snapshot
	db.RevertToSnapshot(snapshot)
	// balance should be reverted
	suite.Require().Equal(big.NewInt(0), db.GetBalance(address))

	// add balance again
	db.AddBalance(address, big.NewInt(10))
	suite.Require().Equal(big.NewInt(10), db.GetBalance(address))

	// commit
	suite.Require().NoError(db.Commit())

	db = statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
	// balance should be committed
	suite.Require().Equal(big.NewInt(10), db.GetBalance(address))
}

func (suite *HybridStateDBTestSuite) TestInvalidSnapshotId() {
	db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
	suite.Require().Panics(func() {
		db.RevertToSnapshot(1)
	})
}

func (suite *HybridStateDBTestSuite) TestAccessList() {
	value1 := common.BigToHash(big.NewInt(1))
	value2 := common.BigToHash(big.NewInt(2))

	testCases := []struct {
		name     string
		malleate func(*statedb.StateDB)
	}{
		{"add address", func(db *statedb.StateDB) {
			suite.Require().False(db.AddressInAccessList(address))
			db.AddAddressToAccessList(address)
			suite.Require().True(db.AddressInAccessList(address))

			addrPresent, slotPresent := db.SlotInAccessList(address, value1)
			suite.Require().True(addrPresent)
			suite.Require().False(slotPresent)

			// add again, should be no-op
			db.AddAddressToAccessList(address)
			suite.Require().True(db.AddressInAccessList(address))
		}},
		{"add slot", func(db *statedb.StateDB) {
			addrPresent, slotPresent := db.SlotInAccessList(address, value1)
			suite.Require().False(addrPresent)
			suite.Require().False(slotPresent)
			db.AddSlotToAccessList(address, value1)
			addrPresent, slotPresent = db.SlotInAccessList(address, value1)
			suite.Require().True(addrPresent)
			suite.Require().True(slotPresent)

			// add another slot
			db.AddSlotToAccessList(address, value2)
			addrPresent, slotPresent = db.SlotInAccessList(address, value2)
			suite.Require().True(addrPresent)
			suite.Require().True(slotPresent)

			// add again, should be noop
			db.AddSlotToAccessList(address, value2)
			addrPresent, slotPresent = db.SlotInAccessList(address, value2)
			suite.Require().True(addrPresent)
			suite.Require().True(slotPresent)
		}},
		{"prepare access list", func(db *statedb.StateDB) {
			suite.SetupTest()

			al := ethtypes.AccessList{{
				Address:     address3,
				StorageKeys: []common.Hash{value1},
			}}
			db.PrepareAccessList(address, &address2, vm.PrecompiledAddressesBerlin, al)

			// check sender and dst
			suite.Require().True(db.AddressInAccessList(address))
			suite.Require().True(db.AddressInAccessList(address2))
			// check precompiles
			suite.Require().True(db.AddressInAccessList(common.BytesToAddress([]byte{1})))
			// check AccessList
			suite.Require().True(db.AddressInAccessList(address3))
			addrPresent, slotPresent := db.SlotInAccessList(address3, value1)
			suite.Require().True(addrPresent)
			suite.Require().True(slotPresent)
			addrPresent, slotPresent = db.SlotInAccessList(address3, value2)
			suite.Require().True(addrPresent)
			suite.Require().False(slotPresent)
		}},
	}

	for _, tc := range testCases {
		db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
		tc.malleate(db)
	}
}

func (suite *HybridStateDBTestSuite) TestLog() {
	txHash := common.BytesToHash([]byte("tx"))
	// use a non-default tx config
	txConfig := statedb.NewTxConfig(
		blockHash,
		txHash,
		1, 1,
	)
	db := statedb.New(suite.Ctx, suite.App.EvmKeeper, txConfig)
	data := []byte("hello world")
	db.AddLog(&ethtypes.Log{
		Address:     address,
		Topics:      []common.Hash{},
		Data:        data,
		BlockNumber: 1,
	})
	suite.Require().Equal(1, len(db.Logs()))
	expectedLog := &ethtypes.Log{
		Address:     address,
		Topics:      []common.Hash{},
		Data:        data,
		BlockNumber: 1,
		BlockHash:   blockHash,
		TxHash:      txHash,
		TxIndex:     1,
		Index:       1,
	}
	suite.Require().Equal(expectedLog, db.Logs()[0])

	db.AddLog(&ethtypes.Log{
		Address:     address,
		Topics:      []common.Hash{},
		Data:        data,
		BlockNumber: 1,
	})
	suite.Require().Equal(2, len(db.Logs()))
	expectedLog.Index++
	suite.Require().Equal(expectedLog, db.Logs()[1])
}

func (suite *HybridStateDBTestSuite) TestRefund() {
	testCases := []struct {
		name      string
		malleate  func(*statedb.StateDB)
		expRefund uint64
		expPanic  bool
	}{
		{"add refund", func(db *statedb.StateDB) {
			db.AddRefund(uint64(10))
		}, 10, false},
		{"sub refund", func(db *statedb.StateDB) {
			db.AddRefund(uint64(10))
			db.SubRefund(uint64(5))
		}, 5, false},
		{"negative refund counter", func(db *statedb.StateDB) {
			db.AddRefund(uint64(5))
			db.SubRefund(uint64(10))
		}, 0, true},
	}
	for _, tc := range testCases {
		suite.SetupTest()

		db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
		if !tc.expPanic {
			tc.malleate(db)
			suite.Require().Equal(tc.expRefund, db.GetRefund())
		} else {
			suite.Require().Panics(func() {
				tc.malleate(db)
			})
		}
	}
}

func (suite *HybridStateDBTestSuite) TestIterateStorage() {
	key1 := common.BigToHash(big.NewInt(1))
	value1 := common.BigToHash(big.NewInt(2))
	key2 := common.BigToHash(big.NewInt(3))
	value2 := common.BigToHash(big.NewInt(4))

	keeper := suite.App.EvmKeeper
	db := statedb.New(suite.Ctx, keeper, emptyTxConfig)
	db.SetState(address, key1, value1)
	db.SetState(address, key2, value2)

	// ForEachStorage only iterate committed state
	suite.Require().Empty(CollectContractStorage(db))

	suite.Require().NoError(db.Commit())

	storage := CollectContractStorage(db)
	suite.Require().Equal(2, len(storage))

	accStorages := suite.GetAllAccountStorage(suite.Ctx, address)
	suite.Require().Equal(accStorages, storage)

	// break early iteration
	storage = make(statedb.Storage)
	db.ForEachStorage(address, func(k, v common.Hash) bool {
		storage[k] = v
		// return false to break early
		return false
	})
	suite.Require().Equal(1, len(storage))
}

func CollectContractStorage(db *statedb.StateDB) map[common.Hash]common.Hash {
	storage := make(map[common.Hash]common.Hash)
	db.ForEachStorage(address, func(k, v common.Hash) bool {
		storage[k] = v
		return true
	})
	return storage
}
