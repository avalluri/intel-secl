#!/bin/bash

SERVICE_NAME=cms
CURRENT_VERSION=v4.1.1
BACKUP_PATH=${BACKUP_PATH:-"/tmp/"}
LOG_FILE=${LOG_FILE:-"/tmp/$SERVICE_NAME-upgrade.log"}
echo "" > $LOG_FILE
./upgrade.sh -s $SERVICE_NAME -v $CURRENT_VERSION -b $BACKUP_PATH |& tee -a $LOG_FILE
exit ${PIPESTATUS[0]}
