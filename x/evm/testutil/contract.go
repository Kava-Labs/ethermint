package testutil

import (
	// embed compiled smart contract
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/server/config"
	"github.com/evmos/ethermint/x/evm/types"
)

var (
	//go:embed StateTest.json
	stateTestContractJSON []byte
	StateTestContract     types.CompiledContract

	//go:embed EIP161Test.json
	EIP161TestContractJSON []byte
	EIP161TestContract     types.CompiledContract
)

func loadContract(jsonBytes []byte) types.CompiledContract {
	var contract types.CompiledContract
	err := json.Unmarshal(jsonBytes, &contract)
	if err != nil {
		panic(err)
	}

	if len(contract.Bin) == 0 {
		panic("load contract failed")
	}

	return contract
}

func init() {
	StateTestContract = loadContract(stateTestContractJSON)
	EIP161TestContract = loadContract(EIP161TestContractJSON)
}

// DeployContract deploys a provided contract and returns the contract address
func (suite *TestSuite) DeployContract(
	contract types.CompiledContract,
	params ...interface{},
) common.Address {
	ctx := sdk.WrapSDKContext(suite.Ctx)
	chainID := suite.App.EvmKeeper.ChainID()

	ctorArgs, err := contract.ABI.Pack("", params...)
	suite.Require().NoError(err)

	nonce := suite.App.EvmKeeper.GetNonce(suite.Ctx, suite.Address)

	data := contract.Bin
	data = append(data, ctorArgs...)
	args, err := json.Marshal(&types.TransactionArgs{
		From: &suite.Address,
		Data: (*hexutil.Bytes)(&data),
	})
	suite.Require().NoError(err)
	res, err := suite.QueryClient.EstimateGas(ctx, &types.EthCallRequest{
		Args:            args,
		GasCap:          config.DefaultGasCap,
		ProposerAddress: suite.Ctx.BlockHeader().ProposerAddress,
	})
	suite.Require().NoError(err)

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
	suite.Require().NoError(err)

	rsp, err := suite.App.EvmKeeper.EthereumTx(ctx, erc20DeployTx)
	suite.Require().NoError(err)
	suite.Require().Empty(rsp.VmError)

	return crypto.CreateAddress(suite.Address, nonce)
}

func (suite *TestSuite) CallContract(
	contract types.CompiledContract,
	contractAddr common.Address,
	value *big.Int,
	method string,
	params ...interface{},
) (*types.MsgEthereumTx, *types.MsgEthereumTxResponse, error) {
	res, err := suite.EstimateCallGas(contract, contractAddr, value, method, params...)
	if err != nil {
		return nil, nil, fmt.Errorf("EstimateGas failed: %w", err)
	}

	suite.T().Logf("estimated gas: %d", res.Gas)

	return suite.CallContractWithGas(contract, contractAddr, value, res.Gas, method, params...)
}

// EstimateCallGas estimates the gas for a contract call
func (suite *TestSuite) EstimateCallGas(
	contract types.CompiledContract,
	contractAddr common.Address,
	value *big.Int,
	method string,
	params ...interface{},
) (*types.EstimateGasResponse, error) {
	ctx := sdk.WrapSDKContext(suite.Ctx)

	transferData, err := contract.ABI.Pack(method, params...)
	if err != nil {
		return nil, err
	}
	args, err := json.Marshal(&types.TransactionArgs{
		To:    &contractAddr,
		From:  &suite.Address,
		Data:  (*hexutil.Bytes)(&transferData),
		Value: (*hexutil.Big)(value),
	})
	if err != nil {
		return nil, err
	}
	return suite.QueryClient.EstimateGas(ctx, &types.EthCallRequest{
		Args:            args,
		GasCap:          25_000_000,
		ProposerAddress: suite.Ctx.BlockHeader().ProposerAddress,
	})
}

func (suite *TestSuite) CallContractWithGas(
	contract types.CompiledContract,
	contractAddr common.Address,
	value *big.Int,
	gas uint64,
	method string,
	params ...interface{},
) (*types.MsgEthereumTx, *types.MsgEthereumTxResponse, error) {
	ctx := sdk.WrapSDKContext(suite.Ctx)
	chainID := suite.App.EvmKeeper.ChainID()

	transferData, err := contract.ABI.Pack(method, params...)
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
			value,
			gas,
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
			value,
			gas,
			nil,
			nil, nil,
			transferData,
			nil,
		)
	}

	ercTransferTx.From = suite.Address.Hex()
	err = ercTransferTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.Signer)
	if err != nil {
		return nil, nil, fmt.Errorf("failed Sign: %w", err)
	}
	rsp, err := suite.App.EvmKeeper.EthereumTx(ctx, ercTransferTx)
	if err != nil {
		return nil, rsp, fmt.Errorf("failed EthereumTx: %w", err)
	}

	return ercTransferTx, rsp, nil
}
