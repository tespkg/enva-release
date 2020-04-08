#!/bin/bash

# rename dir/file(accurate match) recursively under working current dir
# rename file content recursively under current working dir

function tree() {
    # this function have no params are required or used
    for file in $(ls);
    do
        if [[ -d "$file" ]]; then
            echo "+ $file"
            # change dir name to the replacement name
            if [[ "${file}" == "${old_name}" ]]; then
                mv ${file} ${replacement}
                file=${replacement}
            fi

            # visit dir recursively
            cd ${file}
            tree
            cd ..
        else
            echo "- $file"
            # change file name to the replacement name
            if [[ "${file}" == "${old_name}" ]]; then
                mv ${file} ${replacement}
                file=${replacement}
            fi
            # change the file content
            sed -i -e 's#'"${old_name}"'#'"${replacement}"''#g ${file}
        fi
    done
}

# rename.sh gotemplate <replacement name>
if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <old name> <replacement name>"
  exit 1
fi

old_name=$1
replacement=$2

echo "old name ${old_name}"
echo "replacement name ${replacement}"

tree
