#!/bin/bash
echo "Syncing to block height: $STOP_BLOCK"
STOP_BLOCK_P=$((STOP_BLOCK + 5))
block=0
attempts=0

./defid -daemon -stop-block="$STOP_BLOCK_P"
sleep 30
while [ $block -lt "$STOP_BLOCK" ]; do
  sleep 1
  if [ $attempts -gt 1200 ]; then
    echo "Node Stuck After $attempts Recovery Attempt"
    exit 1
  fi
  h=$(./defi-cli getblockcount)
  b=${h:-$block}
  if [ "$block" -eq "$b" ]; then
    attempts=$((attempts + 1))
    # revive defid
    echo "===> Attempt[$attempts/1200] to revive Defid"
    ./defid -daemon -stop-block="$STOP_BLOCK_P"
    sleep 30
    else
    attempts=0
  fi
  block=${b:-$block}
  echo "===> Block Height [$block/$STOP_BLOCK]"
done

./defi-cli stop