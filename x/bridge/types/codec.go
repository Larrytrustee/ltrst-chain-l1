package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterInterfaces wires the Msg types so tx decoders can decode
// MsgUpdateParams, MsgRegisterBridgeAsset, etc., by their typeURL.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgRegisterBridgeAsset{},
		&MsgUpdateBridgeAsset{},
		&MsgRemoveBridgeAsset{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec is a no-op — all Msgs are proto3 with
// signer annotations. Kept so module.go can call it without special-
// casing.
func RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}
