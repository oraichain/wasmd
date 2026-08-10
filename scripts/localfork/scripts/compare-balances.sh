#!/usr/bin/env bash
# Compare balances-pre-fork.json vs balances-post-fork.json and print a summary table.
# Also prints RecoveryAddress dedicated pre→post report from recovery-balances.json
# (captured before bank-send fee noise).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PRE="${ROOT_DIR}/data/balances-pre-fork.json"
POST="${ROOT_DIR}/data/balances-post-fork.json"
REC="${ROOT_DIR}/data/recovery-balances.json"

if [[ ! -f "${PRE}" || ! -f "${POST}" ]]; then
  echo "ERROR: missing ${PRE} and/or ${POST}"
  exit 1
fi

export LOCALFORK_ROOT="${ROOT_DIR}"
python3 - <<'PY'
import json
import os
import sys
from pathlib import Path

root = Path(os.environ["LOCALFORK_ROOT"])
pre = json.loads((root / "data/balances-pre-fork.json").read_text())
post = json.loads((root / "data/balances-post-fork.json").read_text())
rec_path = root / "data/recovery-balances.json"
rec = json.loads(rec_path.read_text()) if rec_path.is_file() else None

print(f"pre  height={pre.get('height')}  post height={post.get('height')}")
print(f"cw20_v1 mint={pre.get('cw20_mint_amount')}  cw20_v2 mint={pre.get('cw20_mint_amount_2')}")
print()

if rec:
    print("=" * 72)
    print(f"RecoveryAddress ({rec.get('address')})  height={rec.get('height')}  [before bank-send tests]")
    print("=" * 72)
    o = rec["orai"]
    want = o.get("want", o.get("pre"))
    pool_rem = o.get("pool_remainder", "0")
    print(f"  ORAI     {o['pre']:>16} → {o['post']:<16}  (want {want}; pool_remainder={pool_rem})")
    c1 = rec["cw20_v1"]
    print(f"  CW20 v1  {c1['pre']:>16} → {c1['post']:<16}  (want {c1['want']})")
    c2 = rec["cw20_v2"]
    print(f"  CW20 v2  {c2['pre']:>16} → {c2['post']:<16}  (want {c2['want']})")
    for n in rec.get("native") or []:
        d = n["denom"]
        short = d if len(d) < 48 else d[:20] + "…" + d[-20:]
        print(f"  native   {n['pre']:>16} → {n['post']:<16}  (want {n['want']})  {short}")
    # hard assert
    ok = True
    if o["post"] != want:
        print(f"ERROR: RecoveryAddress ORAI post={o['post']} want={want}")
        ok = False
    if c1["post"] != c1["want"]:
        print(f"ERROR: CW20 v1 post={c1['post']} want={c1['want']}")
        ok = False
    if c2["post"] != c2["want"]:
        print(f"ERROR: CW20 v2 post={c2['post']} want={c2['want']}")
        ok = False
    for n in rec.get("native") or []:
        if n["post"] != n["want"] or n["pre"] != "0":
            print(f"ERROR: native {n['denom']} pre={n['pre']} post={n['post']} want={n['want']}")
            ok = False
    if not ok:
        sys.exit(1)
    print("✓ RecoveryAddress pre→post assertions OK")
    print()

post_by = {w["address"]: w for w in post["wallets"]}
rows = []
for w in pre["wallets"]:
    a = w["address"]
    p = post_by.get(a, {})
    rows.append({
        "role": w.get("role", ""),
        "orai_pre": w.get("orai", "0"),
        "orai_post": p.get("orai", "0"),
        "cw20_v1_pre": w.get("cw20_v1", "0"),
        "cw20_v1_post": p.get("cw20_v1", "0"),
        "cw20_v2_pre": w.get("cw20_v2", "0"),
        "cw20_v2_post": p.get("cw20_v2", "0"),
    })

print("Full wallet table (post snapshot may include bank-send fee on tester ORAI):")
hdr = f"{'role':<28} {'orai_pre':>16} {'orai_post':>16} {'cw20v1_pre':>12} {'cw20v1_post':>12} {'cw20v2_pre':>12} {'cw20v2_post':>12}"
print(hdr)
print("-" * len(hdr))
for r in rows:
    print(f"{r['role']:<28} {r['orai_pre']:>16} {r['orai_post']:>16} {r['cw20_v1_pre']:>12} {r['cw20_v1_post']:>12} {r['cw20_v2_pre']:>12} {r['cw20_v2_post']:>12}")

natives = pre.get("native_denoms") or []
if natives:
    print()
    print("Native denoms (RecoveryFrom → RecoveryAddress) from full snapshots:")
    frm = next((w for w in pre["wallets"] if "recovery_from" in w.get("role", "")), None)
    to = next((w for w in pre["wallets"] if "recovery_to" in w.get("role", "")), None)
    for item in natives:
        denom = item["denom"]
        short = denom if len(denom) < 56 else denom[:24] + "…" + denom[-24:]
        want = item["amount"]
        frm_pre = (frm or {}).get("native", {}).get(denom, "0")
        frm_post = (post_by.get((frm or {}).get("address", ""), {}) or {}).get("native", {}).get(denom, "0")
        to_pre = (to or {}).get("native", {}).get(denom, "0")
        to_post = (post_by.get((to or {}).get("address", ""), {}) or {}).get("native", {}).get(denom, "0")
        print(f"  {short}")
        print(f"    want_mint={want}  from {frm_pre}→{frm_post}  to {to_pre}→{to_post}")
PY
