package statedb

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var (
	AccessListAddressKey     = []byte{0x01} // common.Address
	AccessListAddressSlotKey = []byte{0x02} // (common.Address, common.Hash)

	LogKey      = []byte{0x03}
	RefundKey   = []byte{0x04}
	SuicidedKey = []byte{0x05}
)

type StateDBStore struct {
	key storetypes.StoreKey
}

func NewStateDBStore(storeKey storetypes.StoreKey) *StateDBStore {
	return &StateDBStore{
		key: storeKey,
	}
}

// AddLog adds a log to the store.
func (ls *StateDBStore) AddLog(ctx sdk.Context, log *ethtypes.Log) {
	store := prefix.NewStore(ctx.KVStore(ls.key), LogKey)

	bz, err := log.MarshalJSON()
	if err != nil {
		panic(err)
	}
	store.Set(log.Address.Bytes(), bz)
}

func (ls *StateDBStore) GetAllLogs(ctx sdk.Context) []*ethtypes.Log {
	store := prefix.NewStore(ctx.KVStore(ls.key), LogKey)

	logs := make([]*ethtypes.Log, 0)

	iter := sdk.KVStorePrefixIterator(store, []byte{})
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var log ethtypes.Log
		err := log.UnmarshalJSON(iter.Value())
		if err != nil {
			panic(err)
		}
		logs = append(logs, &log)
	}

	return logs
}

// AddRefund adds a refund to the store.
func (ls *StateDBStore) AddRefund(ctx sdk.Context, gas uint64) {
	store := prefix.NewStore(ctx.KVStore(ls.key), RefundKey)
	// Add to existing refund
	bz := store.Get([]byte{})
	if bz != nil {
		gas += sdk.BigEndianToUint64(bz)
	}

	store.Set([]byte{}, sdk.Uint64ToBigEndian(gas))
}

func (ls *StateDBStore) SubRefund(ctx sdk.Context, gas uint64) {
	store := prefix.NewStore(ctx.KVStore(ls.key), RefundKey)
	// Subtract from existing refund
	bz := store.Get([]byte{})
	if bz == nil {
		panic("no refund to subtract from")
	}

	refund := sdk.BigEndianToUint64(bz)

	if refund < gas {
		panic("refund is less than gas")
	}

	gas = refund - gas
	store.Set([]byte{}, sdk.Uint64ToBigEndian(gas))
}

func (ls *StateDBStore) GetRefund(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(ls.key), RefundKey)
	bz := store.Get([]byte{})
	if bz == nil {
		return 0
	}
	return sdk.BigEndianToUint64(bz)
}

func (ls *StateDBStore) SetAccountSuicided(ctx sdk.Context, addr common.Address) {
	store := prefix.NewStore(ctx.KVStore(ls.key), SuicidedKey)
	store.Set(addr.Bytes(), []byte{1})
}

func (ls *StateDBStore) GetAccountSuicided(ctx sdk.Context, addr common.Address) bool {
	store := prefix.NewStore(ctx.KVStore(ls.key), SuicidedKey)
	return store.Has(addr.Bytes())
}

func (ls *StateDBStore) GetAllSuicided(ctx sdk.Context) []common.Address {
	store := prefix.NewStore(ctx.KVStore(ls.key), SuicidedKey)

	addrs := make([]common.Address, 0)

	iter := sdk.KVStorePrefixIterator(store, []byte{})
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		addr := common.BytesToAddress(iter.Key())
		addrs = append(addrs, addr)
	}

	return addrs
}
