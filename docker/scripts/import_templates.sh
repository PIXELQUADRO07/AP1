#!/bin/sh
set -e

SRC=${1:-}
DST=${2:-$(pwd)/../config/templates}

if [ -z "$SRC" ]; then
  echo "Usage: $0 <source_templates_dir> [destination_dir]"
  exit 1
fi

if [ ! -d "$SRC" ]; then
  echo "Source templates dir not found: $SRC"
  exit 1
fi

mkdir -p "$DST"
cp -a "$SRC"/. "$DST"/

echo "Imported templates from $SRC to $DST"
