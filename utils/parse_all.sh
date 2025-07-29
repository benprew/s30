#!/bin/bash

cd tmp

find ../MtG_DotP_SotA -iname '*.pic' |xargs  -I{} /bin/bash -c 'echo {}; python3 ../pic2png.py {}'
