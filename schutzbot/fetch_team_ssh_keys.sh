#!/bin/bash
# Fetch SSH public keys from GitHub for team members listed in
# github_usernames.txt.  Outputs keys to stdout, one per line, suitable
# for appending to ~/.ssh/authorized_keys.
#
# When GITHUB_TOKEN is set, a single GraphQL request fetches every
# user's keys at once.  Otherwise falls back to one REST call per user
# (unauthenticated, subject to lower rate limits).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
USERNAMES_FILE="${SCRIPT_DIR}/github_usernames.txt"

if [[ ! -f "$USERNAMES_FILE" ]]; then
    echo "Error: ${USERNAMES_FILE} not found" >&2
    exit 1
fi

mapfile -t USERS < <(sed 's/#.*//' "$USERNAMES_FILE" | sed 's/^[[:blank:]]*//;s/[[:blank:]]*$//' | grep -v '^$')

if [[ ${#USERS[@]} -eq 0 ]]; then
    echo "Warning: no usernames in ${USERNAMES_FILE}" >&2
    exit 0
fi

for u in "${USERS[@]}"; do
    if [[ ! "$u" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$ ]]; then
        echo "Error: invalid GitHub username '${u}'" >&2
        exit 1
    fi
done

fetch_keys_graphql() {
    local query="{"
    for i in "${!USERS[@]}"; do
        query+=" u${i}: user(login: \"${USERS[$i]}\") {"
        query+=" publicKeys(first: 100) { nodes { key } } }"
    done
    query+=" }"

    local payload
    payload=$(jq -nc --arg q "$query" '{"query": $q}')

    local response
    response=$(curl -sf \
        -H "Authorization: bearer ${GITHUB_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        https://api.github.com/graphql) || return 1

    if echo "$response" | jq -e '.errors' &>/dev/null; then
        echo "GraphQL errors:" >&2
        echo "$response" | jq -r '.errors[].message' >&2
        return 1
    fi

    echo "$response" | jq -r '
        .data | to_entries[] | .value.publicKeys.nodes[]?.key // empty
    '
}

fetch_keys_rest() {
    for user in "${USERS[@]}"; do
        if ! curl -sf "https://github.com/${user}.keys"; then
            echo "Warning: failed to fetch keys for ${user}" >&2
        fi
    done
}

if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    if fetch_keys_graphql; then
        exit 0
    fi
    echo "GraphQL failed, falling back to REST" >&2
fi

fetch_keys_rest
