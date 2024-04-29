package keeper_test

import (
	_ "embed"
	"math/big"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/testutil"

	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm/types"

	"github.com/ethereum/go-ethereum/common"

	abci "github.com/cometbft/cometbft/abci/types"
)

var testTokens = sdkmath.NewIntWithDecimal(1000, 18)

type KeeperTestSuite struct {
	testutil.TestSuite
}

var s *KeeperTestSuite

func TestKeeperTestSuite(t *testing.T) {
	if os.Getenv("benchmark") != "" {
		t.Skip("Skipping Gingko Test")
	}
	s = new(KeeperTestSuite)
	s.EnableFeemarket = false
	s.EnableLondonHF = true
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperTestSuite) TestBaseFee() {
	testCases := []struct {
		name            string
		enableLondonHF  bool
		enableFeemarket bool
		expectBaseFee   *big.Int
	}{
		{"not enable london HF, not enable feemarket", false, false, nil},
		{"enable london HF, not enable feemarket", true, false, big.NewInt(0)},
		{"enable london HF, enable feemarket", true, true, big.NewInt(1000000000)},
		{"not enable london HF, enable feemarket", false, true, nil},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.EnableFeemarket = tc.enableFeemarket
			suite.EnableLondonHF = tc.enableLondonHF
			suite.SetupTest()
			suite.App.EvmKeeper.BeginBlock(suite.Ctx, abci.RequestBeginBlock{})
			params := suite.App.EvmKeeper.GetParams(suite.Ctx)
			ethCfg := params.ChainConfig.EthereumConfig(suite.App.EvmKeeper.ChainID())
			baseFee := suite.App.EvmKeeper.GetBaseFee(suite.Ctx, ethCfg)
			suite.Require().Equal(tc.expectBaseFee, baseFee)
		})
	}
	suite.EnableFeemarket = false
	suite.EnableLondonHF = true
}

func (suite *KeeperTestSuite) TestGetAccountStorage() {
	testCases := []struct {
		name     string
		malleate func()
		expRes   []int
	}{
		{
			"Only one account that's not a contract (no storage)",
			func() {},
			[]int{0},
		},
		{
			"Two accounts - one contract (with storage), one wallet",
			func() {
				supply := big.NewInt(100)
				suite.DeployTestContract(suite.T(), suite.Address, supply)
			},
			[]int{2, 0},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()
			i := 0
			suite.App.AccountKeeper.IterateAccounts(suite.Ctx, func(account authtypes.AccountI) bool {
				ethAccount, ok := account.(ethermint.EthAccountI)
				if !ok {
					// ignore non EthAccounts
					return false
				}

				addr := ethAccount.EthAddress()
				storage := suite.App.EvmKeeper.GetAccountStorage(suite.Ctx, addr)

				suite.Require().Equal(tc.expRes[i], len(storage))
				i++
				return false
			})
		})
	}
}

func (suite *KeeperTestSuite) TestGetAccountOrEmpty() {
	empty := statedb.Account{
		// Balance:  new(big.Int),
		CodeHash: types.EmptyCodeHash,
	}

	supply := big.NewInt(100)
	contractAddr := suite.DeployTestContract(suite.T(), suite.Address, supply)

	testCases := []struct {
		name     string
		addr     common.Address
		expEmpty bool
	}{
		{
			"unexisting account - get empty",
			common.Address{},
			true,
		},
		{
			"existing contract account",
			contractAddr,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			res := suite.App.EvmKeeper.GetAccountOrEmpty(suite.Ctx, tc.addr)
			if tc.expEmpty {
				suite.Require().Equal(empty, res)
			} else {
				suite.Require().NotEqual(empty, res)
			}
		})
	}
}
