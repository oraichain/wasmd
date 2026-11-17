## Messages

### Fee token admin (gov authority)

| Msg | Authority | Effect |
|---|---|---|
| `MsgUpdateParams` | module authority (gov) | Replace module params |
| `MsgAddFeeToken` | module authority (gov) | Allow a denom for fees |
| `MsgRemoveFeeToken` | module authority (gov) | Disallow a fee denom |

### Blacklist admin (foundation x/group policy)

| Msg | Authority | Effect |
|---|---|---|
| `MsgAddBlacklist` | `types.FoundationGroupPolicyAddress` | Persist address in txfees blacklist store |
| `MsgRemoveBlacklist` | `types.FoundationGroupPolicyAddress` | Delete address from blacklist store |
| `MsgRemoveBlacklistAndRecover` | `types.FoundationGroupPolicyAddress` | Remove from blacklist, then transfer spendable bank balances from that address to `to_address` |

`FoundationGroupPolicyAddress` is the foundation **x/group policy account**.

Execution path:

1. Create foundation group + policy (`MsgCreateGroupWithPolicy`)
2. Set `FoundationGroupPolicyAddress` to that policy address
3. Members `MsgSubmitProposal` with `MsgAddBlacklist` / `MsgRemoveBlacklist` / `MsgRemoveBlacklistAndRecover` (`authority` = policy)
4. Vote → `MsgExec`
5. Group ensures msg signers == policy, then txfees checks the same constant

`MsgRemoveBlacklistAndRecover` order: validate → remove blacklist entry → `SpendableCoins(from)` → `SendCoins(from, to)`. Empty balances succeed (remove only). Native bank coins only (not CW20).

Do **not** use CW3 flex multisig for these msgs (that contract build lacks `stargate`/`Any`). Ante `BlacklistDecorator` still rejects blacklisted signers / authz granters.
