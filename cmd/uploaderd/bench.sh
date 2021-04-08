#!/bin/bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "use $0 <target-url> <size-mb>"
	exit -1
fi

TARGET=$1
SIZE=$2

echo $TARGET
curl -v -F region=europe -F upload=@<(head -c ${SIZE}M < /dev/urandom) $TARGET


