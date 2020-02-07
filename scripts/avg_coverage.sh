#!/bin/bash

set -o errexit
shopt -s nullglob

cat .testPkg.txt | \
      grep "coverage: .*% of statements" | \
      awk '{print $5}' | \
      awk 'BEGIN{FS="%"}{c+=1; sum+=$1}END{avg = 0.0; if(c!=0) avg = sum/c; printf "avg coverage: %.2f%% of statements in total\n", avg}'
