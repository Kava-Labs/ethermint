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
	"bytes"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	precompile_modules "github.com/ethereum/go-ethereum/precompile/modules"

	"github.com/evmos/ethermint/types"
)

var (
	// DefaultEVMDenom defines the default EVM denomination on Ethermint
	DefaultEVMDenom = types.AttoPhoton
	// DefaultAllowUnprotectedTxs rejects all unprotected txs (i.e false)
	DefaultAllowUnprotectedTxs = false
	// DefaultEnableCreate enables contract creation (i.e true)
	DefaultEnableCreate = true
	// DefaultEnableCall enables contract calls (i.e true)
	DefaultEnableCall = true
)

// AvailableExtraEIPs define the list of all EIPs that can be enabled by the
// EVM interpreter. These EIPs are applied in order and can override the
// instruction sets from the latest hard fork enabled by the ChainConfig. For
// more info check:
// https://github.com/ethereum/go-ethereum/blob/master/core/vm/interpreter.go#L97
var AvailableExtraEIPs = []int64{1344, 1884, 2200, 2929, 3198, 3529}

// NewParams creates a new Params instance
func NewParams(
	evmDenom string,
	allowUnprotectedTxs,
	enableCreate,
	enableCall bool,
	config ChainConfig,
	extraEIPs []int64,
	eip712AllowedMsgs []EIP712AllowedMsg,
	enabledPrecompiles []string,
) Params {
	return Params{
		EvmDenom:            evmDenom,
		AllowUnprotectedTxs: allowUnprotectedTxs,
		EnableCreate:        enableCreate,
		EnableCall:          enableCall,
		ExtraEIPs:           extraEIPs,
		ChainConfig:         config,
		EIP712AllowedMsgs:   eip712AllowedMsgs,
		EnabledPrecompiles:  enabledPrecompiles,
	}
}

// DefaultParams returns default evm parameters
// ExtraEIPs is empty to prevent overriding the latest hard fork instruction set
func DefaultParams() Params {
	return Params{
		EvmDenom:            DefaultEVMDenom,
		EnableCreate:        DefaultEnableCreate,
		EnableCall:          DefaultEnableCall,
		ChainConfig:         DefaultChainConfig(),
		ExtraEIPs:           nil,
		AllowUnprotectedTxs: DefaultAllowUnprotectedTxs,
		EIP712AllowedMsgs:   nil,
		EnabledPrecompiles:  nil,
	}
}

// Validate performs basic validation on evm parameters.
func (p Params) Validate() error {
	if err := sdk.ValidateDenom(p.EvmDenom); err != nil {
		return err
	}

	for _, eip := range p.ExtraEIPs {
		if !vm.ValidEip(int(eip)) {
			return fmt.Errorf("EIP %d is not activateable, valid EIPS are: %s", eip, vm.ActivateableEips())
		}
	}

	if err := p.ChainConfig.Validate(); err != nil {
		return err
	}

	if err := checkEIP712AllowedMsgsForDuplicates(p.EIP712AllowedMsgs); err != nil {
		return err
	}

	if err := validateEnabledPrecompiles(p.EnabledPrecompiles); err != nil {
		return err
	}

	return nil
}

// EIP712AllowedMsgFromMsgType returns the EIP712AllowedMsg for a given message type url.
func (p Params) EIP712AllowedMsgFromMsgType(msgTypeURL string) *EIP712AllowedMsg {
	for _, allowedMsg := range p.EIP712AllowedMsgs {
		if allowedMsg.MsgTypeUrl == msgTypeURL {
			return &allowedMsg
		}
	}
	return nil
}

// EIPs returns the ExtraEIPS as a int slice
func (p Params) EIPs() []int {
	eips := make([]int, len(p.ExtraEIPs))
	for i, eip := range p.ExtraEIPs {
		eips[i] = int(eip)
	}
	return eips
}

func checkEIP712AllowedMsgsForDuplicates(msgs []EIP712AllowedMsg) error {
	seenMsgTypes := make(map[string]struct{})

	for _, msg := range msgs {
		if _, ok := seenMsgTypes[msg.MsgTypeUrl]; ok {
			return fmt.Errorf("duplicate eip712 allowed legacy msg type: %s", msg.MsgTypeUrl)
		}
		seenMsgTypes[msg.MsgTypeUrl] = struct{}{}
	}

	return nil
}

// validateEnabledPrecompiles asserts that the enabled precompiles are valid
// hex addresses, sorted in byte format ascending, and unique in byte format
func validateEnabledPrecompiles(enabledPrecompiles []string) error {
	addrs := make([]common.Address, len(enabledPrecompiles))

	for index, hexAddr := range enabledPrecompiles {
		if !common.IsHexAddress(hexAddr) {
			return fmt.Errorf("invalid hex address: %v in enabled precompiles list", hexAddr)
		}
		addrs[index] = common.HexToAddress(hexAddr)
	}

	for i := 0; i < len(addrs)-1; i++ {
		cmp := bytes.Compare(addrs[i].Bytes(), addrs[i+1].Bytes())

		// addrs[i] > addrs[i+1], not ascending order
		if cmp == 1 {
			return fmt.Errorf("enabled precompiles are not sorted, %v > %v", addrs[i].Hex(), addrs[i+1].Hex())
		}

		// addrs[i] == addrs[i+1]
		if cmp == 0 {
			return fmt.Errorf("enabled precompiles are not unique, %v is duplicated", addrs[i].Hex())
		}
	}

	return nil
}

// IsLondon returns if london hardfork is enabled.
// TODO(nddeluca): does this belong in params?
func IsLondon(ethConfig *params.ChainConfig, height int64) bool {
	return ethConfig.IsLondon(big.NewInt(height))
}

// ValidatePrecompileRegistration checks that all enabled precompiles are registered.
// TODO(nddeluca): does this belong in params?
func ValidatePrecompileRegistration(registeredModules []precompile_modules.Module, enabledPrecompiles []string) error {
	registeredAddrs := make(map[string]struct{}, len(registeredModules))

	for _, module := range registeredModules {
		registeredAddrs[module.Address.String()] = struct{}{}
	}

	for _, enabledPrecompile := range enabledPrecompiles {
		if _, ok := registeredAddrs[enabledPrecompile]; !ok {
			return fmt.Errorf("precompile %v is enabled but not registered", enabledPrecompile)
		}
	}

	return nil
}
