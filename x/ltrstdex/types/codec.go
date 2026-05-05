package types

import (
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterInterfaces registers every concrete sdk.Msg implementation
// this module exposes with the provided InterfaceRegistry and wires
// the gRPC Msg service descriptor for amino / tx decoding.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgCreateSpotMarket{},
		&MsgUpdateSpotMarketStatus{},
		&MsgPlaceSpotOrder{},
		&MsgCancelSpotOrder{},
		&MsgCancelAllSpotOrders{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
