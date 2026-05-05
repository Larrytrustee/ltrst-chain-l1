package ltrstdex

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "ltrstchain/api/ltrstchain/ltrstdex"
)

// AutoCLIOptions wires declarative `ltrstchaind q ltrstdex ...` and
// `ltrstchaind tx ltrstdex ...` subcommands from the generated
// service descriptors. Only the common surface is exposed here;
// low-level authority-gated messages are skipped so the CLI doesn't
// invite users to send them with a wallet key.
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
					RpcMethod: "SpotMarket",
					Use:       "market [market-id]",
					Short:     "Show a single spot market by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "market_id"},
					},
				},
				{
					RpcMethod: "SpotMarketByTicker",
					Use:       "market-by-ticker [ticker]",
					Short:     "Look up a spot market by its ticker",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "ticker"},
					},
				},
				{
					RpcMethod: "SpotMarkets",
					Use:       "markets",
					Short:     "List all spot markets (paginated)",
				},
				{
					RpcMethod: "SpotOrder",
					Use:       "order [market-id] [order-id]",
					Short:     "Show a single spot order",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "market_id"},
						{ProtoField: "order_id"},
					},
				},
				{
					RpcMethod: "SpotOrdersByOwner",
					Use:       "orders-by-owner [owner]",
					Short:     "List every live order owned by an address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "owner"},
					},
				},
				{
					RpcMethod: "SpotOrdersByMarket",
					Use:       "orders-by-market [market-id]",
					Short:     "List every live order on a market",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "market_id"},
					},
				},
				{
					RpcMethod: "SpotOrderbook",
					Use:       "orderbook [market-id]",
					Short:     "Aggregated price-level snapshot of a market",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "market_id"},
					},
				},
				{
					RpcMethod: "SpotFills",
					Use:       "fills [market-id]",
					Short:     "Fill history for a market (paginated)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "market_id"},
					},
				},
			},
		},
		// --------------- tx ---------------
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				// Gov-gated messages — skip the auto CLI so users don't
				// try to send them with a regular wallet; they should
				// go through the governance proposal flow instead.
				{RpcMethod: "UpdateParams", Skip: true},
				{RpcMethod: "CreateSpotMarket", Skip: true},
				{RpcMethod: "UpdateSpotMarketStatus", Skip: true},

				{
					RpcMethod: "PlaceSpotOrder",
					Use:       "place-spot-order [market-id] [side] [type] [tif] [price] [quantity]",
					Short:     "Place a spot order. Side: BUY|SELL. Type: LIMIT|MARKET. Tif: GTC|IOC|FOK.",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "market_id"},
						{ProtoField: "side"},
						{ProtoField: "type"},
						{ProtoField: "tif"},
						{ProtoField: "price"},
						{ProtoField: "quantity"},
					},
				},
				{
					RpcMethod: "CancelSpotOrder",
					Use:       "cancel-spot-order [market-id] [order-id]",
					Short:     "Cancel a single resting order by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "market_id"},
						{ProtoField: "order_id"},
					},
				},
				{
					RpcMethod: "CancelAllSpotOrders",
					Use:       "cancel-all-spot-orders [market-id]",
					Short:     "Cancel every live order owned by the signer (market_id=0 ⇒ all markets)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "market_id"},
					},
				},
			},
		},
	}
}
