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
}

function release {
	echo "Starting release pipeline..."
	if [ ! -f $conf ]; then
		echo file $conf does not exits!
		exit -1
	fi

	echo "Please insert git tag to be used for the release: "
	read version

	echo "Proceding will remove dist/ folder & will add a new release/tag $version. Continue?"
	echo "Yes/no"

	read opt
	if [ "$opt" = "no" ]; then
		echo "quitting..."
		return 1
	elif [ "$opt" != "$Yes" ]; then
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
