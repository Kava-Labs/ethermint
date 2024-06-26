package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/core/vm"
	ethtypes "github.com/evmos/ethermint/types"
	"github.com/evmos/ethermint/x/evm/types"
)

var _ paramtypes.ParamSet = &LegacyParams{}

const (
	DefaultEVMDenom = ethtypes.AttoPhoton
)

// Parameter keys
var (
	ParamStoreKeyEVMDenom          = []byte("EVMDenom")
	ParamStoreKeyEnableCreate      = []byte("EnableCreate")
	ParamStoreKeyEnableCall        = []byte("EnableCall")
	ParamStoreKeyExtraEIPs         = []byte("EnableExtraEIPs")
	ParamStoreKeyChainConfig       = []byte("ChainConfig")
	ParamStoreKeyEIP712AllowedMsgs = []byte("EIP712AllowedMsgs")

	// AvailableExtraEIPs define the list of all EIPs that can be enabled by the
	// EVM interpreter. These EIPs are applied in order and can override the
	// instruction sets from the latest hard fork enabled by the ChainConfig. For
	// more info check:
	// https://github.com/ethereum/go-ethereum/blob/master/core/vm/interpreter.go#L97
	AvailableExtraEIPs = []int64{1344, 1884, 2200, 2929, 3198, 3529}
)

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&LegacyParams{})
}

// NewParams creates a new Params instance
func NewParams(evmDenom string, enableCreate, enableCall bool, config LegacyChainConfig, extraEIPs ...int64) LegacyParams {
	return LegacyParams{
		EvmDenom:          evmDenom,
		EnableCreate:      enableCreate,
		EnableCall:        enableCall,
		ExtraEIPs:         extraEIPs,
		ChainConfig:       config,
		EIP712AllowedMsgs: []types.EIP712AllowedMsg{},
	}
}

// DefaultParams returns default evm parameters
// ExtraEIPs is empty to prevent overriding the latest hard fork instruction set
func DefaultParams() LegacyParams {
	return LegacyParams{
		EvmDenom:          DefaultEVMDenom,
		EnableCreate:      true,
		EnableCall:        true,
		ChainConfig:       DefaultChainConfig(),
		ExtraEIPs:         nil,
		EIP712AllowedMsgs: []types.EIP712AllowedMsg{},
	}
}

// ParamSetPairs returns the parameter set pairs.
func (p *LegacyParams) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyEVMDenom, &p.EvmDenom, validateEVMDenom),
		paramtypes.NewParamSetPair(ParamStoreKeyEnableCreate, &p.EnableCreate, validateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyEnableCall, &p.EnableCall, validateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyExtraEIPs, &p.ExtraEIPs, validateEIPs),
		paramtypes.NewParamSetPair(ParamStoreKeyChainConfig, &p.ChainConfig, validateChainConfig),
		paramtypes.NewParamSetPair(ParamStoreKeyEIP712AllowedMsgs, &p.EIP712AllowedMsgs, validateEIP712AllowedMsgs),
	}
}

// Validate performs basic validation on evm parameters.
func (p LegacyParams) Validate() error {
	if err := sdk.ValidateDenom(p.EvmDenom); err != nil {
		return err
	}

	if err := validateEIPs(p.ExtraEIPs); err != nil {
		return err
	}

	if err := p.ChainConfig.Validate(); err != nil {
		return err
	}

	return validateEIP712AllowedMsgs(p.EIP712AllowedMsgs)
}

// EIP712AllowedMsgFromMsgType returns the EIP712AllowedMsg for a given message type url.
func (p LegacyParams) EIP712AllowedMsgFromMsgType(msgTypeURL string) *types.EIP712AllowedMsg {
	for _, allowedMsg := range p.EIP712AllowedMsgs {
		if allowedMsg.MsgTypeUrl == msgTypeURL {
			return &allowedMsg
		}
	}
	return nil
}

// EIPs returns the ExtraEips as a int slice
func (p LegacyParams) EIPs() []int {
	eips := make([]int, len(p.ExtraEIPs))
	for i, eip := range p.ExtraEIPs {
		eips[i] = int(eip)
	}
	return eips
}

func validateEVMDenom(i interface{}) error {
	denom, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter EVM denom type: %T", i)
	}

	return sdk.ValidateDenom(denom)
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateEIPs(i interface{}) error {
	eips, ok := i.([]int64)
	if !ok {
		return fmt.Errorf("invalid EIP slice type: %T", i)
	}

	for _, eip := range eips {
		if !vm.ValidEip(int(eip)) {
			return fmt.Errorf("EIP %d is not activateable, valid EIPS are: %s", eip, vm.ActivateableEips())
		}
	}

	return nil
}

func validateChainConfig(i interface{}) error {
	cfg, ok := i.(LegacyChainConfig)
	if !ok {
		return fmt.Errorf("invalid chain config type: %T", i)
	}

	return cfg.Validate()
}

func validateEIP712AllowedMsgs(i interface{}) error {
	allowedMsgs, ok := i.([]types.EIP712AllowedMsg)
	if !ok {
		return fmt.Errorf("invalid EIP712AllowedMsg slice type: %T", i)
	}

	// ensure no duplicate msg type urls
	msgTypes := make(map[string]bool)
	for _, allowedMsg := range allowedMsgs {
		if _, ok := msgTypes[allowedMsg.MsgTypeUrl]; ok {
			return fmt.Errorf("duplicate eip712 allowed legacy msg type: %s", allowedMsg.MsgTypeUrl)
		}
		msgTypes[allowedMsg.MsgTypeUrl] = true
	}

	return nil
}
