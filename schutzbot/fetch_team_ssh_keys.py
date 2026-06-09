#!/usr/bin/env python3
"""Fetch SSH public keys from GitHub for team members.

Reads usernames from github_usernames.txt (one per line) and prints
their public SSH keys to stdout, suitable for appending to
~/.ssh/authorized_keys.

When GITHUB_TOKEN is set, a single GraphQL request fetches every
user's keys at once.  Otherwise falls back to one REST call per user
(unauthenticated, subject to lower rate limits).

Compatible with Python 3.6+.
"""

from __future__ import print_function

import json
import os
import re
import sys

try:
    from urllib.request import Request, urlopen
    from urllib.error import URLError
except ImportError:
    from urllib2 import Request, urlopen, URLError


SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
USERNAMES_FILE = os.path.join(SCRIPT_DIR, "github_usernames.txt")
USERNAME_RE = re.compile(r"^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$")


def read_usernames(path):
    if not os.path.isfile(path):
        print("Error: {} not found".format(path), file=sys.stderr)
        sys.exit(1)

    users = []
    with open(path) as fh:
        for line in fh:
            line = line.split("#", 1)[0].strip()
            if line:
                users.append(line)
    return users


def validate_usernames(users):
    for u in users:
        if not USERNAME_RE.match(u):
            print("Error: invalid GitHub username '{}'".format(u),
                  file=sys.stderr)
            sys.exit(1)


def fetch_keys_graphql(users, token):
    aliases = " ".join(
        'u{i}: user(login: "{u}") {{'
        " publicKeys(first: 100) {{ nodes {{ key }} }} }}".format(i=i, u=u)
        for i, u in enumerate(users)
    )
    query = "{ " + aliases + " }"
    payload = json.dumps({"query": query}).encode("utf-8")

    req = Request(
        "https://api.github.com/graphql",
        data=payload,
        headers={
            "Authorization": "bearer " + token,
            "Content-Type": "application/json",
        },
    )

    try:
        resp = urlopen(req)
    except URLError as exc:
        print("GraphQL request failed: {}".format(exc), file=sys.stderr)
        return None

    data = json.loads(resp.read().decode("utf-8"))

    if "errors" in data:
        for err in data["errors"]:
            print("GraphQL error: {}".format(err.get("message", err)),
                  file=sys.stderr)
        return None

    keys = []
    for alias in sorted(data.get("data", {})):
        user_data = data["data"][alias]
        if user_data and "publicKeys" in user_data:
            for node in user_data["publicKeys"]["nodes"]:
                if node.get("key"):
                    keys.append(node["key"])
    return keys


def fetch_keys_rest(users):
    keys = []
    for user in users:
        url = "https://github.com/{}.keys".format(user)
        try:
            resp = urlopen(Request(url))
            body = resp.read().decode("utf-8").strip()
            if body:
                keys.extend(body.splitlines())
        except URLError:
            print("Warning: failed to fetch keys for {}".format(user),
                  file=sys.stderr)
    return keys


def main():
    users = read_usernames(USERNAMES_FILE)
    if not users:
        print("Warning: no usernames in {}".format(USERNAMES_FILE),
              file=sys.stderr)
        return

    validate_usernames(users)

    token = os.environ.get("GITHUB_TOKEN", "")
    if token:
        keys = fetch_keys_graphql(users, token)
        if keys is not None:
            for k in keys:
                print(k)
            return
        print("GraphQL failed, falling back to REST", file=sys.stderr)

    for k in fetch_keys_rest(users):
        print(k)


if __name__ == "__main__":
    main()
