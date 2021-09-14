#!/bin/bash
set -euo pipefail

if [ "$#" -le 1 ]; then
	echo "use $0 <api-url> <auth-token> <car-url> <payload-cid> <piece-cid> <piece-size> <rep-factor:opt> <deadline:opt>"
	exit -1
fi

API_URL=$1/auction-data
AUTH_TOKEN=$2
CAR_URL=$3
PAYLOAD_CID=$4
PIECE_CID=$5
PIECE_SIZE=$6
REP_FACTOR=${7:-1}
DEADLINE=${8:-$(date --date="(date --rfc-3339=seconds) + 2 days" --rfc-3339=second | sed 's/ /T/g')}

echo "Creating storage-request with $CAR_URL [$PAYLOAD_CID, $PIECE_CID, $PIECE_SIZE bytes] with rep-factor $REP_FACTOR and deadline $DEADLINE..."

JSON_TEMPLATE='{"payloadCid":"%s","pieceCid":"%s","pieceSize":%s, "repFactor":%s, "deadline":"%s", "carURL":{"url":"%s"}}\n'
BODY=$(printf "$JSON_TEMPLATE" "$PAYLOAD_CID" "$PIECE_CID" "$PIECE_SIZE" "$REP_FACTOR" "$DEADLINE" "$CAR_URL")

echo $BODY

curl -H "Authorization: Bearer $AUTH_TOKEN" -d "$BODY" $API_URL
