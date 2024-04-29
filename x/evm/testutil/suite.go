package testutil

import (
	"encoding/json"
	"math"
	"math/big"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/simapp"
	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	"github.com/evmos/ethermint/app"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	"github.com/evmos/ethermint/encoding"
	"github.com/evmos/ethermint/server/config"
	"github.com/evmos/ethermint/tests"
	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"
)

type TestSuite struct {
	suite.Suite

	Ctx         sdk.Context
	App         *app.EthermintApp
	QueryClient types.QueryClient
	Address     common.Address
	ConsAddress sdk.ConsAddress

	// for generate test tx
	ClientCtx client.Context
	EthSigner ethtypes.Signer

	AppCodec codec.Codec
	Signer   keyring.Signer

	EnableFeemarket  bool
	EnableLondonHF   bool
	MintFeeCollector bool
	Denom            string
}

func (suite *TestSuite) SetupTest() {
	checkTx := false
	suite.App = app.Setup(checkTx, nil)
	suite.SetupApp(checkTx)
}

func (suite *TestSuite) SetupTestWithT(t require.TestingT) {
	checkTx := false
	suite.App = app.Setup(checkTx, nil)
	suite.SetupAppWithT(checkTx, t)
}

func (suite *TestSuite) SetupApp(checkTx bool) {
	suite.SetupAppWithT(checkTx, suite.T())
}

// SetupApp setup test environment, it uses`require.TestingT` to support both `testing.T` and `testing.B`.
func (suite *TestSuite) SetupAppWithT(checkTx bool, t require.TestingT) {
	// account key, use a constant account to keep unit test deterministic.
	ecdsaPriv, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)
	priv := &ethsecp256k1.PrivKey{
		Key: crypto.FromECDSA(ecdsaPriv),
	}
	suite.Address = common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.Signer = tests.NewSigner(priv)

	suite.Require().Equal("0x71562b71999873DB5b286dF957af199Ec94617F7", suite.Address.Hex())

	// consensus key
	priv = &ethsecp256k1.PrivKey{
		Key: common.Hex2Bytes("a249d5fbd4516fde5765dbd763b93c3542bf2748b6cd512eb96e0862b3583261"),
	}
	require.NoError(t, err)
	suite.ConsAddress = sdk.ConsAddress(priv.PubKey().Address())

	suite.Require().Equal("cosmosvalcons1505eel9r7tnacxyzsqysysudwgucvhl53t7p70", suite.ConsAddress.String())

	suite.App = app.Setup(checkTx, func(app *app.EthermintApp, genesis simapp.GenesisState) simapp.GenesisState {
		feemarketGenesis := feemarkettypes.DefaultGenesisState()
		if suite.EnableFeemarket {
			feemarketGenesis.Params.EnableHeight = 1
			feemarketGenesis.Params.NoBaseFee = false
		} else {
			feemarketGenesis.Params.NoBaseFee = true
		}
		genesis[feemarkettypes.ModuleName] = app.AppCodec().MustMarshalJSON(feemarketGenesis)
		if !suite.EnableLondonHF {
			evmGenesis := types.DefaultGenesisState()
			maxInt := sdkmath.NewInt(math.MaxInt64)
			evmGenesis.Params.ChainConfig.LondonBlock = &maxInt
			evmGenesis.Params.ChainConfig.ArrowGlacierBlock = &maxInt
			evmGenesis.Params.ChainConfig.GrayGlacierBlock = &maxInt
			evmGenesis.Params.ChainConfig.MergeNetsplitBlock = &maxInt
			evmGenesis.Params.ChainConfig.ShanghaiBlock = &maxInt
			evmGenesis.Params.ChainConfig.CancunBlock = &maxInt
			genesis[types.ModuleName] = app.AppCodec().MustMarshalJSON(evmGenesis)
		}
		return genesis
	})

	if suite.MintFeeCollector {
		// mint some coin to fee collector
		coins := sdk.NewCoins(sdk.NewCoin(types.DefaultEVMDenom, sdkmath.NewInt(int64(params.TxGas)-1)))
		genesisState := app.NewTestGenesisState(suite.App.AppCodec())
		balances := []banktypes.Balance{
			{
				Address: suite.App.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName).String(),
				Coins:   coins,
			},
		}
		var bankGenesis banktypes.GenesisState
		suite.App.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenesis)
		// Update balances and total supply
		bankGenesis.Balances = append(bankGenesis.Balances, balances...)
		bankGenesis.Supply = bankGenesis.Supply.Add(coins...)
		genesisState[banktypes.ModuleName] = suite.App.AppCodec().MustMarshalJSON(&bankGenesis)

		// we marshal the genesisState of all module to a byte array
		stateBytes, err := tmjson.MarshalIndent(genesisState, "", " ")
		require.NoError(t, err)

		// Initialize the chain
		suite.App.InitChain(
			abci.RequestInitChain{
				ChainId:         "ethermint_9000-1",
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: app.DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
	}

	suite.Ctx = suite.App.BaseApp.NewContext(checkTx, tmproto.Header{
		Height:  1,
		ChainID: "ethermint_9000-1",
		// Fixed date to have deterministic tests
		Time:            time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
		ProposerAddress: suite.ConsAddress.Bytes(),
		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})

	queryHelper := baseapp.NewQueryServerTestHelper(suite.Ctx, suite.App.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.App.EvmKeeper)
	suite.QueryClient = types.NewQueryClient(queryHelper)

	acc := &ethermint.EthAccount{
		BaseAccount: authtypes.NewBaseAccount(sdk.AccAddress(suite.Address.Bytes()), nil, 0, 0),
		CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
	}

	suite.App.AccountKeeper.SetAccount(suite.Ctx, acc)

	valAddr := sdk.ValAddress(suite.Address.Bytes())

	suite.Equal("cosmosvaloper1w9tzkuvenpeakkegdhu40tcenmy5v9lhc9rt8k", valAddr.String())

	validator, err := stakingtypes.NewValidator(valAddr, priv.PubKey(), stakingtypes.Description{})
	require.NoError(t, err)
	err = suite.App.StakingKeeper.SetValidatorByConsAddr(suite.Ctx, validator)
	require.NoError(t, err)
	err = suite.App.StakingKeeper.SetValidatorByConsAddr(suite.Ctx, validator)
	require.NoError(t, err)
	suite.App.StakingKeeper.SetValidator(suite.Ctx, validator)

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	suite.ClientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
	suite.EthSigner = ethtypes.LatestSignerForChainID(suite.App.EvmKeeper.ChainID())
	suite.AppCodec = encodingConfig.Codec
	suite.Denom = types.DefaultEVMDenom
}

func (suite *TestSuite) EvmDenom() string {
	ctx := sdk.WrapSDKContext(suite.Ctx)
	rsp, _ := suite.QueryClient.Params(ctx, &types.QueryParamsRequest{})
	return rsp.Params.EvmDenom
}

// Commit and begin new block
func (suite *TestSuite) Commit() abci.ResponseCommit {
	res := suite.App.Commit()
	header := suite.Ctx.BlockHeader()
	header.Height++
	suite.App.BeginBlock(abci.RequestBeginBlock{
		Header: header,
	})

	// update ctx
	suite.Ctx = suite.App.BaseApp.NewContext(false, header)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.Ctx, suite.App.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.App.EvmKeeper)
	suite.QueryClient = types.NewQueryClient(queryHelper)

	return res
}

func (suite *TestSuite) StateDB() *statedb.StateDB {
	return statedb.New(
		suite.Ctx,
		suite.App.EvmKeeper,
		statedb.NewEmptyTxConfig(common.BytesToHash(suite.Ctx.HeaderHash().Bytes())),
	)
}

// DeployTestContract deploy a test erc20 contract and returns the contract address
func (suite *TestSuite) DeployTestContract(t require.TestingT, owner common.Address, supply *big.Int) common.Address {
	ctx := sdk.WrapSDKContext(suite.Ctx)
	chainID := suite.App.EvmKeeper.ChainID()

	ctorArgs, err := types.ERC20Contract.ABI.Pack("", owner, supply)
	require.NoError(t, err)

	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)

	data := types.ERC20Contract.Bin
	data = append(data, ctorArgs...)
	args, err := json.Marshal(&types.TransactionArgs{
		From: &suite.Address,
		Data: (*hexutil.Bytes)(&data),
	})
	require.NoError(t, err)
	res, err := suite.QueryClient.EstimateGas(ctx, &types.EthCallRequest{
		Args:            args,
		GasCap:          config.DefaultGasCap,
		ProposerAddress: suite.Ctx.BlockHeader().ProposerAddress,
	})
	require.NoError(t, err)

	var erc20DeployTx *types.MsgEthereumTx
	if suite.EnableFeemarket {
		erc20DeployTx = types.NewTxContract(
			chainID,
			nonce,
			nil,     // amount
			res.Gas, // gasLimit
			nil,     // gasPrice
			suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx),
			big.NewInt(1),
			data,                   // input
			&ethtypes.AccessList{}, // accesses
		)
	} else {
		erc20DeployTx = types.NewTxContract(
			chainID,
			nonce,
			nil,     // amount
			res.Gas, // gasLimit
			nil,     // gasPrice
			nil, nil,
			data, // input
			nil,  // accesses
		)
	}

	erc20DeployTx.From = suite.Address.Hex()
	err = erc20DeployTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.Signer)
	require.NoError(t, err)
	rsp, err := suite.App.EvmKeeper.EthereumTx(ctx, erc20DeployTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return crypto.CreateAddress(suite.Address, nonce)
}

func (suite *TestSuite) MustTransferERC20Token(t require.TestingT, contractAddr, from, to common.Address, amount *big.Int) *types.MsgEthereumTx {
	ercTransferTx, rsp, err := suite.TransferERC20Token(contractAddr, from, to, amount)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return ercTransferTx
}

func (suite *TestSuite) TransferERC20Token(
	contractAddr, from, to common.Address,
	amount *big.Int,
) (*types.MsgEthereumTx, *types.MsgEthereumTxResponse, error) {
	ctx := sdk.WrapSDKContext(suite.Ctx)
	chainID := suite.App.EvmKeeper.ChainID()

	transferData, err := types.ERC20Contract.ABI.Pack("transfer", to, amount)
	if err != nil {
		return nil, nil, err
	}
	args, err := json.Marshal(&types.TransactionArgs{To: &contractAddr, From: &from, Data: (*hexutil.Bytes)(&transferData)})
	if err != nil {
		return nil, nil, err
	}
	res, err := suite.QueryClient.EstimateGas(ctx, &types.EthCallRequest{
		Args:            args,
		GasCap:          25_000_000,
		ProposerAddress: suite.Ctx.BlockHeader().ProposerAddress,
	})
	if err != nil {
		return nil, nil, err
	}

	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)

	var ercTransferTx *types.MsgEthereumTx
	if suite.EnableFeemarket {
		ercTransferTx = types.NewTx(
			chainID,
			nonce,
			&contractAddr,
			nil,
			res.Gas,
			nil,
			suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx),
			big.NewInt(1),
			transferData,
			&ethtypes.AccessList{}, // accesses
		)
	} else {
		ercTransferTx = types.NewTx(
			chainID,
			nonce,
			&contractAddr,
			nil,
			res.Gas,
			nil,
			nil, nil,
			transferData,
			nil,
		)
	}

	ercTransferTx.From = suite.Address.Hex()
	err = ercTransferTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.Signer)
	if err != nil {
		return nil, nil, err
	}
	rsp, err := suite.App.EvmKeeper.EthereumTx(ctx, ercTransferTx)
	if err != nil {
		return nil, rsp, err
	}

	return ercTransferTx, rsp, nil
}

// DeployTestMessageCall deploy a test erc20 contract and returns the contract address
func (suite *TestSuite) DeployTestMessageCall(t require.TestingT) common.Address {
	ctx := sdk.WrapSDKContext(suite.Ctx)
	chainID := suite.App.EvmKeeper.ChainID()

	data := types.TestMessageCall.Bin
	args, err := json.Marshal(&types.TransactionArgs{
		From: &suite.Address,
		Data: (*hexutil.Bytes)(&data),
	})
	require.NoError(t, err)

	res, err := suite.QueryClient.EstimateGas(ctx, &types.EthCallRequest{
		Args:            args,
		GasCap:          config.DefaultGasCap,
		ProposerAddress: suite.Ctx.BlockHeader().ProposerAddress,
	})
	require.NoError(t, err)

	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)

	var erc20DeployTx *types.MsgEthereumTx
	if suite.EnableFeemarket {
		erc20DeployTx = types.NewTxContract(
			chainID,
			nonce,
			nil,     // amount
			res.Gas, // gasLimit
			nil,     // gasPrice
			suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx),
			big.NewInt(1),
			data,                   // input
			&ethtypes.AccessList{}, // accesses
		)
	} else {
		erc20DeployTx = types.NewTxContract(
			chainID,
			nonce,
			nil,     // amount
			res.Gas, // gasLimit
			nil,     // gasPrice
			nil, nil,
			data, // input
			nil,  // accesses
		)
	}

	erc20DeployTx.From = suite.Address.Hex()
	err = erc20DeployTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.Signer)
	require.NoError(t, err)
	rsp, err := suite.App.EvmKeeper.EthereumTx(ctx, erc20DeployTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return crypto.CreateAddress(suite.Address, nonce)
}

func (suite *TestSuite) GetAllAccountStorage(
	ctx sdk.Context,
	addr common.Address,
) map[common.Hash]common.Hash {
	states := make(map[common.Hash]common.Hash)
	suite.App.EvmKeeper.ForEachStorage(ctx, addr, func(key, value common.Hash) bool {
		states[key] = value
		// Iterate all
		return true
	})

	return states
}

func (suite *TestSuite) MintCoinsForAccount(
	ctx sdk.Context,
	addr sdk.AccAddress,
	amount sdk.Coins,
) {
	err := suite.App.BankKeeper.MintCoins(ctx, minttypes.ModuleName, amount)
	suite.Require().NoError(err)

	err = suite.App.BankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		addr,
		amount,
	)
	suite.Require().NoError(err)
}
