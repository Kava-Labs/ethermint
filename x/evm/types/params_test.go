package types_test

import (
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	precompile_modules "github.com/ethereum/go-ethereum/precompile/modules"
	"github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// basic validations
	validPrecompileAddress   = "0xc0ffee254729296a45a3885639AC7E10F9d54979"
	invalidPrecompileAddress = "0xc0ffee254729296a45a3885639AC7E10F9d5497"

	// sort order validations
	precompileSort1 = "0x1000000000000000000000000000000000000000"
	precompileSort2 = "0x2000000000000000000000000000000000000000"
	precompileSort3 = "0xaA00000000000000000000000000000000000000"
	precompileSort4 = "0xAB00000000000000000000000000000000000000"

	// uniqueness validations
	precompileSort1Dup = "0x1000000000000000000000000000000000000000"
	precompileSort2Dup = "0x2000000000000000000000000000000000000000"
	precompileSort3Dup = "0xAa00000000000000000000000000000000000000"
	precompileSort4Dup = "0xab00000000000000000000000000000000000000"
)

var testExtraEips = []int64{2929, 1884, 1344}

type paramTestCase struct {
	name        string
	getParams   func() types.Params
	expectedErr string
}

var paramTestCases = []paramTestCase{
	{
		name:        "default params are valid",
		getParams:   types.DefaultParams,
		expectedErr: "",
	},
	{
		name: "valid construction",
		getParams: func() types.Params {
			return types.NewParams("kava", false, true, true, types.DefaultChainConfig(), testExtraEips, []types.EIP712AllowedMsg{}, []string{})
		},
		expectedErr: "",
	},
	{
		name: "empty evm denom",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EvmDenom = ""
			return params
		},
		expectedErr: "invalid denom: ",
	},
	{
		name: "invalid evm denom with spaces",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EvmDenom = "INVALID DENOM"
			return params
		},
		expectedErr: "invalid denom: INVALID DENOM",
	},
	{
		name: "invalid evm denom with disallowed characters",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EvmDenom = "@!#!@$!@5^32"
			return params
		},
		expectedErr: "invalid denom: @!#!@$!@5^32",
	},
	{
		name: "invalid EIP - is not activateable",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.ExtraEIPs = []int64{1}
			return params
		},
		expectedErr: "EIP 1 is not activateable, valid EIPS are:",
	},
	{
		name: "invalid chain config with negative block value",
		getParams: func() types.Params {
			params := types.DefaultParams()

			invalidBlock := sdkmath.NewInt(-1)
			params.ChainConfig.HomesteadBlock = &invalidBlock

			return params
		},
		expectedErr: "homesteadBlock: block value cannot be negative: -1",
	},
	{
		name: "invalid eip allowed msgs with duplicated msg url",
		getParams: func() types.Params {
			params := types.DefaultParams()

			msg1 := types.EIP712AllowedMsg{
				MsgTypeUrl:       "/cosmos.bank.v1beta1.MsgSend",
				MsgValueTypeName: "MsgValueSend",
				ValueTypes: []types.EIP712MsgAttrType{
					{Name: "from_address", Type: "string"},
					{Name: "to_address", Type: "string"},
					{Name: "amount", Type: "Coin[]"},
				},
			}

			params.EIP712AllowedMsgs = []types.EIP712AllowedMsg{msg1, msg1}

			return params
		},
		expectedErr: "duplicate eip712 allowed legacy msg type: /cosmos.bank.v1beta1.MsgSend",
	},
	{
		name: "empty enabled precompile address",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{""}
			return params
		},
		expectedErr: "invalid hex address",
	},
	{
		name: "enabled precompiles only element invalid hex",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{invalidPrecompileAddress}
			return params
		},
		expectedErr: "invalid hex address",
	},
	{
		name: "enabled precompiles invalid first element hex",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{invalidPrecompileAddress, validPrecompileAddress}
			return params
		},
		expectedErr: "invalid hex address",
	},
	{
		name: "enabled precompiles invalid last element hex",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{validPrecompileAddress, invalidPrecompileAddress}
			return params
		},
		expectedErr: "invalid hex address",
	},
	{
		name: "enabled precompiles can be nil",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = nil
			return params
		},
		expectedErr: "",
	},
	{
		name: "enabled precompiles can be an empty slice",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{}
			return params
		},
		expectedErr: "",
	},
	{
		name: "enabled precompiles are valid with valid hex address",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{validPrecompileAddress}
			return params
		},
		expectedErr: "",
	},
	{
		name: "valid enabled precompile ascending byte order",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort1, precompileSort2, precompileSort3, precompileSort4}
			return params
		},
		expectedErr: "",
	},
	{
		name: "invalid enabled precompile descending byte order",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort4, precompileSort3, precompileSort2, precompileSort1}
			return params
		},
		expectedErr: "enabled precompiles are not sorted",
	},
	{
		name: "invalid enabled precompile first element not sorted",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort2, precompileSort1, precompileSort3, precompileSort4}
			return params
		},
		expectedErr: "enabled precompiles are not sorted",
	},
	{
		name: "invalid enabled precompile last element not sorted",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort1, precompileSort2, precompileSort4, precompileSort3}
			return params
		},
		expectedErr: "enabled precompiles are not sorted",
	},
	{
		name: "invalid enabled precompile middle element not sorted",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort1, precompileSort3, precompileSort2, precompileSort4}
			return params
		},
		expectedErr: "enabled precompiles are not sorted",
	},
	{
		name: "invalid enabled precompiles second element not unique",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort1, precompileSort1Dup, precompileSort2, precompileSort3, precompileSort4}
			return params
		},
		expectedErr: "enabled precompiles are not unique",
	},
	{
		name: "invalid enabled precompiles last element not byte unique",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort1, precompileSort2, precompileSort2, precompileSort3, precompileSort4, precompileSort4Dup}
			return params
		},
		expectedErr: "enabled precompiles are not unique",
	},
	{
		name: "invalid enabled precompiles middle element not byte unique",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort1, precompileSort2, precompileSort3, precompileSort3Dup, precompileSort4}
			return params
		},
		expectedErr: "enabled precompiles are not unique",
	},
	{
		name: "invalid enabled precompiles are not sorted and not unique",
		getParams: func() types.Params {
			params := types.DefaultParams()
			params.EnabledPrecompiles = []string{precompileSort2, precompileSort1, precompileSort2Dup}
			return params
		},
		// we prioritize sort order, then check uniqueness
		expectedErr: "enabled precompiles are not sorted",
	},
}

func TestParamsValidate(t *testing.T) {
	for _, tc := range paramTestCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.getParams().Validate()

			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}

func TestParamsEIPs(t *testing.T) {
	extraEips := []int64{2929, 1884, 1344}

	params := types.DefaultParams()
	params.ExtraEIPs = extraEips

	actual := params.EIPs()
	require.Equal(t, []int{2929, 1884, 1344}, actual)
}

func TestIsLondon(t *testing.T) {
	config := params.MainnetChainConfig

	require.True(t, config.LondonBlock.IsInt64())
	londonHeight := config.LondonBlock.Int64()

	testCases := []struct {
		name     string
		height   int64
		isLondon bool
	}{
		{
			name:     "before london block",
			height:   londonHeight - 1,
			isLondon: false,
		},
		{
			name:     "at london block",
			height:   londonHeight,
			isLondon: true,
		},
		{
			name:     "after london block",
			height:   londonHeight + 1,
			isLondon: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.isLondon {
				assert.True(t, types.IsLondon(config, tc.height), "expected height to be on or after london fork")
			} else {
				assert.False(t, types.IsLondon(config, tc.height), "expected height to be before london fork")
			}
		})
	}
}

func TestValidatePrecompileRegistration(t *testing.T) {
	m := func(addr string) precompile_modules.Module {
		return precompile_modules.Module{
			Address: common.HexToAddress(addr),
		}
	}
	a := func(addr string) string {
		return common.HexToAddress(addr).String()
	}

	testCases := []struct {
		name               string
		registeredModules  []precompile_modules.Module
		enabledPrecompiles []string
		errorMsg           string
	}{
		{
			name:               "success: all enabled precompiles are registered #1",
			registeredModules:  []precompile_modules.Module{m("0x1"), m("0x2"), m("0x3")},
			enabledPrecompiles: []string{a("0x1"), a("0x2"), a("0x3")},
			errorMsg:           "",
		},
		{
			name:               "success: all enabled precompiles are registered #2",
			registeredModules:  []precompile_modules.Module{m("0x1"), m("0x2"), m("0x3")},
			enabledPrecompiles: []string{a("0x1"), a("0x3")},
			errorMsg:           "",
		},
		{
			name:               "success: no enabled precompiles",
			registeredModules:  []precompile_modules.Module{m("0x1"), m("0x2"), m("0x3")},
			enabledPrecompiles: []string{},
			errorMsg:           "",
		},
		{
			name:               "success: no enabled precompiles and no registered modules",
			registeredModules:  []precompile_modules.Module{},
			enabledPrecompiles: []string{},
			errorMsg:           "",
		},
		{
			name:               "failure: precompile is enabled, but not registered",
			registeredModules:  []precompile_modules.Module{m("0x1"), m("0x2"), m("0x3")},
			enabledPrecompiles: []string{a("0x4")},
			errorMsg:           fmt.Sprintf("precompile %v is enabled but not registered", a("0x4")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidatePrecompileRegistration(tc.registeredModules, tc.enabledPrecompiles)

			if tc.errorMsg != "" {
				require.Error(t, err, tc.name)
				require.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err, tc.name)
			}
		})
	}
}
