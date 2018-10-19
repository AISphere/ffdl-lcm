#!/bin/bash

#-------------------------------------------------------------
# IBM Confidential
# OCO Source Materials
# (C) Copyright IBM Corp. 2016
# The source code for this program is not published or
# otherwise divested of its trade secrets, irrespective of
# what has been deposited with the U.S. Copyright Office.
#-------------------------------------------------------------

# Download model from S3 Object Storage to $DATA_DIR.

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPTDIR/utility.sh"

trap panic ERR # exit immediately on error

# Validate input.
: "${DATA_DIR?Need to set DATA_DIR}"
: "${DATA_STORE_OBJECT:?Need to set DATA_STORE_OBJECT to non-empty value}"
: "${DATA_STORE_USERNAME:?Need to set DATA_STORE_USERNAME to non-empty value}"
: "${DATA_STORE_PASSWORD:?Need to set DATA_STORE_PASSWORD to non-empty value}"
: "${DATA_STORE_AUTHURL:?Need to set DATA_STORE_AUTHURL to non-empty value}"

# For S3 Object Storage
export AWS_ACCESS_KEY_ID=$DATA_STORE_USERNAME
export AWS_SECRET_ACCESS_KEY=$DATA_STORE_PASSWORD

echo Using Object Storage account $DATA_STORE_USERNAME at $DATA_STORE_AUTHURL

# Download data.
echo Download start: $(date)
echo "Downloading object $DATA_STORE_OBJECT to $DATA_DIR"
time with_backoff aws --endpoint-url=$DATA_STORE_AUTHURL s3 cp "s3://$DATA_STORE_OBJECT" /tmp/model.zip
mkdir -p "$DATA_DIR"
cd "$DATA_DIR"
unzip /tmp/model.zip
echo Download end: $(date)
chmod -R 777 .
