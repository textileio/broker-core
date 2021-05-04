#!/bin/bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "use $0 <target-url> <size>"
	exit -1
fi

TARGET="${1}/upload"
SIZE=$2
TOKEN="eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IjZURnVRRzFGTHZ4UGxPdGFVbllFQlRlU3haa09GZ3VSSGZwNlN1Q1ZDbG89IiwidXNlIjoic2lnIn19.eyJhdWQiOiJhYXJvbmJyb2tlciIsImlzcyI6ImNhcnNvbmZhcm1lci50ZXN0bmV0Iiwic3ViIjoiZGlkOmtleTp6Nk1rdjlZa25rMzZlUzhwY1pkZjgyWXhIcnBpWmJZZDFFYlNld0R2WEM3amhRRDciLCJuYmYiOjE2MjAwODY2NDMsImlhdCI6MTYyMDA4NjY0MywiZXhwIjozNjAwMDAwMDE2MjAwODY2NjB9.XcGW8z7HEVy6gZl2ZP0yGPyetlcXal8d86_YKvIor8vFQWYS9zSu4vxYmKutmsVkVu2gsopkdF3hsw0_qjCLDQ=="

echo $TARGET
echo "Generating random file..."
TMPFILE=$(mktemp)
head -c ${SIZE} < /dev/urandom > $TMPFILE
echo "Uploading file..."
curl -v -H "Authorization: $TOKEN" -F "region=europe" -F "file=@$TMPFILE" $TARGET
echo "Cleaning..."
rm $TMPFILE
