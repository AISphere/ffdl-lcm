#!/bin/bash

#-------------------------------------------------------------
# IBM Confidential
# OCO Source Materials
# (C) Copyright IBM Corp. 2016
# The source code for this program is not published or
# otherwise divested of its trade secrets, irrespective of
# what has been deposited with the U.S. Copyright Office.
#-------------------------------------------------------------

PUSHGATEWAY_HOST="pushgateway"
PUSHGATEWAY_UDP_PORT="9125"

# Retries a command a with backoff.
#
# The retry count is given by ATTEMPTS (default 5), the
# initial backoff timeout is given by TIMEOUT in seconds
# (default 1.)
#
# Successive backoffs double the timeout.
#
# Beware of set -e killing your whole script!
function with_backoff {
  local max_attempts=${ATTEMPTS-5}
  local timeout=${TIMEOUT-1}
  local attempt=0
  local exitCode=0

  while [[ $attempt < $max_attempts ]]
  do
    "$@"
    exitCode=$?

    if [[ $exitCode == 0 ]]
    then
      break
    elif [[ $exitCode == 2 ]]
    then
       attempt=0
       timeout=${TIMEOUT-1}
    fi

    echo "Failure! Retrying in $timeout.." 1>&2
    updateMetricsOnFailure $attempt &
    sleep $timeout
    attempt=$(( attempt + 1 ))
    timeout=$(( timeout * 2 ))
  done

  if [[ $exitCode != 0 ]]
  then
    echo "You've failed me for the last time! ($@)" 1>&2
  fi

  return $exitCode
}

# Exit the program immediately.
function panic {
    echo "Exiting with panic"
    exit 1
}


function updateMetricsOnFailure() {
  counter=$1
  metrics="databroker.s3.failures.$counter:1|c"
  echo "got metrics to push as $metrics"
  pushMetrics $metrics  
}

function pushMetrics() {
  metrics=$1
  # Setup UDP socket with statsd server
  exec 3<> /dev/udp/$PUSHGATEWAY_HOST/$PUSHGATEWAY_UDP_PORT
  # Send data
  printf "$metrics" >&3
  # Close UDP socket
  exec 3<&-
  exec 3>&-
}