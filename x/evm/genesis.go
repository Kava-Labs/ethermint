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
package evm

import (
	"bytes"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	precompile_modules "github.com/ethereum/go-ethereum/precompile/modules"

	ethermint "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm/keeper"
	"github.com/evmos/ethermint/x/evm/types"
)

// InitGenesis initializes genesis state based on the provided genesis state which
// is either for a new chain, or from an exported existing chain.
//
// The context is used to provide context to keeper function calls and to load
// the EVM Chain ID, which is required by EIP-155.
//
// The EVM Keeper is used to normalize and store the provided data or genesis state.
//
// The Account Keeper is used to check corresponding accounts exist for the module
// and EVM genesis accounts.
//
// The registered precompiles list is used to ensure that any param enabled precompiles
// exist and are included in the binary.
//
// Since the data provided is assumed to have already passed basic validations,
// we only directly check stateful validations and assumptions of external state
// from the Account Keeper and registered precompile list.
func InitGenesis(
	ctx sdk.Context,
	k *keeper.Keeper,
	accountKeeper types.AccountKeeper,
	data types.GenesisState,
	registeredModules []precompile_modules.Module,
) []abci.ValidatorUpdate {
	k.WithChainID(ctx)

	// For an enabled precompile to be valid,
	// it must exist in the binary and be registered
	err := types.ValidatePrecompileRegistration(
		registeredModules,
		data.Params.GetEnabledPrecompiles(),
	)
	if err != nil {
		panic(err)
	}

	err = k.SetParams(ctx, data.Params)
	if err != nil {
		panic(fmt.Errorf("error setting params %s", err))
	}

	// ensure evm module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the EVM module account has not been set")
	}

	isEnabledPrecompile := make(map[string]struct{})
	for _, ep := range data.Params.EnabledPrecompiles {
		isEnabledPrecompile[ep] = struct{}{}
	}

	for _, account := range data.Accounts {
		address := common.HexToAddress(account.Address)
		accAddress := sdk.AccAddress(address.Bytes())

		// check that the EVM balance the matches the account balance
		acc := accountKeeper.GetAccount(ctx, accAddress)
		if acc == nil {
			panic(fmt.Errorf("account not found for address %s", account.Address))
		}

		ethAcct, ok := acc.(ethermint.EthAccountI)
		if !ok {
			panic(
				fmt.Errorf("account %s must be an EthAccount interface, got %T",
					account.Address, acc,
				),
			)
		}

		if ethAcct.GetSequence() == 0 {
			panic(fmt.Errorf("account %s must have a positive nonce", account.Address))
		}

		if ethAcct.GetPubKey() != nil {
			panic(fmt.Errorf("account %s must not have a public key set", account.Address))
		}

		code := common.Hex2Bytes(account.Code)
		codeHash := crypto.Keccak256Hash(code)

		if _, ok := isEnabledPrecompile[account.Address]; ok && !bytes.Equal(code, []byte{0x01}) {
			panic(fmt.Errorf("enabled precompile %s must have code set to 0x01, got 0x%s", account.Address, account.Code))
		}

		if !bytes.Equal(ethAcct.GetCodeHash().Bytes(), codeHash.Bytes()) {
			s := "the evm state code doesn't match with the codehash\n"
			panic(fmt.Sprintf("%s account: %s , evm state codehash: %v, ethAccount codehash: %v, evm state code: %s\n",
				s, account.Address, codeHash, ethAcct.GetCodeHash(), account.Code))
		}

		k.SetCode(ctx, codeHash.Bytes(), code)

		for _, storage := range account.Storage {
			k.SetState(ctx, address, common.HexToHash(storage.Key), common.HexToHash(storage.Value).Bytes())
		}
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports genesis state of the EVM module
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper, ak types.AccountKeeper) *types.GenesisState {
	var ethGenAccounts []types.GenesisAccount
	ak.IterateAccounts(ctx, func(account sdk.AccountI) bool {
		ethAccount, ok := account.(ethermint.EthAccountI)
		if !ok {
			// ignore non EthAccounts
			return false
		}

		addr := ethAccount.EthAddress()

		storage := k.GetAccountStorage(ctx, addr)

		genAccount := types.GenesisAccount{
			Address: addr.String(),
			Code:    common.Bytes2Hex(k.GetCode(ctx, ethAccount.GetCodeHash())),
			Storage: storage,
		}

		ethGenAccounts = append(ethGenAccounts, genAccount)
		return false
	})

	return &types.GenesisState{
		Accounts: ethGenAccounts,
		Params:   k.GetParams(ctx),
	}
}
