#!/bin/bash

# Exit on first error
set -e

# Check if jq is installed
if ! [ -x "$(command -v jq)" ]; then
    echo "Error: jq is not installed." >&2
    exit 1
fi

if [ $# -lt 2 ]; then
    echo "Please specify: [path/to/artifacts/.json] [path/to/output/.json]"
    exit 1
fi

# Check if $1 exists
if [ ! -f "$1" ]; then
    echo "File not found: $1"
    exit 1
fi

# Confirm if $2 exists
if [ -f "$2" ]; then
    read -p "File exists: $2. Overwrite? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Convert artifact to the correct format Ethermint can use
jq '{ abi: .abi | tostring, bin: .bytecode | ltrimstr("0x")}' \
    "$1" \
    > "$2"

echo "Generated Ethermint JSON file at $2"
