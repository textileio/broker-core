Dockfiles="$(find $1  -name 'Dockerfile')"
d=$(date +%s)
i=0
for file in $Dockfiles; do
i=$(( i + 1 ))
    echo "patching timestamp for $file"
    touch -d @$(( d + i )) "$file"
done
