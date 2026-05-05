package bridge

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "ltrstchain/api/ltrstchain/bridge"
)

// AutoCLIOptions declaratively wires `ltrstchaind q bridge ...` and
// `ltrstchaind tx bridge ...` subcommands from the generated service
// descriptors. Mutating Msgs default to Skip because they are gov-
// gated; leave CLI discoverability for read ops.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		// --------------- query ---------------
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Show the module parameters",
				},
				{
					RpcMethod: "BridgeAsset",
					Use:       "asset [ibc-denom]",
					Short:     "Fetch a single bridge asset by ibc_denom",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "ibc_denom"},
					},
				},
				{
					RpcMethod: "BridgeAssets",
					Use:       "assets",
					Short:     "List every registered bridge asset (paginated)",
				},
			},
		},
		// --------------- tx ---------------
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// All mutating messages are authority-only (x/gov by
				// default). Skip them in autocli so users don't try
				// to sign with wallet keys.
				{RpcMethod: "UpdateParams", Skip: true},
				{RpcMethod: "RegisterBridgeAsset", Skip: true},
				{RpcMethod: "UpdateBridgeAsset", Skip: true},
				{RpcMethod: "RemoveBridgeAsset", Skip: true},
			},
		},
	}
}
