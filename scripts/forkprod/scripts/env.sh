#!/usr/bin/env bash
# Shared paths, ports and process helpers. The three validators run as local processes on
# distinct ports — no Docker required.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../.." && pwd)"
DATA_DIR="${ROOT_DIR}/data"
FIX="${DATA_DIR}/fixtures.json"

BIN_NEW="${BIN_NEW:-${DATA_DIR}/oraid-prod}"   # current workspace, production fixtures
BIN_OLD="${BIN_OLD:-${DATA_DIR}/oraid-old}"    # v0.50.13b
BIN="${BIN:-${BIN_NEW}}"                       # default for genesis/query helpers

NODES=(a b s)

# node -> p2p rpc lcd grpc jsonrpc
port_p2p()     { case $1 in a) echo 27656;; b) echo 27666;; s) echo 27676;; esac; }
port_rpc()     { case $1 in a) echo 27657;; b) echo 27667;; s) echo 27677;; esac; }
port_lcd()     { case $1 in a) echo 1417;;  b) echo 1427;;  s) echo 1437;;  esac; }
port_grpc()    { case $1 in a) echo 9190;;  b) echo 9200;;  s) echo 9210;;  esac; }
port_jsonrpc() { case $1 in a) echo 18545;; b) echo 18555;; s) echo 18565;; esac; }

node_home() { echo "${DATA_DIR}/node-$1"; }
rpc_url()   { echo "http://127.0.0.1:$(port_rpc "$1")"; }
lcd_url()   { echo "http://127.0.0.1:$(port_lcd "$1")"; }
pid_file()  { echo "${DATA_DIR}/node-$1.pid"; }

json() { sed -n '/^[[{]/,$p'; }

# start_node <node> <binary> [extra args...]
start_node() {
  local n=$1 bin=$2; shift 2
  local home; home=$(node_home "${n}")
  "${bin}" start --home "${home}" \
    --rpc.laddr "tcp://127.0.0.1:$(port_rpc "${n}")" \
    --p2p.laddr "tcp://127.0.0.1:$(port_p2p "${n}")" \
    "$@" >>"${home}/start.log" 2>&1 &
  echo $! > "$(pid_file "${n}")"
}

stop_node() {
  local n=$1 pf
  pf=$(pid_file "${n}")
  [[ -f "${pf}" ]] || return 0
  kill "$(cat "${pf}")" 2>/dev/null || true
  for _ in $(seq 1 30); do
    kill -0 "$(cat "${pf}")" 2>/dev/null || break
    sleep 1
  done
  kill -9 "$(cat "${pf}")" 2>/dev/null || true
  rm -f "${pf}"
}

stop_all() { for n in "${NODES[@]}"; do stop_node "${n}"; done; }

height() {
  curl -sf "$(rpc_url "$1")/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // 0'
}

node_alive() {
  local pf; pf=$(pid_file "$1")
  [[ -f "${pf}" ]] && kill -0 "$(cat "${pf}")" 2>/dev/null
}
