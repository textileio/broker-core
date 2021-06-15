#!/bin/bash
set -euo pipefail

checkEnv() {
    if [[ -z ${!1+set} ]]; then
       echo "Oop! Need to define the $1 environment variable"
       exit 1
    fi
}
checkEnv SEED_PHRASE

lockFunds() {
  echo "Locking funds on NEAR..."
  near call filecoin-bridge.testnet addDeposit '{ "brokerId": "filecoin-bridge.testnet", "accountId": "lock-box.testnet" }' \
  --account-id "lock-box.testnet" \
  --amount 1 \
  --seedPhrase "$SEED_PHRASE"
}

if [ "$#" -le 1 ]; then
	echo "use $0 <target-url> <min-size> <max-size:opt> <count:opt> <sleep:opt>"
	exit -1
fi

TARGET="${1}/upload"
MIN_SIZE=$2
MAX_SIZE=${3:-$MIN_SIZE}
COUNT=${4:-1}
# TODO: rewrite all this as a node program and use the js client.
TOKEN="Bearer eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6Iko5ZHRFeFpnOUhlWlNyRFlqZ2JSWXNNQnZKYjZHTEVHaF9nUU5PZi0zY289IiwidXNlIjoic2lnIn19.eyJhdWQiOiJmaWxlY29pbi1icmlkZ2UudGVzdG5ldCIsImlzcyI6ImxvY2stYm94LnRlc3RuZXQiLCJzdWIiOiJkaWQ6a2V5Ono2TWtoOG5VQnpudHdodEF6SEFtcGtSQkw5TEd1Zzd2blQ3UGZZalhxRVprMW9MTSIsIm5iZiI6MTYyMzc5NjY2MiwiaWF0IjoxNjIzNzk2NjYyLCJleHAiOjE5MzkxNTY2NjJ9.F47Ogmwkr3k9cafbSRb_tLD25KmnJrOhxSNQ6bKGua9zbwo0TnT0R9VNUpVfDqqdGhzzA7gKCAWuj78tr50oAQ=="
SLEEP=${5:-0}

echo "Hitting $TARGET with sizes [$MIN_SIZE, $MAX_SIZE] for $COUNT times..."

lockFunds

i=1
while ((i<=$COUNT)); do
  SIZE=$(($RANDOM*($MAX_SIZE-$MIN_SIZE+1)/32767 + $MIN_SIZE))
  TMPFILE=$(mktemp)
  head -c ${SIZE} < /dev/urandom > $TMPFILE

  echo "Uploading file of size $SIZE..."
  OUT=$(curl -H "Authorization: $TOKEN" -F "region=europe" -F "file=@$TMPFILE" $TARGET 2>/dev/null)
  if [[ $OUT == *"account doesn't have deposited funds"* ]]; then
	  lockFunds
  fi

  rm $TMPFILE
  let i++
  sleep $SLEEP
done;
