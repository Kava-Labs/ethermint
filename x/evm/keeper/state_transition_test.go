package keeper_test

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"

	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store/iavl"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/evmos/ethermint/tests"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
)

func (suite *KeeperTestSuite) TestGetHashFn() {
	header := suite.Ctx.BlockHeader()
	h, _ := tmtypes.HeaderFromProto(&header)
	hash := h.Hash()

	testCases := []struct {
		msg      string
		height   uint64
		malleate func()
		expHash  common.Hash
	}{
		{
			"case 1.1: context hash cached",
			uint64(suite.Ctx.BlockHeight()),
			func() {
				suite.Ctx = suite.Ctx.WithHeaderHash(tmhash.Sum([]byte("header")))
			},
			common.BytesToHash(tmhash.Sum([]byte("header"))),
		},
		{
			"case 1.2: failed to cast Tendermint header",
			uint64(suite.Ctx.BlockHeight()),
			func() {
				header := tmproto.Header{}
				header.Height = suite.Ctx.BlockHeight()
				suite.Ctx = suite.Ctx.WithBlockHeader(header)
			},
			common.Hash{},
		},
		{
			"case 1.3: hash calculated from Tendermint header",
			uint64(suite.Ctx.BlockHeight()),
			func() {
				suite.Ctx = suite.Ctx.WithBlockHeader(header)
			},
			common.BytesToHash(hash),
		},
		{
			"case 2.1: height lower than current one, hist info not found",
			1,
			func() {
				suite.Ctx = suite.Ctx.WithBlockHeight(10)
			},
			common.Hash{},
		},
		{
			"case 2.2: height lower than current one, invalid hist info header",
			1,
			func() {
				suite.App.StakingKeeper.SetHistoricalInfo(suite.Ctx, 1, &stakingtypes.HistoricalInfo{})
				suite.Ctx = suite.Ctx.WithBlockHeight(10)
			},
			common.Hash{},
		},
		{
			"case 2.3: height lower than current one, calculated from hist info header",
			1,
			func() {
				histInfo := &stakingtypes.HistoricalInfo{
					Header: header,
				}
				suite.App.StakingKeeper.SetHistoricalInfo(suite.Ctx, 1, histInfo)
				suite.Ctx = suite.Ctx.WithBlockHeight(10)
			},
			common.BytesToHash(hash),
		},
		{
			"case 3: height greater than current one",
			200,
			func() {},
			common.Hash{},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()

			hash := suite.App.EvmKeeper.GetHashFn(suite.Ctx)(tc.height)
			suite.Require().Equal(tc.expHash, hash)
		})
	}
}

func (suite *KeeperTestSuite) TestGetCoinbaseAddress() {
	valOpAddr := tests.GenerateAddress()

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"validator not found",
			func() {
				header := suite.Ctx.BlockHeader()
				header.ProposerAddress = []byte{}
				suite.Ctx = suite.Ctx.WithBlockHeader(header)
			},
			false,
		},
		{
			"success",
			func() {
				valConsAddr, privkey := tests.NewAddrKey()

				pkAny, err := codectypes.NewAnyWithValue(privkey.PubKey())
				suite.Require().NoError(err)

				validator := stakingtypes.Validator{
					OperatorAddress: sdk.ValAddress(valOpAddr.Bytes()).String(),
					ConsensusPubkey: pkAny,
				}

				suite.App.StakingKeeper.SetValidator(suite.Ctx, validator)
				err = suite.App.StakingKeeper.SetValidatorByConsAddr(suite.Ctx, validator)
				suite.Require().NoError(err)

				header := suite.Ctx.BlockHeader()
				header.ProposerAddress = valConsAddr.Bytes()
				suite.Ctx = suite.Ctx.WithBlockHeader(header)

				_, found := suite.App.StakingKeeper.GetValidatorByConsAddr(suite.Ctx, valConsAddr.Bytes())
				suite.Require().True(found)

				suite.Require().NotEmpty(suite.Ctx.BlockHeader().ProposerAddress)
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			proposerAddress := suite.Ctx.BlockHeader().ProposerAddress
			coinbase, err := suite.App.EvmKeeper.GetCoinbaseAddress(suite.Ctx, sdk.ConsAddress(proposerAddress))
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(valOpAddr, coinbase)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetEthIntrinsicGas() {
	testCases := []struct {
		name               string
		data               []byte
		accessList         ethtypes.AccessList
		height             int64
		isContractCreation bool
		noError            bool
		expGas             uint64
	}{
		{
			"no data, no accesslist, not contract creation, not homestead, not istanbul",
			nil,
			nil,
			1,
			false,
			true,
			params.TxGas,
		},
		{
			"with one zero data, no accesslist, not contract creation, not homestead, not istanbul",
			[]byte{0},
			nil,
			1,
			false,
			true,
			params.TxGas + params.TxDataZeroGas*1,
		},
		{
			"with one non zero data, no accesslist, not contract creation, not homestead, not istanbul",
			[]byte{1},
			nil,
			1,
			true,
			true,
			params.TxGas + params.TxDataNonZeroGasFrontier*1,
		},
		{
			"no data, one accesslist, not contract creation, not homestead, not istanbul",
			nil,
			[]ethtypes.AccessTuple{
				{},
			},
			1,
			false,
			true,
			params.TxGas + params.TxAccessListAddressGas,
		},
		{
			"no data, one accesslist with one storageKey, not contract creation, not homestead, not istanbul",
			nil,
			[]ethtypes.AccessTuple{
				{StorageKeys: make([]common.Hash, 1)},
			},
			1,
			false,
			true,
			params.TxGas + params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas*1,
		},
		{
			"no data, no accesslist, is contract creation, is homestead, not istanbul",
			nil,
			nil,
			2,
			true,
			true,
			params.TxGasContractCreation,
		},
		{
			"with one zero data, no accesslist, not contract creation, is homestead, is istanbul",
			[]byte{1},
			nil,
			3,
			false,
			true,
			params.TxGas + params.TxDataNonZeroGasEIP2028*1,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			params := suite.App.EvmKeeper.GetParams(suite.Ctx)
			ethCfg := params.ChainConfig.EthereumConfig(suite.App.EvmKeeper.ChainID())
			ethCfg.HomesteadBlock = big.NewInt(2)
			ethCfg.IstanbulBlock = big.NewInt(3)
			signer := ethtypes.LatestSignerForChainID(suite.App.EvmKeeper.ChainID())

			suite.Ctx = suite.Ctx.WithBlockHeight(tc.height)

			nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)
			m, err := newNativeMessage(
				nonce,
				suite.Ctx.BlockHeight(),
				suite.Address,
				ethCfg,
				suite.Signer,
				signer,
				ethtypes.AccessListTxType,
				tc.data,
				tc.accessList,
			)
			suite.Require().NoError(err)

			gas, err := suite.App.EvmKeeper.GetEthIntrinsicGas(suite.Ctx, m, ethCfg, tc.isContractCreation)
			if tc.noError {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			suite.Require().Equal(tc.expGas, gas)
		})
	}
}

func (suite *KeeperTestSuite) TestGasToRefund() {
	testCases := []struct {
		name           string
		gasconsumed    uint64
		refundQuotient uint64
		expGasRefund   uint64
		expPanic       bool
	}{
		{
			"gas refund 5",
			5,
			1,
			5,
			false,
		},
		{
			"gas refund 10",
			10,
			1,
			10,
			false,
		},
		{
			"gas refund availableRefund",
			11,
			1,
			10,
			false,
		},
		{
			"gas refund quotient 0",
			11,
			0,
			0,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.MintFeeCollector = true
			suite.SetupTest() // reset
			vmdb := suite.StateDB()
			vmdb.AddRefund(10)

			if tc.expPanic {
				panicF := func() {
					keeper.GasToRefund(vmdb.GetRefund(), tc.gasconsumed, tc.refundQuotient)
				}
				suite.Require().Panics(panicF)
			} else {
				gr := keeper.GasToRefund(vmdb.GetRefund(), tc.gasconsumed, tc.refundQuotient)
				suite.Require().Equal(tc.expGasRefund, gr)
			}
		})
	}
	suite.MintFeeCollector = false
}

func (suite *KeeperTestSuite) TestRefundGas() {
	var (
		m   core.Message
		err error
	)

	testCases := []struct {
		name           string
		leftoverGas    uint64
		refundQuotient uint64
		noError        bool
		expGasRefund   uint64
		malleate       func()
	}{
		{
			name:           "leftoverGas more than tx gas limit",
			leftoverGas:    params.TxGas + 1,
			refundQuotient: params.RefundQuotient,
			noError:        false,
			expGasRefund:   params.TxGas + 1,
		},
		{
			name:           "leftoverGas equal to tx gas limit, insufficient fee collector account",
			leftoverGas:    params.TxGas,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   0,
		},
		{
			name:           "leftoverGas less than to tx gas limit",
			leftoverGas:    params.TxGas - 1,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   0,
		},
		{
			name:           "no leftoverGas, refund half used gas ",
			leftoverGas:    0,
			refundQuotient: params.RefundQuotient,
			noError:        true,
			expGasRefund:   params.TxGas / params.RefundQuotient,
		},
		{
			name:           "invalid Gas value in msg",
			leftoverGas:    0,
			refundQuotient: params.RefundQuotient,
			noError:        false,
			expGasRefund:   params.TxGas,
			malleate: func() {
				keeperParams := suite.App.EvmKeeper.GetParams(suite.Ctx)
				m, err = suite.createContractGethMsg(
					suite.StateDB().GetNonce(suite.Address),
					ethtypes.LatestSignerForChainID(suite.App.EvmKeeper.ChainID()),
					keeperParams.ChainConfig.EthereumConfig(suite.App.EvmKeeper.ChainID()),
					big.NewInt(-100),
				)
				suite.Require().NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.MintFeeCollector = true
			suite.SetupTest() // reset

			keeperParams := suite.App.EvmKeeper.GetParams(suite.Ctx)
			ethCfg := keeperParams.ChainConfig.EthereumConfig(suite.App.EvmKeeper.ChainID())
			signer := ethtypes.LatestSignerForChainID(suite.App.EvmKeeper.ChainID())
			vmdb := suite.StateDB()

			m, err = newNativeMessage(
				vmdb.GetNonce(suite.Address),
				suite.Ctx.BlockHeight(),
				suite.Address,
				ethCfg,
				suite.Signer,
				signer,
				ethtypes.AccessListTxType,
				nil,
				nil,
			)
			suite.Require().NoError(err)

			vmdb.AddRefund(params.TxGas)

			if tc.leftoverGas > m.Gas() {
				return
			}

			if tc.malleate != nil {
				tc.malleate()
			}

			gasUsed := m.Gas() - tc.leftoverGas
			refund := keeper.GasToRefund(vmdb.GetRefund(), gasUsed, tc.refundQuotient)
			suite.Require().Equal(tc.expGasRefund, refund)

			err = suite.App.EvmKeeper.RefundGas(suite.Ctx, m, refund, "aphoton")
			if tc.noError {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
	suite.MintFeeCollector = false
}

func (suite *KeeperTestSuite) TestResetGasMeterAndConsumeGas() {
	testCases := []struct {
		name        string
		gasConsumed uint64
		gasUsed     uint64
		expPanic    bool
	}{
		{
			"gas consumed 5, used 5",
			5,
			5,
			false,
		},
		{
			"gas consumed 5, used 10",
			5,
			10,
			false,
		},
		{
			"gas consumed 10, used 10",
			10,
			10,
			false,
		},
		{
			"gas consumed 11, used 10, NegativeGasConsumed panic",
			11,
			10,
			true,
		},
		{
			"gas consumed 1, used 10, overflow panic",
			1,
			math.MaxUint64,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			panicF := func() {
				gm := sdk.NewGasMeter(10)
				gm.ConsumeGas(tc.gasConsumed, "")
				ctx := suite.Ctx.WithGasMeter(gm)
				suite.App.EvmKeeper.ResetGasMeterAndConsumeGas(ctx, tc.gasUsed)
			}

			if tc.expPanic {
				suite.Require().Panics(panicF)
			} else {
				suite.Require().NotPanics(panicF)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestEVMConfig() {
	proposerAddress := suite.Ctx.BlockHeader().ProposerAddress
	cfg, err := suite.App.EvmKeeper.EVMConfig(suite.Ctx, proposerAddress, big.NewInt(9000))
	suite.Require().NoError(err)
	suite.Require().Equal(types.DefaultParams(), cfg.Params)
	// london hardfork is enabled by default
	suite.Require().Equal(big.NewInt(0), cfg.BaseFee)
	suite.Require().Equal(suite.Address, cfg.CoinBase)
	suite.Require().Equal(types.DefaultParams().ChainConfig.EthereumConfig(big.NewInt(9000)), cfg.ChainConfig)
}

func (suite *KeeperTestSuite) TestContractDeployment() {
	contractAddress := suite.DeployTestContract(suite.T(), suite.Address, big.NewInt(10000000000000))
	db := suite.StateDB()
	suite.Require().Greater(db.GetCodeSize(contractAddress), 0)
}

func (suite *KeeperTestSuite) TestApplyMessage() {
	expectedGasUsed := params.TxGas
	var msg core.Message

	proposerAddress := suite.Ctx.BlockHeader().ProposerAddress
	config, err := suite.App.EvmKeeper.EVMConfig(suite.Ctx, proposerAddress, big.NewInt(9000))
	suite.Require().NoError(err)

	keeperParams := suite.App.EvmKeeper.GetParams(suite.Ctx)
	chainCfg := keeperParams.ChainConfig.EthereumConfig(suite.App.EvmKeeper.ChainID())
	signer := ethtypes.LatestSignerForChainID(suite.App.EvmKeeper.ChainID())
	tracer := suite.App.EvmKeeper.Tracer(suite.Ctx, msg, config.ChainConfig)
	vmdb := suite.StateDB()

	msg, err = newNativeMessage(
		vmdb.GetNonce(suite.Address),
		suite.Ctx.BlockHeight(),
		suite.Address,
		chainCfg,
		suite.Signer,
		signer,
		ethtypes.AccessListTxType,
		nil,
		nil,
	)
	suite.Require().NoError(err)

	res, err := suite.App.EvmKeeper.ApplyMessage(suite.Ctx, msg, tracer, true)

	suite.Require().NoError(err)
	suite.Require().Equal(expectedGasUsed, res.GasUsed)
	suite.Require().False(res.Failed())
}

func (suite *KeeperTestSuite) TestApplyMessageWithConfig() {
	var (
		msg             core.Message
		err             error
		expectedGasUsed uint64
		config          *statedb.EVMConfig
		keeperParams    types.Params
		signer          ethtypes.Signer
		vmdb            *statedb.StateDB
		txConfig        statedb.TxConfig
		chainCfg        *params.ChainConfig
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   bool
	}{
		{
			"messsage applied ok",
			func() {
				msg, err = newNativeMessage(
					vmdb.GetNonce(suite.Address),
					suite.Ctx.BlockHeight(),
					suite.Address,
					chainCfg,
					suite.Signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"call contract tx with config param EnableCall = false",
			func() {
				config.Params.EnableCall = false
				msg, err = newNativeMessage(
					vmdb.GetNonce(suite.Address),
					suite.Ctx.BlockHeight(),
					suite.Address,
					chainCfg,
					suite.Signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"create contract tx with config param EnableCreate = false",
			func() {
				msg, err = suite.createContractGethMsg(vmdb.GetNonce(suite.Address), signer, chainCfg, big.NewInt(1))
				suite.Require().NoError(err)
				config.Params.EnableCreate = false
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			expectedGasUsed = params.TxGas

			proposerAddress := suite.Ctx.BlockHeader().ProposerAddress
			config, err = suite.App.EvmKeeper.EVMConfig(suite.Ctx, proposerAddress, big.NewInt(9000))
			suite.Require().NoError(err)

			keeperParams = suite.App.EvmKeeper.GetParams(suite.Ctx)
			chainCfg = keeperParams.ChainConfig.EthereumConfig(suite.App.EvmKeeper.ChainID())
			signer = ethtypes.LatestSignerForChainID(suite.App.EvmKeeper.ChainID())
			vmdb = suite.StateDB()
			txConfig = suite.App.EvmKeeper.TxConfig(suite.Ctx, common.Hash{})

			tc.malleate()
			res, err := suite.App.EvmKeeper.ApplyMessageWithConfig(suite.Ctx, msg, nil, true, config, txConfig)

			if tc.expErr {
				suite.Require().Error(err)
				return
			}

			suite.Require().NoError(err)
			suite.Require().False(res.Failed())
			suite.Require().Equal(expectedGasUsed, res.GasUsed)
		})
	}
}

func (suite *KeeperTestSuite) createContractGethMsg(nonce uint64, signer ethtypes.Signer, cfg *params.ChainConfig, gasPrice *big.Int) (core.Message, error) {
	ethMsg, err := suite.createContractMsgTx(nonce, signer, cfg, gasPrice)
	if err != nil {
		return nil, err
	}

	msgSigner := ethtypes.MakeSigner(cfg, big.NewInt(suite.Ctx.BlockHeight()))
	return ethMsg.AsMessage(msgSigner, nil)
}

func (suite *KeeperTestSuite) createContractMsgTx(nonce uint64, signer ethtypes.Signer, cfg *params.ChainConfig, gasPrice *big.Int) (*types.MsgEthereumTx, error) {
	contractCreateTx := &ethtypes.AccessListTx{
		GasPrice: gasPrice,
		Gas:      params.TxGasContractCreation * 2,
		To:       nil,
		// Minimal contract data
		// https://ethereum.stackexchange.com/questions/40757/what-is-the-shortest-bytecode-that-will-publish-a-contract-with-non-zero-bytecod
		// Using the previous string "contract_data" as contract code may cause
		// an error as it includes 0x5f which is PUSH0, only on Shanghai and later
		Data:  common.Hex2Bytes("0x3859818153F3"),
		Nonce: nonce,
	}
	ethTx := ethtypes.NewTx(contractCreateTx)
	ethMsg := &types.MsgEthereumTx{}
	ethMsg.FromEthereumTx(ethTx)
	ethMsg.From = suite.Address.Hex()

	return ethMsg, ethMsg.Sign(signer, suite.Signer)
}

func (suite *KeeperTestSuite) TestGetProposerAddress() {
	var a sdk.ConsAddress
	address := sdk.ConsAddress(suite.Address.Bytes())
	proposerAddress := sdk.ConsAddress(suite.Ctx.BlockHeader().ProposerAddress)
	testCases := []struct {
		msg    string
		adr    sdk.ConsAddress
		expAdr sdk.ConsAddress
	}{
		{
			"proposer address provided",
			address,
			address,
		},
		{
			"nil proposer address provided",
			nil,
			proposerAddress,
		},
		{
			"typed nil proposer address provided",
			a,
			proposerAddress,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.Require().Equal(tc.expAdr, keeper.GetProposerAddress(suite.Ctx, tc.adr))
		})
	}
}

var (
	blockHash     common.Hash      = common.BigToHash(big.NewInt(9999))
	emptyTxConfig statedb.TxConfig = statedb.NewEmptyTxConfig(blockHash)
)

type StateDBCommit interface {
	vm.StateDB
	Commit() error
}

func (suite *KeeperTestSuite) GetStoreCommitHashes(
	storeKeys []string,
) map[string]string {
	hashes := make(map[string]string)

	for _, storeKey := range storeKeys {
		key := suite.App.GetKey(storeKey)
		store := suite.App.CommitMultiStore().GetStore(key)
		iavlStore := store.(*iavl.Store)
		commitID := iavlStore.LastCommitID()

		hashes[storeKey] = hex.EncodeToString(commitID.Hash)
	}

	return hashes
}

func (suite *KeeperTestSuite) TestNoopStateChange_UnmodifiedIAVLTree() {
	// On StateDB.Commit(), if there is a dirty state change that matches the
	// committed state, it should be skipped. Only state changes that are
	// different from committed state should be applied to the underlying store.
	// Corresponding journal based StateDB code:
	// https://github.com/ethereum/go-ethereum/blob/e31709db6570e302557a9bccd681034ea0dcc246/core/state/state_object.go#L302-L305
	// https://github.com/Kava-Labs/ethermint/blob/877e8fd1bd140c37ad05ed613f31e28f0130c0c4/x/evm/statedb/statedb.go#L469-L472

	// Even with store.Set() on the same pre-existing key and value, it will
	// update the underlying iavl.Node version and thus the node hash, parent
	// hashes, and the commitID

	// E.g. SetState(A, B) -> xxx -> SetState(A, B) should not be applied to
	// the underlying store.
	// xxx could be 0 or more state changes to the same key. It can be different
	// values since the only value that actually matters is the last one when
	// Commit() is called.

	addr := common.BigToAddress(big.NewInt(1))
	key := common.BigToHash(big.NewInt(10))
	value := common.BigToHash(big.NewInt(20))

	tests := []struct {
		name              string
		affectedStoreKeys []string
		initializeState   func(vmdb vm.StateDB)
		maleate           func(vmdb vm.StateDB)
	}{
		{
			"SetState - no extra snapshots",
			[]string{types.StoreKey},
			func(vmdb vm.StateDB) {
				vmdb.SetState(addr, key, value)
			},
			func(vmdb vm.StateDB) {
				vmdb.SetState(addr, key, value)
			},
		},
		{
			"SetState - 2nd snapshot, same value",
			[]string{types.StoreKey},
			func(vmdb vm.StateDB) {
				vmdb.SetState(addr, key, value)
			},
			func(vmdb vm.StateDB) {
				// A -> A
				vmdb.SetState(addr, key, value)

				// Same value, just different snapshot. Should be skipped in all
				// snapshots.
				_ = vmdb.Snapshot()
				vmdb.SetState(addr, key, value)
			},
		},
		{
			"SetState - 2nd snapshot, different value",
			[]string{types.StoreKey},
			func(vmdb vm.StateDB) {
				vmdb.SetState(addr, key, value)
			},
			func(vmdb vm.StateDB) {
				// A -> B -> A

				// Different value in 1st snapshot
				value2 := common.BigToHash(big.NewInt(30))
				vmdb.SetState(addr, key, value2)

				// Back to original value in 2nd snapshot
				_ = vmdb.Snapshot()
				vmdb.SetState(addr, key, value)
			},
		},
		{
			"SetState - multiple snapshots, different value",
			[]string{types.StoreKey},
			func(vmdb vm.StateDB) {
				vmdb.SetState(addr, key, value)
			},
			func(vmdb vm.StateDB) {
				// A -> B -> C -> A

				// Different value in 1st snapshot
				value2 := common.BigToHash(big.NewInt(30))
				value3 := common.BigToHash(big.NewInt(40))
				_ = vmdb.Snapshot()
				vmdb.SetState(addr, key, value2)

				// Extra empty snapshot for good measure
				_ = vmdb.Snapshot()

				_ = vmdb.Snapshot()
				vmdb.SetState(addr, key, value3)

				// Back to original value in last snapshot
				_ = vmdb.Snapshot()
				vmdb.SetState(addr, key, value)
			},
		},
		{
			"no-op balance change",
			[]string{types.StoreKey, banktypes.ModuleName},
			func(vmdb vm.StateDB) {
				// Start with some non-zero balance
				vmdb.AddBalance(addr, big.NewInt(10))
			},
			func(vmdb vm.StateDB) {
				// No-op balance change
				vmdb.AddBalance(addr, big.NewInt(50))
				vmdb.SubBalance(addr, big.NewInt(40))
				vmdb.SubBalance(addr, big.NewInt(10))
			},
		},
		{
			"no-op balance change to zero",
			[]string{types.StoreKey, banktypes.ModuleName},
			func(vmdb vm.StateDB) {
				// Start with some non-zero balance
				vmdb.AddBalance(addr, big.NewInt(10))
			},
			func(vmdb vm.StateDB) {
				// No-op balance change
				vmdb.AddBalance(addr, big.NewInt(50))

				// Down to zero
				vmdb.SubBalance(addr, big.NewInt(60))
				suite.Require().Equal(int64(0), vmdb.GetBalance(addr).Int64())

				// Back to a non-zero balance same as before
				vmdb.AddBalance(addr, big.NewInt(10))
			},
		},
		{
			"no-op balance change with snapshots",
			[]string{types.StoreKey, banktypes.ModuleName},
			func(vmdb vm.StateDB) {
				// Start with some non-zero balance
				vmdb.AddBalance(addr, big.NewInt(10))
			},
			func(vmdb vm.StateDB) {
				// No-op balance change
				vmdb.AddBalance(addr, big.NewInt(50))

				_ = vmdb.Snapshot()
				vmdb.SubBalance(addr, big.NewInt(40))

				_ = vmdb.Snapshot()
				vmdb.SubBalance(addr, big.NewInt(10))
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// reset
			suite.SetupTest()

			db := statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
			tt.initializeState(db)

			suite.Require().NoError(db.Commit())
			suite.Commit()

			store := suite.App.CommitMultiStore().GetStore(suite.App.GetKey(types.StoreKey))
			iavlStore := store.(*iavl.Store)
			commitID1 := iavlStore.LastCommitID()

			iavlHashes1 := s.GetStoreCommitHashes(tt.affectedStoreKeys)

			// New statedb that should not modify the underlying store
			db = statedb.New(suite.Ctx, suite.App.EvmKeeper, emptyTxConfig)
			tt.maleate(db)

			suite.Require().NoError(db.Commit())
			// suite.Commit()

			commitID2 := iavlStore.LastCommitID()
			iavlHashes2 := s.GetStoreCommitHashes(tt.affectedStoreKeys)

			// We can compare the commitIDs since this is *only* the x/evm store which
			// doesn't change between blocks without state changes. Any version change,
			// e.g. no-op change that was written when it shouldn't, will modify the
			// hash.
			suite.Require().Equal(
				common.Bytes2Hex(commitID1.Hash),
				common.Bytes2Hex(commitID2.Hash),
				"evm store should be unchanged",
			)

			// Check all affected stores
			suite.Require().Equal(iavlHashes1, iavlHashes2)
		})
	}
}
