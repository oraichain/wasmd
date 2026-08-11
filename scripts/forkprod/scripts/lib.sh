#!/usr/bin/env bash
# Shared helpers for the single-node scenario tests (07, 08).
# The 3-validator lane covers consensus behaviour; these two cover deterministic
# properties of the fork logic itself, so one validator is enough and much faster.

json() { sed -n '/^[[{]/,$p'; }

# make_single_node <home> <chain-id> <initial-height> [underfund_revert_addr]
#
# Builds a single-validator genesis from the compiled production fixtures. If
# underfund_revert_addr is given, that RevertAddress is funded BELOW its burn Amount so the
# fork's bal.LT(entry.Amount) guard trips — used by the panic scenario.
make_single_node() {
  local home=$1 chain_id=$2 initial_height=$3 underfund=${4:-}
  local fix="${DATA_DIR}/fixtures.json"
  local denom; denom=$(jq -r '.denom' "${fix}")

  rm -rf "${home}"
  "${BIN}" init scenario --home "${home}" --chain-id "${chain_id}" >/dev/null 2>&1
  "${BIN}" keys add val --home "${home}" --keyring-backend test >/dev/null 2>&1
  local val; val=$("${BIN}" keys show val -a --home "${home}" --keyring-backend test)

  "${BIN}" genesis add-genesis-account "${val}" "100000000000${denom}" \
    --home "${home}" --keyring-backend test >/dev/null 2>&1

  while IFS= read -r addr; do
    "${BIN}" genesis add-genesis-account "${addr}" "777000000${denom}" \
      --home "${home}" --keyring-backend test >/dev/null 2>&1
  done < <(jq -r '.blacklist_addresses[]' "${fix}")

  while IFS=$'\t' read -r addr amount; do
    local total
    if [[ "${addr}" == "${underfund}" ]]; then
      total=$(python3 -c "print(int('${amount}') - 1)")
    else
      total=$(python3 -c "print(int('${amount}') + 1000000)")
    fi
    "${BIN}" genesis add-genesis-account "${addr}" "${total}${denom}" \
      --home "${home}" --keyring-backend test >/dev/null 2>&1
  done < <(jq -r '.revert_addresses[] | [.address, .amount] | @tsv' "${fix}")

  local native
  native=$(jq -r '[.recovery_native_denoms[] | "123000000" + .] | join(",")' "${fix}")
  while IFS= read -r addr; do
    "${BIN}" genesis add-genesis-account "${addr}" "${native}" \
      --home "${home}" --keyring-backend test --append >/dev/null 2>&1
  done < <(jq -r '.recovery_from_address[]' "${fix}")

  "${BIN}" genesis gentx val "50000000${denom}" \
    --home "${home}" --keyring-backend test --chain-id "${chain_id}" >/dev/null 2>&1
  "${BIN}" genesis collect-gentxs --home "${home}" >/dev/null 2>&1

  local gen="${home}/config/genesis.json"
  jq --slurpfile wasm "${DATA_DIR}/wasm-genesis.json" --arg ih "${initial_height}" \
    '.app_state.wasm = $wasm[0] | .initial_height = $ih' "${gen}" > "${gen}.tmp"
  mv "${gen}.tmp" "${gen}"

  sed -i.bak "s|^minimum-gas-prices = .*|minimum-gas-prices = \"0${denom}\"|" "${home}/config/app.toml"
  sed -i.bak 's|^timeout_commit = .*|timeout_commit = "1s"|' "${home}/config/config.toml"
  rm -f "${home}/config/app.toml.bak" "${home}/config/config.toml.bak"
}
