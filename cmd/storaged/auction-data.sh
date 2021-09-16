#!/bin/bash
set -euo pipefail

if [ "$#" -le 1 ]; then
	echo "use $0 <api-url> <auth-token> <car-url> <payload-cid> <piece-cid> <piece-size> <rep-factor:opt> <deadline:opt> <peerid,auth-token,wallet_addr:opt>"
	exit -1
fi

API_URL=$1/auction-data
AUTH_TOKEN=$2
CAR_URL=$3
PAYLOAD_CID=$4
PIECE_CID=$5
PIECE_SIZE=$6
REP_FACTOR=${7:-1}
DEADLINE=${8:-$(date --date="(date --rfc-3339=seconds) + 10 days" --rfc-3339=second | sed 's/ /T/g')}
REMOTE_WALLET=$9

echo "Creating storage-request with $CAR_URL [$PAYLOAD_CID, $PIECE_CID, $PIECE_SIZE bytes] with rep-factor $REP_FACTOR and deadline $DEADLINE..."

RW_JSON=""
if [ ! -z "$REMOTE_WALLET" ]; then 
	PARAMS=($(echo "$REMOTE_WALLET" | tr ',' '\n'))
	echo "Using remote wallet with peer-id ${PARAMS[0]} and wallet addr ${PARAMS[2]}"

	RW_TEMPLATE=',"remoteWallet":{"peerID":"%s","authToken":"%s","walletAddr":"%s"}'
	RW_JSON=$(printf "$RW_TEMPLATE" "${PARAMS[0]}" "${PARAMS[1]}" "${PARAMS[2]}")
fi

JSON_TEMPLATE='{"payloadCid":"%s","pieceCid":"%s","pieceSize":%s, "repFactor":%s, "deadline":"%s", "carURL":{"url":"%s"} %s}\n'
BODY=$(printf "$JSON_TEMPLATE" "$PAYLOAD_CID" "$PIECE_CID" "$PIECE_SIZE" "$REP_FACTOR" "$DEADLINE" "$CAR_URL" "$RW_JSON")

echo $BODY

curl -H "Authorization: Bearer $AUTH_TOKEN" -d "$BODY" $API_URL
