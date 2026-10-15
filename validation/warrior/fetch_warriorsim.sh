#!/usr/bin/env bash
# Clones WarriorSim at the commit our reference numbers were recorded with.
set -euo pipefail
COMMIT=ae03455a9421d04a1563c27393e909cffeb56cf2
DIR="$(cd "$(dirname "$0")" && pwd)/.cache/WarriorSim"
if [ ! -d "$DIR/.git" ]; then
  git clone --quiet https://github.com/GuybrushGit/WarriorSim "$DIR"
fi
git -C "$DIR" fetch --quiet origin "$COMMIT" 2>/dev/null || true
git -C "$DIR" checkout --quiet "$COMMIT"
echo "WarriorSim at $(git -C "$DIR" rev-parse --short HEAD)"
