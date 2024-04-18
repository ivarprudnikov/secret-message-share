#!/bin/bash

set -euo pipefail

# Function app and storage account names must be unique.

LOCATION="northeurope"
RG="secret-share-northeurope-rg"
STORAGE_ACCOUNT="secretsharenortheurope"
STORAGE_SKU="Standard_LRS"
STORAGE_FILE_SHARE_NAME="secret-share-northeurope-file-share"
STORAGE_FILE_SHARE_DIR="secret-share-northeurope-dir"
FN_NAME="secret-share"
FN_VER="4"
FN_STORAGE_SHARE_ID="linked-generic-file-share"
FN_STORAGE_MOUNT_PATH="/var/tmp/fn"

# Create a resource group
echo "Creating $RG in "$LOCATION"..."
az group create --name $RG --location "$LOCATION"

# Create an Azure storage account in the resource group.
echo "Creating $STORAGE_ACCOUNT"
az storage account create --name $STORAGE_ACCOUNT --location "$LOCATION" --resource-group $RG --sku $STORAGE_SKU

# Set the storage account key as an environment variable. 
export AZURE_STORAGE_KEY=$(az storage account keys list -g $RG -n $STORAGE_ACCOUNT --query '[0].value' -o tsv)

# Create a serverless function app in the resource group.
echo "Creating $FN_NAME"
az functionapp create --name $FN_NAME \
--storage-account $STORAGE_ACCOUNT \
--consumption-plan-location "$LOCATION" \
--resource-group $RG \
--os-type Linux \
--runtime custom \
--functions-version $FN_VER

# Work with Storage account using the set env variables.
# Create a share in Azure Files.
echo "Creating $STORAGE_FILE_SHARE_NAME"
az storage share create --name $STORAGE_FILE_SHARE_NAME --account-name $STORAGE_ACCOUNT

# Create a directory in the share.
echo "Creating $STORAGE_FILE_SHARE_DIR in $STORAGE_FILE_SHARE_NAME"
az storage directory create --share-name $STORAGE_FILE_SHARE_NAME --name $STORAGE_FILE_SHARE_DIR --account-name $STORAGE_ACCOUNT

# Create webapp config storage account
echo "Creating $STORAGE_ACCOUNT"
az webapp config storage-account add \
--resource-group $RG \
--name $FN_NAME \
--custom-id $FN_STORAGE_SHARE_ID \
--storage-type AzureFiles \
--share-name $STORAGE_FILE_SHARE_NAME \
--account-name $STORAGE_ACCOUNT \
--mount-path $FN_STORAGE_MOUNT_PATH \
--access-key $AZURE_STORAGE_KEY

# List webapp storage account
az webapp config storage-account list --resource-group $RG --name $FN_NAME

# Create tables to use in the app
az storage table create --account-name $STORAGE_ACCOUNT --account-key $AZURE_STORAGE_KEY --name users --fail-on-exist
az storage table create --account-name $STORAGE_ACCOUNT --account-key $AZURE_STORAGE_KEY --name messages --fail-on-exist