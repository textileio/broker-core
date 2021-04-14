#!/bin/bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "use $0 <target-url> <size>"
	exit -1
fi

TARGET="${1}/upload"
SIZE=$2

echo $TARGET
echo "Generating random file..."
TMPFILE=$(mktemp)
head -c ${SIZE} < /dev/urandom > $TMPFILE
echo "Uploading file..."
curl -v -H "Authorization: something" -F "region=europe" -F "file=@$TMPFILE" $TARGET
echo "Cleaning..."
rm $TMPFILE


