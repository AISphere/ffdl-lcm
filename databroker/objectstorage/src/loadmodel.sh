#!/bin/bash

#-------------------------------------------------------------
# IBM Confidential
# OCO Source Materials
# (C) Copyright IBM Corp. 2016
# The source code for this program is not published or
# otherwise divested of its trade secrets, irrespective of
# what has been deposited with the U.S. Copyright Office.
#-------------------------------------------------------------

# Download model from Object Storage to $DATA_DIR.

# Validate input.
: "${DATA_DIR?Need to set DATA_DIR}"
: "${DATA_STORE_OBJECT:?Need to set DATA_STORE_OBJECT to non-empty value}"
: "${DATA_STORE_USERNAME:?Need to set DATA_STORE_USERNAME to non-empty value}"
: "${DATA_STORE_PASSWORD:?Need to set DATA_STORE_PASSWORD to non-empty value}"
: "${DATA_STORE_AUTHURL:?Need to set DATA_STORE_AUTHURL to non-empty value}"

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPTDIR/utility.sh"

trap panic ERR # exit immediately on error

constructSwiftConnectionArgs
echo Connection args: "${SWIFT_CONNECTION_ARGS[@]}"

echo Using Object Storage account $DATA_STORE_USERNAME at $DATA_STORE_AUTHURL

bucket=$(echo "$DATA_STORE_OBJECT" |cut -d / -f 1)
object=$(echo "$DATA_STORE_OBJECT" |cut -d / -f 2-)

# Download data.
echo Download start: $(date)
echo "Downloading object $DATA_STORE_OBJECT to $DATA_DIR"
time with_backoff swift --verbose "${SWIFT_CONNECTION_ARGS[@]}" download -o /tmp/model.zip "$bucket" "$object"
mkdir -p "$DATA_DIR"
cd "$DATA_DIR"
unzip /tmp/model.zip
echo Download end: $(date)
chmod -R 777 .
