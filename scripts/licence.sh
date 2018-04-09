#!/bin/bash

notice=`cat << EOF
/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
EOF
`
tmp=tt.txt

function applyNotice {
	local target=$1
	if [ ! -f $target ]; then
		echo file $target does not exits!
		return
	fi

	local sn="Copyright (C) 2018 Daniel Morandini"
	if `grep -q "$sn" "$target"` ; then
		echo file $target contains notice already!
		return
	fi

	echo appending copyright notice to $target...
	{ printf "$notice\n\n"; cat $target; } > $tmp && cat $tmp > $target
}

# find all .go files in the proj
files=( `find . -name "*.go" -type f -not -path "*vendor*"` )

# apply notice to each file
for f in "${files[@]}"; do
	printf "$f\n"
	applyNotice $f
done

# cleanup
if [ -f $tmp ]; then
	echo cleaning up...
	rm $tmp
fi

exit 0
