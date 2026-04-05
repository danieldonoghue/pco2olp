#!/usr/bin/env python3
"""Update versions.json for the gh-pages docs site.

Reads VERSION from the environment. Adds it to versions.json,
sorts descending by semver, and updates the 'latest' key.
"""
import json
import os
import sys


def ver_key(v: str) -> list[int]:
    try:
        return [int(x) for x in v.lstrip("v").split(".")]
    except ValueError:
        return [0]


def main() -> None:
    version = os.environ.get("VERSION")
    if not version:
        print("ERROR: VERSION environment variable not set", file=sys.stderr)
        sys.exit(1)

    path = "versions.json"
    try:
        with open(path) as f:
            data = json.load(f)
    except FileNotFoundError:
        data = {"versions": [], "latest": ""}

    versions = data.get("versions", [])
    if version not in versions:
        versions.insert(0, version)
    versions.sort(key=ver_key, reverse=True)

    data["versions"] = versions
    data["latest"] = versions[0]

    with open(path, "w") as f:
        json.dump(data, f, indent=2)
        f.write("\n")

    print("versions.json updated:", data)


if __name__ == "__main__":
    main()
