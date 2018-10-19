#!/bin/bash

#-------------------------------------------------------------
# IBM Confidential
# OCO Source Materials
# (C) Copyright IBM Corp. 2016
# The source code for this program is not published or
# otherwise divested of its trade secrets, irrespective of
# what has been deposited with the U.S. Copyright Office.
#-------------------------------------------------------------

# Set or update a status node.

# Expected inputs:
#   $1 arg: the ZNode path.
#   $2 arg: the value to record in the ZNode.
#   $3 arg (optional): if value is "create" then create the ZNode.

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPTDIR/utility.sh"

function recordStatusInEtcd {

	ZNODE_PATH=$1
	STATUS_STRING=${2:-""}

	# Append timestamp to node path, as we are storing the history of nodes
	# NOTE: The command below only provides the desired nanosecond precision on some systems
	#       (including Ubuntu), but on other systems (e.g., Alpine) it only has seconds precision.
	nano_time=$(date "+%s%N")
	ZNODE_PATH=$ZNODE_PATH/$nano_time

	STATUS_VALUE=$STATUS_STRING
	# STATUS_VALUE could be a JSON string like {"status":"FAILED",...}
	if [[ "$STATUS_VALUE" == "{"* ]]; then
		STATUS_VALUE=$( echo $STATUS_VALUE | sed -e 's/{.*"status"\s*:\s*"\([^"]*\)".*}/\1/g' )
	fi

	# Update the ZNode value using with_backoff logic and finite tries for intermediate steps but with infinite retries for final steps
	case "${STATUS_VALUE}" in
	    (COMPLETED)
	        infinite_exp_backoff runEtcdCommand put ${ZNODE_PATH} "${STATUS_STRING}"
	        ;;
	    (FAILED)
	        updateMetricsOnTrainingFailure "FAILED" &
	        infinite_exp_backoff runEtcdCommand put ${ZNODE_PATH} "${STATUS_STRING}"
	        ;;
	    (HALTED)
	        updateMetricsOnTrainingFailure "HALTED" &
	        infinite_exp_backoff runEtcdCommand put ${ZNODE_PATH} "${STATUS_STRING}"
	        ;;
	    (*)
	        with_backoff runEtcdCommand put ${ZNODE_PATH} "${STATUS_STRING}"

	esac

}
