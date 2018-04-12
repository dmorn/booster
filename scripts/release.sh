#
# MIT License

# Copyright (c) 2018 Daniel Morandini

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
# 
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
# 
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

#

#!/bin/bash

set -e

conf=.goreleaser.yml

function uploadSnaps {
	# allow multiple upload failures
	set +e
	echo "Uploading snaps..."
	local files=( `find dist -name "*.snap" -type f` )

	echo "Snaps found: ${files[@]}."
	for f in ${files[@]}; do
		snapcraft push $f
	done

	set -e

	echo "Remember to execute \`snapcraft release <snap name> revision channel\` for each revision provided!"
	echo "Revisions:"
	snapcraft list-revisions booster
}

function release {
	command -v goreleaser >/dev/null 2>&1 || { echo >&2 "goreleaser not installed. Quitting..."; exit 1; }

	echo "Starting release pipeline..."
	if [ ! -f $conf ]; then
		echo file $conf does not exits!
		exit -1
	fi

	echo "Please insert git tag to be used for the release: "
	read version

	echo "Proceding will remove dist/ folder & will add/substitute a new release/tag $version. Continue?"
	echo "Yes/no"

	read opt
	if [ "$opt" = "no" ]; then
		echo "quitting..."
		return 1
	elif [ "$opt" != "Yes" ]; then
		echo "no such option: $opt"
		return -1
	fi

	echo "Creating tag $version..."
	git tag -a "$version" -m "Release $version"
	git push origin "$version"

	echo "Executing goreleaser..."
	goreleaser release --rm-dist
}


# main

OPTS="Release Upload-snaps Exit"
select opt in $OPTS; do
	if [ "$opt" = "Release" ]; then
		release
	elif [ "$opt" = "Upload-snaps" ]; then
		uploadSnaps
	else
		exit
	fi
done

exit 0
