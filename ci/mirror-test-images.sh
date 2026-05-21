#!/usr/bin/env bash
# Mirror Docker Hub test images to ghcr.io/thin-edge/test-images to avoid pull rate limits.
# Usage: ./ci/mirror-test-images.sh
# Prerequisites:
#   - skopeo installed (brew install skopeo  /  apt install skopeo)
#   - logged in to GHCR:
#       gh auth token | skopeo login ghcr.io -u $(gh api user --jq .login) --password-stdin
set -euo pipefail

GHCR_NAMESPACE="ghcr.io/thin-edge/test-images"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_LIST="${SCRIPT_DIR}/../tests/test-images.txt"

while read -r ghcr_tag sources_str || [[ -n "${ghcr_tag:-}" ]]; do
    # skip blank lines and comments
    [[ "${ghcr_tag:-}" =~ ^#  ]] && continue
    [[ -z "${ghcr_tag:-}"     ]] && continue

    # Allow the first column to be a fully-qualified destination (e.g. a
    # private namespace); otherwise prepend the default GHCR namespace.
    if [[ "${ghcr_tag}" == ghcr.io/* ]]; then
        dest="${ghcr_tag}"
    else
        dest="${GHCR_NAMESPACE}/${ghcr_tag}"
    fi

    # Try each whitespace-separated source left-to-right; use the first reachable one.
    # skopeo inspect is a single manifest GET — cheap against rate limits.
    source_image=""
    read -ra source_list <<< "${sources_str}"
    for candidate in "${source_list[@]}"; do
        if skopeo inspect --raw "docker://${candidate}" >/dev/null 2>&1; then
            source_image="${candidate}"
            break
        fi
        echo "Source unavailable, skipping: ${candidate}"
    done

    if [[ -z "${source_image}" ]]; then
        echo "ERROR: no reachable source found for ${ghcr_tag} (tried: ${sources_str})" >&2
        exit 1
    fi

    # Compare manifest digests — skip if destination is already up to date.
    # Destination check hits GHCR only; source check is one manifest GET.
    dest_digest=$(skopeo inspect --raw "docker://${dest}" 2>/dev/null | sha256sum | awk '{print $1}' || echo "none")
    src_digest=$(skopeo inspect --raw "docker://${source_image}" | sha256sum | awk '{print $1}')

    if [[ "${src_digest}" == "${dest_digest}" ]]; then
        echo "Already up to date: ${dest} (${src_digest:0:12}…), skipping"
        continue
    fi

    echo "Mirroring ${source_image} → ${dest}"
    # --all preserves the full multi-arch manifest index (no layers pulled locally)
    # --image-parallel-copies 1 serialises uploads to avoid GHCR burst rate limits
    # --retry-times 3 handles transient failures automatically
    skopeo copy \
        --all \
        --retry-times 3 \
        --image-parallel-copies 3 \
        "docker://${source_image}" \
        "docker://${dest}"
done < "${IMAGE_LIST}"
