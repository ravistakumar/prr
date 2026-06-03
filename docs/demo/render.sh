#!/bin/sh
# Render docs/demo.gif reproducibly: build prr, put a fake "claude" on PATH so
# the demo never calls a real agent, then run the VHS tape.
#
# Requires: go, vhs (https://github.com/charmbracelet/vhs).
set -e

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo_root"

bin=$(mktemp -d)
trap 'rm -rf "$bin"' EXIT

go build -o "$bin/prr" ./cmd/prr
cp docs/demo/claude "$bin/claude"
chmod +x "$bin/claude"

PATH="$bin:$PATH" PRR_MODE=confirm vhs docs/demo/demo.tape
echo "Wrote docs/demo.gif"
