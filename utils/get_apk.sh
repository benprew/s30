#!/bin/bash

set -x

ID=$(gh run list --workflow='Android build' |head -n1 |cut -f 7)
gh run download "$ID"
