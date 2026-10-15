#!/usr/bin/env bash
# Moves a built site from /classic/ to another path, e.g. /classic-gear-sim-fork/ for
# GitHub Pages. The site has /classic/ written into its HTML, CSS and JS.
#   tools/rebase_site.sh dist/classic /classic-gear-sim-fork/
set -euo pipefail
dir=$1
base=$2
[[ $base == /*/ ]] || { echo "base must start and end with /" >&2; exit 1; }
# Only rewrite /classic/ when it starts a path, so links like wowhead.com/classic/ stay as they are.
find "$dir" -type f \( -name '*.html' -o -name '*.js' -o -name '*.css' \) -print0 |
	xargs -0 perl -pi -e 's#(?<![\w.\-])/classic/#'"$base"'#g'
