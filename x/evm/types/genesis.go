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
package types

import (
	"fmt"

	ethermint "github.com/evmos/ethermint/types"
)

// NewGenesisState creates a new genesis state with EVM configuration parameters
// and initial contract accounts.
func NewGenesisState(params Params, accounts []GenesisAccount) *GenesisState {
	return &GenesisState{
		Accounts: accounts,
		Params:   params,
	}
}

// DefaultGenesisState sets default evm genesis state with default parameters and
// no initial genesis accounts.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Accounts: []GenesisAccount{},
		Params:   DefaultParams(),
	}
}

// Validate performs a basic validation of a GenesisAccount fields.
// A valid genesis account has a valid address, non-empty code, and
// valid storage.
func (ga GenesisAccount) Validate() error {
	if err := ethermint.ValidateAddress(ga.Address); err != nil {
		return err
	}

	if ga.Code == "" {
		return fmt.Errorf("code can not be empty")
	}

	return ga.Storage.Validate()
}

// Validate performs basic genesis state validation returning an error upon any
// failure. It ensures the params are valid and every genesis account is unique
// and valid.
func (gs GenesisState) Validate() error {
	seenAccounts := make(map[string]struct{})

	for _, acc := range gs.Accounts {
		if err := acc.Validate(); err != nil {
			return fmt.Errorf("invalid genesis account %s: %w", acc.Address, err)
		}

		if _, ok := seenAccounts[acc.Address]; ok {
			return fmt.Errorf("duplicated genesis account %s", acc.Address)
		}

		seenAccounts[acc.Address] = struct{}{}
	}

	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	for _, ep := range gs.Params.EnabledPrecompiles {
		if _, ok := seenAccounts[ep]; !ok {
			return fmt.Errorf("enabled precompile %s must have a matching genesis account", ep)
		}
	}

	return nil
}
