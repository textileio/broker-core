#!/bin/bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "use $0 <target-url> <size>"
	exit -1
fi

TARGET="${1}/upload"
SIZE=$2
TOKEN="eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IjZURnVRRzFGTHZ4UGxPdGFVbllFQlRlU3haa09GZ3VSSGZwNlN1Q1ZDbG89IiwidXNlIjoic2lnIn19.eyJhdWQiOiJhYXJvbmJyb2tlciIsImlzcyI6ImNhcnNvbmZhcm1lci50ZXN0bmV0Iiwic3ViIjoiZGlkOmtleTp6Nk1rdjlZa25rMzZlUzhwY1pkZjgyWXhIcnBpWmJZZDFFYlNld0R2WEM3amhRRDciLCJuYmYiOjE2MjAwODIwMzksImlhdCI6MTYyMDA4MjAzOSwiZXhwIjoxNjIwMDg1NjM5fQ==.iaRq6Gee5YcNWRgkkU-E1GXO8DxpU86faCdAVWXZeCT2z4V9kLiO7tepuGLBMG_xB7r4ho1MrthpwX-wRsZ-Cg=="

echo $TARGET
echo "Generating random file..."
TMPFILE=$(mktemp)
head -c ${SIZE} < /dev/urandom > $TMPFILE
echo "Uploading file..."
curl -v -H "Authorization: $TOKEN" -F "region=europe" -F "file=@$TMPFILE" $TARGET
echo "Cleaning..."
rm $TMPFILE
