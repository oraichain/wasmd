package v2

import (
	"github.com/CosmWasm/wasmd/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	NewAllowedMsgs = []types.EIP712AllowedMsg{
		// x/evmutil
		{
			MsgTypeUrl:       "/kava.evmutil.v1beta1.MsgConvertERC20ToCoin",
			MsgValueTypeName: "MsgValueEVMConvertERC20ToCoin",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "initiator", Type: "string"},
				{Name: "receiver", Type: "string"},
				{Name: "orai_erc20_address", Type: "string"},
				{Name: "amount", Type: "string"},
			},
		},
		{
			MsgTypeUrl:       "/kava.evmutil.v1beta1.MsgConvertCoinToERC20",
			MsgValueTypeName: "MsgValueEVMConvertCoinToERC20",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "initiator", Type: "string"},
				{Name: "receiver", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		// x/earn
		{
			MsgTypeUrl:       "/kava.earn.v1beta1.MsgDeposit",
			MsgValueTypeName: "MsgValueEarnDeposit",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "depositor", Type: "string"},
				{Name: "amount", Type: "Coin"},
				{Name: "strategy", Type: "int32"},
			},
		},
		{
			MsgTypeUrl:       "/kava.earn.v1beta1.MsgWithdraw",
			MsgValueTypeName: "MsgValueEarnWithdraw",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "from", Type: "string"},
				{Name: "amount", Type: "Coin"},
				{Name: "strategy", Type: "int32"},
			},
		},
		// x/staking
		{
			MsgTypeUrl:       "/cosmos.staking.v1beta1.MsgDelegate",
			MsgValueTypeName: "MsgValueStakingDelegate",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "delegator_address", Type: "string"},
				{Name: "validator_address", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		{
			MsgTypeUrl:       "/cosmos.staking.v1beta1.MsgUndelegate",
			MsgValueTypeName: "MsgValueStakingUndelegate",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "delegator_address", Type: "string"},
				{Name: "validator_address", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		{
			MsgTypeUrl:       "/cosmos.staking.v1beta1.MsgBeginRedelegate",
			MsgValueTypeName: "MsgValueStakingBeginRedelegate",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "delegator_address", Type: "string"},
				{Name: "validator_src_address", Type: "string"},
				{Name: "validator_dst_address", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		// x/incentive
		{
			MsgTypeUrl:       "/kava.incentive.v1beta1.MsgClaimUSDXMintingReward",
			MsgValueTypeName: "MsgValueIncentiveClaimUSDXMintingReward",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "sender", Type: "string"},
				{Name: "multiplier_name", Type: "string"},
			},
		},
		{
			MsgTypeUrl:       "/kava.incentive.v1beta1.MsgClaimHardReward",
			MsgValueTypeName: "MsgValueIncentiveClaimHardReward",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "sender", Type: "string"},
				{Name: "denoms_to_claim", Type: "IncentiveSelection[]"},
			},
			NestedTypes: []types.EIP712NestedMsgType{
				{
					Name: "IncentiveSelection",
					Attrs: []types.EIP712MsgAttrType{
						{Name: "denom", Type: "string"},
						{Name: "multiplier_name", Type: "string"},
					},
				},
			},
		},
		{
			MsgTypeUrl:       "/kava.incentive.v1beta1.MsgClaimDelegatorReward",
			MsgValueTypeName: "MsgValueIncentiveClaimDelegatorReward",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "sender", Type: "string"},
				{Name: "denoms_to_claim", Type: "IncentiveSelection[]"},
			},
			NestedTypes: []types.EIP712NestedMsgType{
				{
					Name: "IncentiveSelection",
					Attrs: []types.EIP712MsgAttrType{
						{Name: "denom", Type: "string"},
						{Name: "multiplier_name", Type: "string"},
					},
				},
			},
		},
		{
			MsgTypeUrl:       "/kava.incentive.v1beta1.MsgClaimSwapReward",
			MsgValueTypeName: "MsgValueIncentiveClaimSwapReward",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "sender", Type: "string"},
				{Name: "denoms_to_claim", Type: "IncentiveSelection[]"},
			},
			NestedTypes: []types.EIP712NestedMsgType{
				{
					Name: "IncentiveSelection",
					Attrs: []types.EIP712MsgAttrType{
						{Name: "denom", Type: "string"},
						{Name: "multiplier_name", Type: "string"},
					},
				},
			},
		},
		{
			MsgTypeUrl:       "/kava.incentive.v1beta1.MsgClaimSavingsReward",
			MsgValueTypeName: "MsgValueIncentiveClaimSavingsReward",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "sender", Type: "string"},
				{Name: "denoms_to_claim", Type: "IncentiveSelection[]"},
			},
			NestedTypes: []types.EIP712NestedMsgType{
				{
					Name: "IncentiveSelection",
					Attrs: []types.EIP712MsgAttrType{
						{Name: "denom", Type: "string"},
						{Name: "multiplier_name", Type: "string"},
					},
				},
			},
		},
		{
			MsgTypeUrl:       "/kava.incentive.v1beta1.MsgClaimEarnReward",
			MsgValueTypeName: "MsgValueIncentiveClaimEarnReward",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "sender", Type: "string"},
				{Name: "denoms_to_claim", Type: "IncentiveSelection[]"},
			},
			NestedTypes: []types.EIP712NestedMsgType{
				{
					Name: "IncentiveSelection",
					Attrs: []types.EIP712MsgAttrType{
						{Name: "denom", Type: "string"},
						{Name: "multiplier_name", Type: "string"},
					},
				},
			},
		},
		// x/router
		{
			MsgTypeUrl:       "/kava.router.v1beta1.MsgMintDeposit",
			MsgValueTypeName: "MsgValueRouterMintDeposit",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "depositor", Type: "string"},
				{Name: "validator", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		{
			MsgTypeUrl:       "/kava.router.v1beta1.MsgDelegateMintDeposit",
			MsgValueTypeName: "MsgValueRouterDelegateMintDeposit",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "depositor", Type: "string"},
				{Name: "validator", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		{
			MsgTypeUrl:       "/kava.router.v1beta1.MsgWithdrawBurn",
			MsgValueTypeName: "MsgValueRouterWithdrawBurn",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "from", Type: "string"},
				{Name: "validator", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		{
			MsgTypeUrl:       "/kava.router.v1beta1.MsgWithdrawBurnUndelegate",
			MsgValueTypeName: "MsgValueRouterWithdrawBurnUndelegate",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "from", Type: "string"},
				{Name: "validator", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
		},
		// x/gov
		{
			MsgTypeUrl:       "/cosmos.gov.v1beta1.MsgVote",
			MsgValueTypeName: "MsgValueGovVote",
			ValueTypes: []types.EIP712MsgAttrType{
				{Name: "proposal_id", Type: "uint64"},
				{Name: "voter", Type: "string"},
				{Name: "option", Type: "int32"},
			},
		},
	}
)

// MigrateStore sets the default AllowUnprotectedTxs parameter.
func MigrateStore(ctx sdk.Context, paramstore *paramtypes.Subspace) error {
	if !paramstore.HasKeyTable() {
		ps := paramstore.WithKeyTable(types.ParamKeyTable())
		paramstore = &ps
	}
	paramstore.Set(ctx, types.ParamStoreKeyEIP712AllowedMsgs, NewAllowedMsgs)
	return nil
}
