package eip712

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func getTypedDataDomain(chainID uint64) apitypes.TypedDataDomain {
	return apitypes.TypedDataDomain{
		Name:    "Kava Cosmos",
		Version: "1.0.0",
		ChainId: math.NewHexOrDecimal256(int64(chainID)),

		// Fields below are not used signature verification so they are
		// explicitly set empty to exclude them from the hash to be signed.

		// Salt in most cases is not used, other chains sometimes set the
		// chainID as the salt instead of using the chainId field and not
		// together.
		// Discussion on salt usage:
		// https://github.com/OpenZeppelin/openzeppelin-contracts/issues/4318
		Salt: "",

		// VerifyingContract is empty as there is no contract that is verifying
		// the signature. Signature verification is done in the ante handler.
		// Smart contracts that handle EIP712 signatures will include their own
		// address in the domain separator.
		VerifyingContract: "",
	}
}

func getRootTypes() apitypes.Types {
	return apitypes.Types{
		"EIP712Domain": {
			{
				Name: "name",
				Type: "string",
			},
			{
				Name: "version",
				Type: "string",
			},
			{
				Name: "chainId",
				Type: "uint256",
			},
		},
		"Tx": {
			{Name: "account_number", Type: "string"},
			{Name: "chain_id", Type: "string"},
			{Name: "fee", Type: "Fee"},
			{Name: "memo", Type: "string"},
			{Name: "sequence", Type: "string"},
			// Note timeout_height was removed because it was not getting filled with the legacyTx
			// {Name: "timeout_height", Type: "string"},
		},
		"Fee": {
			{Name: "amount", Type: "Coin[]"},
			{Name: "gas", Type: "string"},
		},
		"Coin": {
			{Name: "denom", Type: "string"},
			{Name: "amount", Type: "string"},
		},
	}
}
