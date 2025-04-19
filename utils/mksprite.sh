#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage: $0 <output_file.png> <input_file1.png> [input_file2.png ...]"
  exit 1
fi

output_file="$1"
shift
input_files=("$@")

if [ ${#input_files[@]} -eq 0 ]; then
  echo "Error: No input PNG files provided."
  exit 1
fi

montage "${input_files[@]}" -tile 0x0 -geometry +0+0 "$output_file"
