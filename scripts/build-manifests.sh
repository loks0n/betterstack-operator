#!/usr/bin/env bash

set -euo pipefail

: "${VERSION:?VERSION environment variable must be set}"
: "${IMAGE:?IMAGE environment variable must be set}"
IMAGE_NAME=${IMAGE_NAME:-ghcr.io/loks0n/betterstack-operator}

DIST_DIR=${DIST_DIR:-dist}
MANAGER_DIR="config/manager"
CRD_DIR=${CRD_DIR:-config/crd/bases}
MANAGER_OUTPUT="${DIST_DIR}/manager.yaml"
ARCHIVE_OUTPUT="${DIST_DIR}/betterstack-operator-${VERSION}.tar.gz"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${DIST_DIR}"

cp -r "${MANAGER_DIR}" "${TMP_DIR}/manager"
(
  cd "${TMP_DIR}/manager"
  kustomize edit set image "${IMAGE_NAME}=${IMAGE}:${VERSION}"
  kustomize build . > "${MANAGER_OUTPUT}"
)

CRD_OUTPUT_BASENAMES=()
if [ -d "${CRD_DIR}" ]; then
  while IFS= read -r -d '' crd_file; do
    crd_dest="${DIST_DIR}/$(basename "${crd_file}")"
    cp "${crd_file}" "${crd_dest}"
    CRD_OUTPUT_BASENAMES+=("$(basename "${crd_dest}")")
  done < <(find "${CRD_DIR}" -maxdepth 1 -type f \( -name '*.yaml' -o -name '*.yml' \) -print0 | sort -z)
fi

tar czf "${ARCHIVE_OUTPUT}" -C "${DIST_DIR}" \
  "$(basename "${MANAGER_OUTPUT}")" \
  "${CRD_OUTPUT_BASENAMES[@]}"
