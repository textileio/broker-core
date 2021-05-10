#!/bin/bash
set -euo pipefail

checkEnv() {
    if [[ -z ${!1+set} ]]; then
       echo "Oop! Need to define the $1 environment variable"
       exit 1
    fi
}
checkEnv SEED_PHRASE

if [ "$#" -le 1 ]; then
	echo "use $0 <target-url> <min-size> <max-size:opt> <count:opt> <sleep:opt>"
	exit -1
fi

TARGET="${1}/upload"
MIN_SIZE=$2
MAX_SIZE=${3:-$MIN_SIZE}
COUNT=${4:-1}
TOKEN="Bearer eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IlYyZmNCUTJudHE3VDJ4UFpjVkVMVTFhUEstaGhHVTZzOENrZ2M1R3lSSVU9IiwidXNlIjoic2lnIn19.eyJhdWQiOiJsb2NrLWJveC50ZXN0bmV0IiwiaXNzIjoibG9jay1ib3gudGVzdG5ldCIsInN1YiI6ImRpZDprZXk6ejZNa2tMVE5NYzRoRVN1UlR5QVVRelBjajNIRnRZNjZkNjJWNjNMNW1ZN0pFdDRMIiwibmJmIjoxNjIwMzIwNzM2LCJpYXQiOjE2MjAzMjA3MzYsImV4cCI6MTAwMDAwMDAxNjIwMzIwNzQwfQ==.E4eLnR7sXvne-r3aV4XwjVhThmu85HSEoE83IpTF1vDp71zgO_DAbhOT4o0PGpfTo-P6kXLKX1ixdZ6fgmMEBA=="
SLEEP=${5:-0}

echo "Hitting $TARGET with sizes [$MIN_SIZE, $MAX_SIZE] for $COUNT times..."

echo "Locking funds on NEAR..."
near call lock-box.testnet lockFunds '{ "brokerId": "lock-box.testnet", "accountId": "lock-box.testnet" }' \
--account-id "lock-box.testnet" \
--amount 1 \
--seedPhrase "$SEED_PHRASE"

i=1
while ((i<=$COUNT)); do
  SIZE=$(($RANDOM*($MAX_SIZE-$MIN_SIZE+1)/32767 + $MIN_SIZE))
  TMPFILE=$(mktemp)
  head -c ${SIZE} < /dev/urandom > $TMPFILE

  echo "Uploading file of size $SIZE..."
  curl -H "Authorization: $TOKEN" -F "region=europe" -F "file=@$TMPFILE" $TARGET

  rm $TMPFILE
  let i++
  sleep $SLEEP
done;
