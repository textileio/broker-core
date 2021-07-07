#!/bin/bash
set -euo pipefail

# TODO: rewrite all this as a node program and use the js client.

TARGET="${1}/upload"
CONTRACT_SUFFIX="-edge"
TOKEN="Bearer eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IlJlcTBMNHBXc21kankyZGc3bExEajJTZWlmcm92WWk0eHRiQkRxZlRwY1U9IiwidXNlIjoic2lnIn19.eyJpc3MiOiJsb2NrLWJveC50ZXN0bmV0Iiwic3ViIjoiZGlkOmtleTp6Nk1rakFCa1N2QWdqTUJtTXozazJheXNDQUJ1TXhLU0ZkQWF6RHh4OFZOYTFnQ1UiLCJuYmYiOjE2MjUxNjAxNjMsImlhdCI6MTYyNTE2MDE2MywiZXhwIjoxOTQwNTIwMTYzLCJhdWQiOiJmaWxlY29pbi1icmlkZ2UtZWRnZS50ZXN0bmV0In0=.4KuYDrpw8WAEDbrrkNF3a3QRvRZVfcdjLMGyKyrfnkVEN4U4hYrXf1Z56KOxOhShd4g0gXVDVYh-NzYEDaBgCA=="
if [[ "$1" == *"staging"* ]]; then
    CONTRACT_SUFFIX=""
    TOKEN="Bearer eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6Iko5ZHRFeFpnOUhlWlNyRFlqZ2JSWXNNQnZKYjZHTEVHaF9nUU5PZi0zY289IiwidXNlIjoic2lnIn19.eyJhdWQiOiJmaWxlY29pbi1icmlkZ2UudGVzdG5ldCIsImlzcyI6ImxvY2stYm94LnRlc3RuZXQiLCJzdWIiOiJkaWQ6a2V5Ono2TWtoOG5VQnpudHdodEF6SEFtcGtSQkw5TEd1Zzd2blQ3UGZZalhxRVprMW9MTSIsIm5iZiI6MTYyMzc5NjY2MiwiaWF0IjoxNjIzNzk2NjYyLCJleHAiOjE5MzkxNTY2NjJ9.F47Ogmwkr3k9cafbSRb_tLD25KmnJrOhxSNQ6bKGua9zbwo0TnT0R9VNUpVfDqqdGhzzA7gKCAWuj78tr50oAQ=="
fi

checkEnv() {
    if [[ -z ${!1+set} ]]; then
       echo "Oop! Need to define the $1 environment variable"
       exit 1
    fi
}
checkEnv SEED_PHRASE

lockFunds() {
  if [[ "$1" == *"127.0.0.1"* ]]; then
    return
  fi

  echo "Locking funds on NEAR..."
  near call filecoin-bridge${CONTRACT_SUFFIX}.testnet addDeposit "{ \"brokerId\": \"filecoin-bridge${CONTRACT_SUFFIX}.testnet\", \"accountId\": \"lock-box.testnet\" }" \
  --account-id "lock-box.testnet" \
  --amount 0.25  \
  --seedPhrase "$SEED_PHRASE"
}

if [ "$#" -le 1 ]; then
	echo "use $0 <target-url> <min-size> <max-size:opt> <count:opt> <sleep:opt>"
	exit -1
fi

MIN_SIZE=$2
MAX_SIZE=${3:-$MIN_SIZE}
COUNT=${4:-1}
SLEEP=${5:-0}

echo "Hitting $TARGET with sizes [$MIN_SIZE, $MAX_SIZE] for $COUNT times..."

lockFunds $TARGET

i=1
while ((i<=$COUNT)); do
  SIZE=$(($RANDOM*($MAX_SIZE-$MIN_SIZE+1)/32767 + $MIN_SIZE))
  TMPFILE=$(mktemp)
  head -c ${SIZE} < /dev/urandom > $TMPFILE

  echo "Uploading file of size $SIZE..."
  OUT=$(curl -H "Authorization: $TOKEN" -F "file=@$TMPFILE" $TARGET 2>/dev/null)
  echo $OUT
  if [[ $OUT == *"account doesn't have deposited funds"* ]]; then
	  lockFunds $TARGET
  fi

  rm $TMPFILE
  let i++
  sleep $SLEEP
done;
