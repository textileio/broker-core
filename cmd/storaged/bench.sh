#!/bin/bash
set -euo pipefail

if [ "$#" -ne 3 ]; then
	echo "use $0 <target-url> <size> <seedPhrase>"
	exit -1
fi

TARGET="${1}/upload"
SIZE=$2
TOKEN="eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IlYyZmNCUTJudHE3VDJ4UFpjVkVMVTFhUEstaGhHVTZzOENrZ2M1R3lSSVU9IiwidXNlIjoic2lnIn19.eyJhdWQiOiJsb2NrLWJveC50ZXN0bmV0IiwiaXNzIjoibG9jay1ib3gudGVzdG5ldCIsInN1YiI6ImRpZDprZXk6ejZNa2tMVE5NYzRoRVN1UlR5QVVRelBjajNIRnRZNjZkNjJWNjNMNW1ZN0pFdDRMIiwibmJmIjoxNjIwMzIwNzM2LCJpYXQiOjE2MjAzMjA3MzYsImV4cCI6MTAwMDAwMDAxNjIwMzIwNzQwfQ==.E4eLnR7sXvne-r3aV4XwjVhThmu85HSEoE83IpTF1vDp71zgO_DAbhOT4o0PGpfTo-P6kXLKX1ixdZ6fgmMEBA=="
SEED=$3

echo $TARGET
echo "Locking funds..."
near call lock-box.testnet lockFunds '{ "brokerId": "lock-box.testnet", "accountId": "lock-box.testnet" }' --account-id "lock-box.testnet" --amount 1 --seedPhrase "$SEED"
echo "Generating random file..."
TMPFILE=$(mktemp)
head -c ${SIZE} < /dev/urandom > $TMPFILE
echo "Uploading file..."
curl -v -H "Authorization: $TOKEN" -F "region=europe" -F "file=@$TMPFILE" $TARGET
echo "Cleaning..."
rm $TMPFILE
