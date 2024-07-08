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
package statedb

import (
	"bytes"

	"github.com/ethereum/go-ethereum/crypto"
)

var emptyCodeHash = crypto.Keccak256(nil)

// Account is the Ethereum consensus representation of accounts.
// These objects are stored in the storage of auth module.
type Account struct {
	// Balance is *not* included as it is managed by bank
	Nonce    uint64
	CodeHash []byte
}

// NewEmptyAccount returns an empty account.
func NewEmptyAccount() *Account {
	return &Account{
		CodeHash: emptyCodeHash,
	}
}

// IsContract returns if the account contains contract code.
func (acct Account) IsContract() bool {
	return !bytes.Equal(acct.CodeHash, emptyCodeHash)
}
