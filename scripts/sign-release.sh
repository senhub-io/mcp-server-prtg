#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: scripts/sign-release.sh <tag> [--repo owner/name]" >&2
  exit 2
}

TAG=""
REPO="senhub-io/mcp-server-prtg"
while [ $# -gt 0 ]; do
  case "$1" in
    --repo)
      REPO="${2:?--repo needs a value}"
      shift 2
      ;;
    -h | --help)
      usage
      ;;
    *)
      if [ -z "$TAG" ]; then
        TAG="$1"
        shift
      else
        usage
      fi
      ;;
  esac
done
[ -n "$TAG" ] || usage

if ! [[ "$TAG" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
  echo "error: invalid tag '$TAG'" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SIGNER="$REPO_ROOT/packaging/windows/sign-authenticode.sh"

# Prerequisites
gh auth status >/dev/null 2>&1 || { echo "error: gh is not authenticated" >&2; exit 1; }
PERM="$(gh repo view "$REPO" --json viewerPermission -q .viewerPermission)"
case "$PERM" in
  ADMIN | MAINTAIN | WRITE) ;;
  *) echo "error: need write access to $REPO (current: $PERM)" >&2; exit 1 ;;
esac
for tool in unzip zip shasum jq; do
  command -v "$tool" >/dev/null 2>&1 || { echo "error: required tool not found: $tool" >&2; exit 1; }
done
[ -x "$SIGNER" ] || { echo "error: signer not found or not executable: $SIGNER" >&2; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# a. Only sign draft releases: modifying a published release would change public assets.
IS_DRAFT="$(gh release view "$TAG" --repo "$REPO" --json isDraft -q .isDraft)"
if [ "$IS_DRAFT" != "true" ]; then
  echo "error: release $TAG is not a draft; refusing to modify a published release" >&2
  exit 1
fi

# b. Check the CI checksums, then sign the exe inside the Windows zip.
ZIP_NAME="mcp-server-prtg-${TAG}-windows-amd64.zip"
MSI_NAME="mcp-server-prtg-${TAG}-amd64.msi"
EXPECTED_ASSETS="SHA256SUMS
${ZIP_NAME}
mcp-server-prtg-${TAG}-linux-amd64.tar.gz
mcp-server-prtg-${TAG}-linux-arm64.tar.gz"

echo ">> checking the draft release against the CI checksums"
mkdir "$WORK/ci"
gh release download "$TAG" --repo "$REPO" --dir "$WORK/ci"
ACTUAL_ASSETS="$(cd "$WORK/ci" && find . -maxdepth 1 -type f | sed "s|^\./||" | LC_ALL=C sort)"
if [ "$ACTUAL_ASSETS" != "$(printf '%s\n' "$EXPECTED_ASSETS" | LC_ALL=C sort)" ]; then
  echo "error: unexpected assets on release $TAG (run this script once, on the release main-release.yml created):" >&2
  printf '%s\n' "$ACTUAL_ASSETS" >&2
  exit 1
fi
(cd "$WORK/ci" && shasum -a 256 -c SHA256SUMS) || {
  echo "error: an asset does not match the SHA256SUMS produced by CI" >&2
  exit 1
}

mkdir "$WORK/unzipped"
unzip -q "$WORK/ci/$ZIP_NAME" -d "$WORK/unzipped"
EXES="$(cd "$WORK/unzipped" && find . -name '*.exe' | LC_ALL=C sort)"
if [ "$EXES" != "./mcp-server-prtg.exe" ]; then
  echo "error: the Windows zip must contain exactly mcp-server-prtg.exe, found:" >&2
  printf '%s\n' "$EXES" >&2
  exit 1
fi
echo ">> signing mcp-server-prtg.exe"
"$SIGNER" "$WORK/unzipped/mcp-server-prtg.exe" --in-place
echo ">> re-zipping and uploading"
(cd "$WORK/unzipped" && zip -q -9 -r "$WORK/$ZIP_NAME" .)
gh release upload "$TAG" --repo "$REPO" "$WORK/$ZIP_NAME" --clobber

# c. Dispatch the MSI build from the signed exe and wait for it.
echo ">> dispatching windows-msi.yml"
DISPATCH_TS="$(date +%s)"
gh workflow run windows-msi.yml --repo "$REPO" --ref "$TAG" \
  -f version="$TAG" -f release_tag="$TAG" -f exe_from_release=true
RUN_ID=""
for _ in $(seq 1 12); do
  RUN_ID="$(gh run list --workflow windows-msi.yml --repo "$REPO" \
    --event workflow_dispatch --limit 5 \
    --json databaseId,createdAt,headBranch \
    | jq -r --arg tag "$TAG" --argjson ts "$DISPATCH_TS" \
      '[.[] | select(.headBranch == $tag and (.createdAt | fromdateiso8601) >= $ts)] | .[0].databaseId // empty')"
  [ -n "$RUN_ID" ] && break
  sleep 5
done
if [ -z "$RUN_ID" ]; then
  echo "error: could not find the dispatched workflow run" >&2
  exit 1
fi
echo ">> watching run $RUN_ID"
gh run watch "$RUN_ID" --repo "$REPO" --exit-status

# d. Download, sign and re-upload the MSI with its checksum.
echo ">> downloading MSI"
gh release download "$TAG" --repo "$REPO" --pattern "$MSI_NAME" --dir "$WORK"
MSI="$WORK/$MSI_NAME"
[ -f "$MSI" ] || { echo "error: MSI not found on release $TAG" >&2; exit 1; }
"$SIGNER" "$MSI" --in-place
(cd "$WORK" && shasum -a 256 "$MSI_NAME" > "$MSI_NAME.sha256")
gh release upload "$TAG" --repo "$REPO" "$MSI" "$WORK/$MSI_NAME.sha256" --clobber

# e. SHA256SUMS: keep the CI entries of the Linux archives (verified above)
# and replace only the files this script signed.
echo ">> updating SHA256SUMS"
{
  grep -v -e "  ${ZIP_NAME}\$" -e "  ${MSI_NAME}\$" "$WORK/ci/SHA256SUMS" | sed 's/  \*/  /'
  (cd "$WORK" && shasum -a 256 "$ZIP_NAME" "$MSI_NAME")
} | LC_ALL=C sort -k2 > "$WORK/SHA256SUMS"
gh release upload "$TAG" --repo "$REPO" "$WORK/SHA256SUMS" --clobber

# f. Summary; publishing is a deliberate manual step.
echo ">> release $TAG is signed and ready (still a draft)"
echo ">> verify on Windows with: Get-AuthenticodeSignature <file>  (expect Status=Valid)"
echo ">> publish with: gh release edit $TAG --repo $REPO --draft=false"
