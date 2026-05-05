package ltrstchain

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	modulev1 "ltrstchain/api/ltrstchain/ltrstchain"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: modulev1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				// this line is used by ignite scaffolding # autocli/query
			},
			SubCommands: map[string]*autocliv1.ServiceCommandDescriptor{
				"privacy": {
					Service: modulev1.QueryPrivacy_ServiceDesc.ServiceName,
					RpcCommandOptions: []*autocliv1.RpcCommandOptions{
						{
							RpcMethod: "Commitment",
							Use:       "commitment [commit-hex]",
							Short:     "Look up a stored shielded commitment by its hex-encoded 32-byte commit",
							PositionalArgs: []*autocliv1.PositionalArgDescriptor{
								{ProtoField: "commit"},
							},
						},
						{
							RpcMethod: "NullifierUsed",
							Use:       "nullifier-used [nullifier-hex]",
							Short:     "Check whether a hex-encoded 32-byte nullifier has already been spent",
							PositionalArgs: []*autocliv1.PositionalArgDescriptor{
								{ProtoField: "nullifier"},
							},
						},
						{
							RpcMethod: "MerkleRoot",
							Use:       "merkle-root",
							Short:     "Show the current 32-byte rolling Merkle accumulator root",
						},
					},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              modulev1.Msg_ServiceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
			SubCommands: map[string]*autocliv1.ServiceCommandDescriptor{
				"privacy": {
					Service: modulev1.MsgPrivacy_ServiceDesc.ServiceName,
					RpcCommandOptions: []*autocliv1.RpcCommandOptions{
						{
							RpcMethod: "ShieldedTransfer",
							Use:       "shielded-transfer [commitment-json]",
							Short:     "Submit a shielded commitment (LT-Shield-Commit-v2) to the chain",
							Long: "Submit a MsgShieldedTransfer containing a 32-byte commit, " +
								"32-byte nullifier, optional field commitments, and optional " +
								"field openings. The submitter field is always overwritten " +
								"with the tx signer address.",
						},
						{
							RpcMethod: "ShieldedDeposit",
							Use:       "shielded-deposit [commitment-json] [amount]",
							Short:     "Deposit coins into the shielded pool with a commitment",
							Long: "Submit a MsgShieldedDeposit that atomically records a " +
								"shielded commitment AND escrows the supplied amount from " +
								"your account into the ltrstchain_shielded_pool module " +
								"account. The commitment follows LT-Shield-Commit-v2.",
						},
						{
							RpcMethod: "ShieldedWithdraw",
							Use:       "shielded-withdraw [recipient] [amount] [nullifier-hex]",
							Short:     "Withdraw from the shielded pool (authority-gated)",
							Long: "Submit a MsgShieldedWithdraw that releases coins from the " +
								"shielded pool to [recipient]. INTERIM: only the module " +
								"authority (governance) can invoke this. The nullifier must " +
								"be 32 bytes hex-encoded, and is marked spent atomically.",
						},
					},
				},
			},
		},
	}
}
