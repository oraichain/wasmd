#!/bin/bash
# Before running this script, you must setup local network:

set -e

HIDE_LOGS="/dev/null"

blocks_per_yer=$(oraid query mint params --output json | jq '.params.blocks_per_year | tonumber')
goal_bonded=$(oraid query mint params --output json | jq '.params.goal_bonded | tonumber')
inflation_max=$(oraid query mint params --output json | jq '.params.inflation_max | tonumber')
inflation_min=$(oraid query mint params --output json | jq '.params.inflation_min | tonumber')
inflation_rate_change=$(oraid query mint params --output json | jq '.params.inflation_rate_change | tonumber')
mint_denom=$(oraid query mint params --output json | jq '.params.mint_denom')

if [[ $blocks_per_yer -ne 39420000 ]] ; then
   echo "Mint params Upgrade Failed" >&2; exit 1
fi

if [[ $goal_bonded -ne 670000000000000000 ]] ; then
   echo "Mint params Upgrade Failed" >&2; exit 1
fi

if [[ $inflation_max -ne 85000000000000000 ]] ; then
   echo "Mint params Upgrade Failed" >&2; exit 1
fi

if [[ $inflation_min -ne 85000000000000000 ]] ; then
   echo "Mint params Upgrade Failed" >&2; exit 1
fi

if [[ $inflation_rate_change -ne 130000000000000000 ]] ; then
   echo "Mint params Upgrade Failed" >&2; exit 1
fi

denom="orai"
trimmed_mint_denom=$(echo "$mint_denom" | tr -d '"')  # Deletes all double quotes
if ! [[ $trimmed_mint_denom == $denom ]] ; then
   echo "Mint params Upgrade Failed" >&2; exit 1
fi

echo "Test Mint Params Passed";