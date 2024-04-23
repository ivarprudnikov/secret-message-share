#!/bin/bash

set -euo pipefail

RG="secret-share-northeurope-rg"
FN_NAME="secret-share"
# Remove prior versions
rm -f bin/server package.zip
# Build the binary
GOOS=linux GOARCH=amd64 go build -o bin/server .
# Prepare function package
zip -r package . --include "bin/*" "func-match-all-config/*" host.json
# Deploy the package
az functionapp deployment source config-zip -g $RG -n $FN_NAME --src package.zip
# Preview hostnames 
az functionapp show -g $RG -n $FN_NAME --query hostNames

