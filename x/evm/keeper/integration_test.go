package keeper_test

import (
	_ "embed"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core/vm"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/testutil"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	testutil.TestSuite
}

var test *IntegrationTestSuite

func TestIntegrationTestSuite(t *testing.T) {
	test = new(IntegrationTestSuite)
	test.EnableFeemarket = false
	test.EnableLondonHF = false
	suite.Run(t, test)
}

func (suite *IntegrationTestSuite) TestState_EmptyToEmpty() {
	addr := suite.DeployContract(testutil.StateTestContract)
	storageBefore := suite.GetAllAccountStorage(suite.Ctx, addr)

	contractAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, addr)
	suite.Require().NotNil(contractAcc)
	suite.Require().Equal(uint64(1), contractAcc.Nonce, "EIP-161: CREATE should increment nonce by 1 over default value (0)")

	_, rsp, err := suite.CallContract(
		testutil.StateTestContract,
		addr,
		common.Big0,
		"tempChangeEmpty",
		big.NewInt(1000),
	)
	suite.Require().NoError(err)
	suite.Require().Emptyf(rsp.VmError, "contract call should not fail: %s", rsp.VmError)

	storageAfter := suite.GetAllAccountStorage(suite.Ctx, addr)
	suite.Equal(storageBefore, storageAfter)
	// 1 key because of the non-empty test, empty test should always be empty
	// even when the contract tries to write 0x0 empty hash.
	suite.Len(storageAfter, 1, "storage should have one key")
}

func (suite *IntegrationTestSuite) TestState_NonEmptyToNonEmpty() {
	addr := suite.DeployContract(testutil.StateTestContract)
	storageBefore := suite.GetAllAccountStorage(suite.Ctx, addr)

	newValue := big.NewInt(1000)
	valueKey := "0x0000000000000000000000000000000000000000000000000000000000000001"

	_, rsp, err := suite.CallContract(
		testutil.StateTestContract,
		addr,
		common.Big0,
		"tempChangeNonEmpty",
		newValue,
	)
	suite.Require().NoError(err)
	suite.Require().Emptyf(rsp.VmError, "contract call should not fail: %s", rsp.VmError)

	storageAfter := suite.GetAllAccountStorage(suite.Ctx, addr)

	// Check that the value at the key has been updated to the new value
	storageBefore[common.HexToHash(valueKey)] = common.BytesToHash(newValue.Bytes())

	suite.Equal(storageBefore, storageAfter)
}

// ----------------------------------------------------------------------------
// EIP-161
// a) Nonce == 1 on CREATE
// b) CALL & SUICIDE charges 25k gas on >0 value && destination non-existent || empty
// - REVERT
// c) Non-existent accounts stay non-existent
// d) Touched accounts that are empty at the end of the tx are deleted
// - REVERT also reverts account deletions

func (suite *IntegrationTestSuite) TestEIP158_Enabled() {
	// Params.ChainConfig.EIP158Block needs to be set for
	// EIP-161 to be applied in geth EVM.Call()
	params := suite.App.EvmKeeper.GetParams(suite.Ctx)
	suite.Require().LessOrEqual(
		params.ChainConfig.EIP158Block.Int64(),
		suite.Ctx.BlockHeight(),
		"EIP-158 should be enabled in params",
	)
}

// EIP-161 a)
func (suite *IntegrationTestSuite) TestEIP161_CreateNonce() {
	addr := suite.DeployContract(testutil.EIP161TestContract)
	contractAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, addr)
	suite.Require().NotNil(contractAcc)
	suite.Require().Equal(
		uint64(1),
		contractAcc.Nonce,
		"EIP-161: CREATE should increment nonce by 1 over default value (0)",
	)
}

// EIP-161 b)
func (suite *IntegrationTestSuite) TestEIP161_CallGas() {
	addr := suite.DeployContract(testutil.EIP161TestContract)

	// Non-existent account
	targetAddr := common.Address{1, 2, 3, 4, 5, 6}
	targetAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, targetAddr)
	suite.Require().Nil(targetAcc, "target should not exist")

	var gasUsed1 uint64

	suite.Run("Call 0 value - no 25k gas charge", func() {
		_, rsp, err := suite.CallContract(
			testutil.EIP161TestContract,
			addr,
			common.Big0, // no transfer
			"callAccount",
			targetAddr,
		)
		suite.Require().NoError(err)
		suite.Require().Empty(rsp.VmError)

		targetAcc = suite.App.EvmKeeper.GetAccount(suite.Ctx, targetAddr)
		suite.Require().Nil(
			targetAcc,
			"target should still not exist after being called with no value",
		)

		gasUsed1 = rsp.GasUsed
	})

	suite.Run("Call >0 value - 25k gas charge", func() {
		suite.MintCoinsForAccount(
			suite.Ctx,
			sdk.AccAddress(suite.Address.Bytes()),
			sdk.NewCoins(
				sdk.NewCoin(types.AttoPhoton, sdk.NewInt(100000000000)),
			),
		)

		value := big.NewInt(10000)

		_, rsp, err := suite.CallContract(
			testutil.EIP161TestContract,
			addr,
			value, // >0 value transfer
			"callAccount",
			targetAddr,
		)
		suite.Require().NoError(err)
		suite.Require().Empty(rsp.VmError)

		// Check account + bal
		targetAcc = suite.App.EvmKeeper.GetAccount(suite.Ctx, targetAddr)
		suite.Require().NotNil(
			targetAcc,
			"target should exist after call with value",
		)

		bal := suite.App.EvmKeeper.GetBalance(suite.Ctx, targetAddr)
		suite.Require().Equal(
			value,
			bal,
			"target should have received the value",
		)

		// Check gas
		suite.Require().Greater(
			rsp.GasUsed,
			gasUsed1,
			"call with value transfer should use more gas than 0 value call",
		)
		suite.Require().GreaterOrEqual(
			rsp.GasUsed,
			gasUsed1+gethparams.CallNewAccountGas,
			"EIP-161: 25k gas charge when transferring >0 value & destination is dead",
		)
	})
}

// EIP-161 b)
// Same as TestEIP161_CallGas but with self destruct
func (suite *IntegrationTestSuite) TestEIP161_SuicideGas() {
	suite.MintCoinsForAccount(
		suite.Ctx,
		sdk.AccAddress(suite.Address.Bytes()),
		sdk.NewCoins(
			sdk.NewCoin(types.AttoPhoton, sdk.NewInt(100000000000)),
		),
	)

	addr := suite.DeployContract(testutil.EIP161TestContract)

	// Non-existent account
	targetAddr := common.Address{10}
	targetAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, targetAddr)
	suite.Require().Nil(targetAcc, "target should not exist")

	value := big.NewInt(10000)

	var gasUsed1 uint64

	suite.Run("suicide - non-existent destination", func() {
		_, rsp, err := suite.CallContract(
			testutil.EIP161TestContract,
			addr,
			value, // >0 value transfer
			"selfDestructTo",
			targetAddr,
		)
		suite.Require().NoError(err)
		suite.Require().Empty(rsp.VmError)

		contractAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, addr)
		suite.Require().Nil(contractAcc, "self destructed contract should be deleted")

		gasUsed1 = rsp.GasUsed
	})

	// Deploy again since contract self destructed
	addr = suite.DeployContract(testutil.EIP161TestContract)

	suite.Run("suicide - existing destination", func() {
		targetAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, targetAddr)
		targetBal := suite.App.EvmKeeper.GetBalance(suite.Ctx, targetAddr)
		suite.Require().NotNil(targetAcc, "target should exist")

		_, rsp, err := suite.CallContract(
			testutil.EIP161TestContract,
			addr,
			value, // >0 value transfer
			"selfDestructTo",
			targetAddr, // same target
		)
		suite.Require().NoError(err)
		suite.Require().Empty(rsp.VmError)

		contractAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, addr)
		suite.Require().Nil(contractAcc, "self destructed contract should be deleted")

		// Check destination balances
		targetBalAfter := suite.App.EvmKeeper.GetBalance(suite.Ctx, targetAddr)
		suite.Require().Equal(
			new(big.Int).Add(targetBal, value),
			targetBalAfter,
			"balance should be transferred",
		)

		// Check gas
		suite.Require().Greater(
			gasUsed1,
			rsp.GasUsed,
			"first call should use more gas - created account",
		)
		suite.Require().GreaterOrEqual(
			gasUsed1,
			rsp.GasUsed+gethparams.CallNewAccountGas,
			"EIP-161: additional 25k gas charge for first call",
		)
	})
}

// EIP-161 d)
func (suite *IntegrationTestSuite) TestEIP161_TouchEmptyDeletes() {
	// From EIP-161 point D:
	// At the end of the transaction, any account touched by the execution of
	// that transaction which is now empty SHALL instead become non-existent
	// (i.e. deleted).

	// Where:
	// An account is considered to be touched when it is involved in any
	// potentially state-changing operation. This includes, but is not limited
	// to, being the recipient of a transfer of zero value.

	// An account changes state when:
	// 1) it is the target or refund of a SUICIDE operation for zero or more
	// value;
	// 2) it is the source or destination of a CALL operation or message-call
	// transaction transferring zero or more value;
	// 3) it is the source or creation of a CREATE operation or
	// contract-creation transaction endowing zero or more value;

	// Not relevant:
	// 4) as the block author (“miner”) it is the recipient of block-rewards or
	// transaction-fees of zero or more value.

	// Some cases aren't relevant when clearing state. e.g. as an EOA
	// sending a tx, I will never be empty as my nonce increments.
	// Account state changes with a >0 value will also *not* clear the account
	// as it will be non-empty with a >0 balance.

	var contractAddr common.Address
	targetAddr := common.Address{10}

	type accountState struct {
		contractDeleted bool
		targetDeleted   bool
	}

	tests := []struct {
		name             string
		malleate         func()
		wantAccountState accountState
	}{
		{
			"not touched",
			func() {
				_, rsp, err := suite.CallContract(
					testutil.EIP161TestContract,
					contractAddr,
					common.Big0,
					"selfDestructTo",
					common.Address{20}, // unrelated account
				)
				suite.Require().NoError(err)
				suite.Require().Empty(rsp.VmError)
			},
			accountState{
				contractDeleted: true,
				targetDeleted:   false,
			},
		},
		{
			"self destruct beneficiary - 0 value",
			func() {
				// beneficiary account with 0 value should be touched
				// (self destruct sends the remaining funds to the beneficiary)
				_, rsp, err := suite.CallContract(
					testutil.EIP161TestContract,
					contractAddr,
					common.Big0,
					"selfDestructTo",
					targetAddr,
				)
				suite.Require().NoError(err)
				suite.Require().Empty(rsp.VmError)
			},
			accountState{
				contractDeleted: true,
				targetDeleted:   true,
			},
		},
		{
			"REVERTED: self destruct beneficiary - 0 value",
			func() {
				// beneficiary account with 0 value should be touched
				_, rsp, err := suite.CallContractWithGas(
					testutil.EIP161TestContract,
					contractAddr,
					common.Big0,
					28211, // Enough intrinsic gas, but not enough for entire tx
					"selfDestructToRevert",
					targetAddr,
				)
				suite.T().Logf("rsp: %+v", rsp.GasUsed)
				suite.Require().NoError(err, "revert should be in rsp, not err")
				suite.Require().Equal(vm.ErrOutOfGas.Error(), rsp.VmError)
			},
			accountState{
				// Nothing is deleted on revert
				contractDeleted: false,
				targetDeleted:   false,
			},
		},
		{
			"call target - 0 value",
			func() {
				// target account with 0 value should be touched
				_, rsp, err := suite.CallContract(
					testutil.EIP161TestContract,
					contractAddr,
					common.Big0,
					"callAccount",
					targetAddr,
				)
				suite.Require().NoError(err)
				suite.Require().Empty(rsp.VmError)
			},
			accountState{
				contractDeleted: false,
				targetDeleted:   true,
			},
		},
		{
			"REVERTED: call target - 0 value",
			func() {
				_, err := suite.EstimateCallGas(
					testutil.EIP161TestContract,
					contractAddr,
					common.Big0,
					"callAccountRevert",
					targetAddr,
				)
				suite.Require().Error(err, "estimate gas should fail since it will revert")

				_, rsp, err := suite.CallContractWithGas(
					testutil.EIP161TestContract,
					contractAddr,
					common.Big0,
					29277, // Enough gas to cover tx
					"callAccountRevert",
					targetAddr,
				)
				suite.Require().NoError(err, "revert should be in rsp, not err")
				suite.Require().Equal(vm.ErrExecutionReverted.Error(), rsp.VmError)
			},
			accountState{
				contractDeleted: false,
				// also reverts account deletion, so target should not be deleted
				targetDeleted: false,
			},
		},
		{
			"create call",
			func() {
				// target account with 0 value should be touched
				_, rsp, err := suite.CallContract(
					testutil.EIP161TestContract,
					contractAddr,
					common.Big0,
					"createContract",
				)
				suite.Require().NoError(err)
				suite.Require().Empty(rsp.VmError)
				suite.Require().NotEmpty(rsp.Ret)

				// Get the address of the created contract
				contractAddr = common.BytesToAddress(rsp.Ret)
				contractAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, contractAddr)
				suite.Require().NotNil(contractAcc)
				suite.Require().Equal(
					uint64(1),
					contractAcc.Nonce,
					"CREATE should increment nonce by 1 over default value (0)",
				)
			},
			accountState{
				contractDeleted: false,
				targetDeleted:   false,
			},
		},
		{
			"transfer zero amount",
			func() {
				// native transfer of 0 value funds
				_, rsp, err := suite.TransferValue(
					suite.Address,
					targetAddr,
					common.Big0,
				)
				suite.Require().NoError(err)
				suite.Require().Empty(rsp.VmError)
			},
			accountState{
				contractDeleted: false,
				targetDeleted:   true,
			},
		},
		{
			"REVERTED: transfer zero amount",
			func() {
				// Transfers funds from the account -> contract -> target, then
				// reverts after transfer.
				// Use a custom contract method so we can easily revert the
				// transfer after the transfer.
				_, rsp, err := suite.CallContractWithGas(
					testutil.EIP161TestContract,
					contractAddr,
					common.Big0,
					29277, // Enough gas to cover tx - estimategas will fail due to revert
					"transferValueRevert",
					targetAddr,
				)
				suite.Require().NoError(err, "revert should be in rsp, not err")
				suite.Require().Equal(vm.ErrExecutionReverted.Error(), rsp.VmError)
			},
			accountState{
				contractDeleted: false,
				targetDeleted:   false,
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Setup test
			suite.SetupTest()

			// Deploy contract
			contractAddr = suite.DeployContract(testutil.EIP161TestContract)

			// Create empty account
			acc := statedb.NewEmptyAccount()
			err := suite.App.EvmKeeper.SetAccount(suite.Ctx, targetAddr, *acc)
			suite.Require().NoError(err, "empty target should be created")

			targetAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, targetAddr)
			suite.Require().NotNil(targetAcc, "empty account should exist")
			suite.Require().Equal(acc, targetAcc)

			// Run test specific setup
			tt.malleate()

			// Check result
			targetAcc = suite.App.EvmKeeper.GetAccount(suite.Ctx, targetAddr)
			if tt.wantAccountState.targetDeleted {
				suite.Require().Nil(
					targetAcc,
					"EIP-161: empty account should be deleted after being touched",
				)
			} else {
				suite.Require().NotNil(
					targetAcc,
					"EIP-161: empty account should not be deleted if not touched or reverted",
				)
			}

			contractAcc := suite.App.EvmKeeper.GetAccount(suite.Ctx, contractAddr)
			if tt.wantAccountState.contractDeleted {
				suite.Require().Nil(
					contractAcc,
					"EIP-161: contract should be deleted after touching empty account",
				)
			} else {
				suite.Require().NotNil(
					contractAcc,
					"EIP-161: contract should not be deleted if not touching empty account or reverted",
				)
			}
		})
	}
}
