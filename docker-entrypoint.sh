#!/bin/sh

set -e

# if the first argument look like a parameter (i.e. start with '-'), run enva as well
if [[ "${1#-}" != "$1" ]]; then
	set -- enva "$@"
fi

echo $@
exec "$@"