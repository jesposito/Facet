#!/usr/bin/env bash
set -euo pipefail

files=("$@")
if [ "${#files[@]}" -eq 0 ]; then
    files=(docker/Dockerfile docker/Dockerfile.dev)
fi

status=0

for file in "${files[@]}"; do
    previous_line=""
    line_no=0

    while IFS= read -r line || [ -n "$line" ]; do
        line_no=$((line_no + 1))

        if [[ "$line" =~ ^[[:space:]]*FROM[[:space:]]+([^[:space:]]+) ]]; then
            image="${BASH_REMATCH[1]}"
            if [[ "$image" == "scratch" || "$image" == *@sha256:* || "$previous_line" == *"supply-chain: allow-floating-base"* ]]; then
                previous_line="$line"
                continue
            fi

            echo "$file:$line_no: base image '$image' must be pinned with @sha256 or preceded by '# supply-chain: allow-floating-base ...'" >&2
            status=1
        fi

        previous_line="$line"
    done < "$file"
done

exit "$status"
