#!/usr/bin/env bash

set -xe

GOOGLE_DRIVE_TOKEN_DIR="${GOOGLE_DRIVE_SECRET:-$HOME/.google_api}"
GOOGLE_DRIVE_SECRET="${GOOGLE_DRIVE_SECRET:-$HOME/.google_api/client_secret.json}"
GOOGLE_DRIVE_FOLDER_ID="${GOOGLE_DRIVE_FOLDER_ID:-1ppT--AjpAddQol6nDXxiDMfhjrc-bMaR}"

rm -rf exported

go run -C tools ./cmd/exporter/main.go --client-secret $GOOGLE_DRIVE_SECRET --client-token-dir $GOOGLE_DRIVE_TOKEN_DIR --folder-id $GOOGLE_DRIVE_FOLDER_ID --destination $(pwd)/exported
