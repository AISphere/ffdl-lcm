#!/bin/bash

#-------------------------------------------------------------
# IBM Confidential
# OCO Source Materials
# (C) Copyright IBM Corp. 2016
# The source code for this program is not published or
# otherwise divested of its trade secrets, irrespective of
# what has been deposited with the U.S. Copyright Office.
#-------------------------------------------------------------

# Upload files from $DATA_DIR to Object Storage.

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPTDIR/utility.sh"

trap panic ERR # exit immediately on error

# Validate input.
: "${DATA_DIR:?Need to set DATA_DIR to non-empty value}"
: "${DATA_STORE_BUCKET:?Need to set DATA_STORE_BUCKET to non-empty value}"
: "${DATA_STORE_USERNAME:?Need to set DATA_STORE_USERNAME to non-empty value}"
: "${DATA_STORE_PASSWORD:?Need to set DATA_STORE_PASSWORD to non-empty value}"
: "${DATA_STORE_AUTHURL:?Need to set DATA_STORE_AUTHURL to non-empty value}"

# For S3 Object Storage
export AWS_ACCESS_KEY_ID=$DATA_STORE_USERNAME
export AWS_SECRET_ACCESS_KEY=$DATA_STORE_PASSWORD

env |sort

echo Using Object Storage account $DATA_STORE_USERNAME at $DATA_STORE_AUTHURL

# Upload data.
echo Upload start: $(date)
echo "Uploading from $DATA_DIR to bucket $DATA_STORE_BUCKET"
mkdir -p "$DATA_DIR"
time with_backoff aws --endpoint-url=$DATA_STORE_AUTHURL s3 sync "$DATA_DIR" "s3://$DATA_STORE_BUCKET"
echo Upload end: $(date)
