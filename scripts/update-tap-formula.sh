#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
if [[ -z "$version" ]]; then
  echo "usage: scripts/update-tap-formula.sh X.Y.Z [formula_path]" >&2
  exit 2
fi
version="${version#v}"
tag="v${version}"

formula_path="${2:-${TRELLI_FORMULA_PATH:-../homebrew-tap/Formula/trelli.rb}}"
repo="${TRELLI_GITHUB_REPO:-multikoop/trelli}"

if [[ ! -f "$formula_path" ]]; then
  echo "missing formula at $formula_path" >&2
  exit 2
fi

tmp_assets_dir="$(mktemp -d -t trelli-formula-update)"
trap 'rm -rf "$tmp_assets_dir"' EXIT

gh release download "$tag" -R "$repo" -p checksums.txt -D "$tmp_assets_dir" >/dev/null
checksums_file="$tmp_assets_dir/checksums.txt"

sha_for_asset() {
  local name="$1"
  awk -v n="$name" '$2==n {print $1}' "$checksums_file"
}

sha_darwin_arm64="$(sha_for_asset "trelli_${version}_darwin_arm64.tar.gz")"
sha_darwin_amd64="$(sha_for_asset "trelli_${version}_darwin_amd64.tar.gz")"
sha_linux_arm64="$(sha_for_asset "trelli_${version}_linux_arm64.tar.gz")"
sha_linux_amd64="$(sha_for_asset "trelli_${version}_linux_amd64.tar.gz")"

for val in "$sha_darwin_arm64" "$sha_darwin_amd64" "$sha_linux_arm64" "$sha_linux_amd64"; do
  if [[ -z "$val" ]]; then
    echo "failed to resolve one or more checksums from checksums.txt" >&2
    exit 2
  fi
done

tmp_formula="$(mktemp -t trelli-formula)"
awk \
  -v version="$version" \
  -v repo="$repo" \
  -v s_da64="$sha_darwin_arm64" \
  -v s_dx64="$sha_darwin_amd64" \
  -v s_la64="$sha_linux_arm64" \
  -v s_lx64="$sha_linux_amd64" '
  BEGIN { target = "" }
  {
    if ($0 ~ /^[[:space:]]*version "/) {
      sub(/version ".*"/, "version \"" version "\"")
    }

    if ($0 ~ /darwin_arm64\.tar\.gz"$/) {
      sub(/url ".*"/, "url \"https://github.com/" repo "/releases/download/v#{version}/trelli_#{version}_darwin_arm64.tar.gz\"")
      target = "darwin_arm64"
    } else if ($0 ~ /darwin_amd64\.tar\.gz"$/) {
      sub(/url ".*"/, "url \"https://github.com/" repo "/releases/download/v#{version}/trelli_#{version}_darwin_amd64.tar.gz\"")
      target = "darwin_amd64"
    } else if ($0 ~ /linux_arm64\.tar\.gz"$/) {
      sub(/url ".*"/, "url \"https://github.com/" repo "/releases/download/v#{version}/trelli_#{version}_linux_arm64.tar.gz\"")
      target = "linux_arm64"
    } else if ($0 ~ /linux_amd64\.tar\.gz"$/) {
      sub(/url ".*"/, "url \"https://github.com/" repo "/releases/download/v#{version}/trelli_#{version}_linux_amd64.tar.gz\"")
      target = "linux_amd64"
    }

    if ($0 ~ /^[[:space:]]*sha256 "/) {
      if (target == "darwin_arm64") {
        sub(/sha256 ".*"/, "sha256 \"" s_da64 "\"")
      } else if (target == "darwin_amd64") {
        sub(/sha256 ".*"/, "sha256 \"" s_dx64 "\"")
      } else if (target == "linux_arm64") {
        sub(/sha256 ".*"/, "sha256 \"" s_la64 "\"")
      } else if (target == "linux_amd64") {
        sub(/sha256 ".*"/, "sha256 \"" s_lx64 "\"")
      }
      target = ""
    }

    print
  }
' "$formula_path" > "$tmp_formula"

mv "$tmp_formula" "$formula_path"

echo "Updated $formula_path for $tag using checksums from $repo."
