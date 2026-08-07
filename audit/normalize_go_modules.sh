#!/usr/bin/env bash
set -u
cd "$(dirname "$0")/.."
log="audit/go_module_normalization.log"
: > "$log"
while IFS= read -r mod; do
  dir="${mod%/go.mod}"
  printf '\n== %s ==\n' "$dir" | tee -a "$log"
  (cd "$dir" && gofmt -w $(find . -name '*.go' -type f -not -path './vendor/*') && GOTOOLCHAIN=local go mod tidy && GOTOOLCHAIN=local go test ./...) >> "$log" 2>&1
  status=$?
  printf 'STATUS=%s\n' "$status" | tee -a "$log"
done < <(find . -name go.mod -not -path '*/node_modules/*' -print | sort)
