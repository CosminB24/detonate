#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
OUT_DIR="$(cd .. && pwd)/out"

mkdir -p "$OUT_DIR"

docker run \
  --rm \
  --network=none \
  --cap-add=SYS_PTRACE \
  --security-opt seccomp=unconfined \
  -v "$OUT_DIR:/out" \
  detonate-spike \
  strace -f -tt -y -s 512 -e trace=%process,%file,%network -o /out/clean.log \
  npm install --offline --no-audit --no-fund --no-save --ignore-scripts /fixtures/lodash-4.17.21.tgz

docker run \
  --rm \
  --network=none \
  --cap-add=SYS_PTRACE \
  --security-opt seccomp=unconfined \
  -v "$OUT_DIR:/out" \
  -v "$PWD/testdata/evil-pkg:/pkg:ro" \
  detonate-spike \
  strace -f -tt -y -s 512 -e trace=%process,%file,%network -o /out/evil.log \
  npm install --offline --no-audit --no-fund --no-save /pkg

echo
wc -l "$OUT_DIR/clean.log" "$OUT_DIR/evil.log"
