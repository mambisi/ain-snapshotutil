echo "Syncing to block height: $STOP_BLOCK"

block=0
attempts=0

./defid -daemon -stop-block="$STOP_BLOCK"

while [ $block -lt "$STOP_BLOCK" ]; do
  sleep 1
  if [ $attempts -gt 1200 ]; then
    echo "Node stuck for more than 20 minutes"
    exit 1
  fi
  h=$(./defi-cli getblockcount)
  b=$${h:-$block}
  if [ "$block" -eq "$b" ]; then
    attempts=$((attempts + 1))
    else
    attempts=0
  fi
  block=$${b:-$block}
  echo "===> Block Height $block"
done

./defi-cli stop