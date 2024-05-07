// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. If not, see https://github.com/evmos/ethermint/blob/main/LICENSE
package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"github.com/armon/go-metrics"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/evmos/ethermint/x/evm/types"
)

const PrecompileNonce uint64 = 1

var PrecompileCode = []byte{0x1}

var _ types.MsgServer = &Keeper{}

// EthereumTx implements the gRPC MsgServer interface. It receives a transaction which is then
// executed (i.e applied) against the go-ethereum EVM. The provided SDK Context is set to the Keeper
// so that it can implements and call the StateDB methods without receiving it as a function
// parameter.
func (k *Keeper) EthereumTx(goCtx context.Context, msg *types.MsgEthereumTx) (*types.MsgEthereumTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := msg.From
	tx := msg.AsTransaction()
	txIndex := k.GetTxIndexTransient(ctx)

	labels := []metrics.Label{
		telemetry.NewLabel("tx_type", fmt.Sprintf("%d", tx.Type())),
	}
	if tx.To() == nil {
		labels = append(labels, telemetry.NewLabel("execution", "create"))
	} else {
		labels = append(labels, telemetry.NewLabel("execution", "call"))
	}

	response, err := k.ApplyTransaction(ctx, tx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to apply transaction")
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "ethereum_tx", "total"},
			1,
			labels,
		)

		if response.GasUsed != 0 {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "ethereum_tx", "gas_used", "total"},
				float32(response.GasUsed),
				labels,
			)

			// Observe which users define a gas limit >> gas used. Note, that
			// gas_limit and gas_used are always > 0
			gasLimit := sdk.NewDec(int64(tx.Gas()))
			gasRatio, err := gasLimit.QuoInt64(int64(response.GasUsed)).Float64()
			if err == nil {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "ethereum_tx", "gas_limit", "per", "gas_used"},
					float32(gasRatio),
					labels,
				)
			}
		}
	}()

	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyAmount, tx.Value().String()),
		// add event for ethereum transaction hash format
		sdk.NewAttribute(types.AttributeKeyEthereumTxHash, response.Hash),
		// add event for index of valid ethereum tx
		sdk.NewAttribute(types.AttributeKeyTxIndex, strconv.FormatUint(txIndex, 10)),
		// add event for eth tx gas used, we can't get it from cosmos tx result when it contains multiple eth tx msgs.
		sdk.NewAttribute(types.AttributeKeyTxGasUsed, strconv.FormatUint(response.GasUsed, 10)),
	}

	if len(ctx.TxBytes()) > 0 {
		// add event for tendermint transaction hash format
		hash := tmbytes.HexBytes(tmtypes.Tx(ctx.TxBytes()).Hash())
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyTxHash, hash.String()))
	}

	if to := tx.To(); to != nil {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyRecipient, to.Hex()))
	}

	if response.Failed() {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyEthereumTxFailed, response.VmError))
	}

	txLogAttrs := make([]sdk.Attribute, len(response.Logs))
	for i, log := range response.Logs {
		value, err := json.Marshal(log)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to encode log")
		}
		txLogAttrs[i] = sdk.NewAttribute(types.AttributeKeyTxLog, string(value))
	}

	// emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeEthereumTx,
			attrs...,
		),
		sdk.NewEvent(
			types.EventTypeTxLog,
			txLogAttrs...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, sender),
			sdk.NewAttribute(types.AttributeKeyTxType, fmt.Sprintf("%d", tx.Type())),
		),
	})

	return response, nil
}

// UpdateParams implements the gRPC MsgServer interface. When an UpdateParams
// proposal passes, it updates the module parameters. The update can only be
// performed if the requested authority is the Cosmos SDK governance module
// account.
func (k *Keeper) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority, expected %s, got %s", k.authority.String(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	oldParams := k.GetParams(ctx)
	oldEnabledPrecompiles := HexToAddresses(oldParams.GetEnabledPrecompiles())
	newEnabledPrecompiles := HexToAddresses(req.Params.EnabledPrecompiles)

	err := k.InitializePrecompiles(ctx, newEnabledPrecompiles)
	if err != nil {
		return nil, err
	}

	disabledPrecompiles := SetDifference(oldEnabledPrecompiles, newEnabledPrecompiles)
	err = k.UninitializePrecompiles(ctx, disabledPrecompiles)
	if err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// InitializePrecompiles initializes list of precompiles at specified addresses.
// Initialization of precompile sets non-zero nonce and non-empty code at specified address to resemble behavior of
// regular smart contract.
func (k *Keeper) InitializePrecompiles(ctx sdk.Context, addrs []common.Address) error {
	for _, addr := range addrs {
		// Set the nonce of the precompile's address (as is done when a contract is created) to ensure
		// that it is marked as non-empty and will not be cleaned up when the statedb is finalized.
		codeHash := crypto.Keccak256Hash(PrecompileCode)
		err := k.SetAccount(ctx, addr, statedb.Account{
			Nonce:    PrecompileNonce,
			Balance:  big.NewInt(0),
			CodeHash: codeHash[:],
		})
		if err != nil {
			return err
		}

		// Set the code of the precompile's address to a non-zero length byte slice to ensure that the precompile
		// can be called from within Solidity contracts. Solidity adds a check before invoking a contract to ensure
		// that it does not attempt to invoke a non-existent contract.
		k.SetCode(ctx, codeHash[:], PrecompileCode)
	}

	return nil
}

// UninitializePrecompiles uninitializes list of precompiles at specified addresses.
// Uninitialization of precompile sets zero nonce and empty code at specified address.
func (k *Keeper) UninitializePrecompiles(ctx sdk.Context, addrs []common.Address) error {
	for _, addr := range addrs {
		if err := k.DeleteAccount(ctx, addr); err != nil {
			return err
		}
	}

	return nil
}

// SetDifference returns difference between two sets, example can be:
// a   : {1, 2, 3}
// b   : {1, 3}
// diff: {2}
func SetDifference(a []common.Address, b []common.Address) []common.Address {
	bMap := make(map[common.Address]struct{}, len(b))
	for _, elem := range b {
		bMap[elem] = struct{}{}
	}

	diff := make([]common.Address, 0)
	for _, elem := range a {
		if _, ok := bMap[elem]; !ok {
			diff = append(diff, elem)
		}
	}

	return diff
}

func HexToAddresses(hexAddrs []string) []common.Address {
	addrs := make([]common.Address, len(hexAddrs))
	for i, hexAddr := range hexAddrs {
		addrs[i] = common.HexToAddress(hexAddr)
	}

	return addrs
}

func AddressesToHex(addrs []common.Address) []string {
	hexAddrs := make([]string, len(addrs))
	for i, addr := range addrs {
		hexAddrs[i] = addr.Hex()
	}

	return hexAddrs
}
