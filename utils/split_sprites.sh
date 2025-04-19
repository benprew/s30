#!/bin/bash

input_image=$1
sub_width=$2
sub_height=$3

# Get the original image dimensions
dimensions=$(identify -format "%wx%h" "$input_image")
original_width=$(echo "$dimensions" | cut -dx -f1)
original_height=$(echo "$dimensions" | cut -dx -f2)

# Calculate the number of sub-images in each direction
num_x=$((original_width / sub_width))
num_y=$((original_height / sub_height))

# Loop through the grid and crop
for ((i=0; i<num_y; i++)); do
  for ((j=0; j<num_x; j++)); do
    x_offset=$((j * sub_width))
    y_offset=$((i * sub_height))
    output_file="output_${i}_${j}.png"
    convert "$input_image" -crop "${sub_width}x${sub_height}+${x_offset}+${y_offset}" "$output_file"
  done
done
