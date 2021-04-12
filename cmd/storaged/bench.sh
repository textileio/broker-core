#!/bin/bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "use $0 <target-url> <size-mb>"
	exit -1
fi

TARGET="${1}/upload"
SIZE=$2

echo $TARGET
echo "Generating random file..."
head -c ${SIZE} < /dev/urandom > foo.bin
echo "Uploading file..."
curl -v -F "region=europe" -F "file=@$(pwd)/foo.bin" $TARGET
echo "Cleaning..."
rm foo.bin


