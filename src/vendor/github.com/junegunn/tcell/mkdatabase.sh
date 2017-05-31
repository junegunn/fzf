#!/bin/ksh

# Copyright 2015 The TCell Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use file except in compliance with the License.
# You may obtain a copy of the license at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# This shell script builds the Go database, which is somewhat minimal for
# size reasons (it only contains the most commonly used entries), and
# then builds the complete JSON database.
#

terms=/tmp/terms.$$

# This script is not very efficient, but there isn't really a better way without
# writing code to decode the terminfo binary format directly.  Its not worth
# worrying about.

# now get the rest
all=`toe -a | cut -f1`
echo Scanning terminal definitions
echo > $terms
for f in $all; do
	infocmp $f | awk -v FS="|" -v OFS=" " '/^[^#	]/ { print $1; for (i = 2; i < NF; i++) print $i "=" $1; }' |sort >> $terms
	printf "."
done
echo

# make sure we have mkinfo
echo "Building mkinfo"
go build mkinfo.go

# first make the database.go file
echo "Building Go database"
./mkinfo -go database.go `cat models.txt aliases.txt`
go fmt database.go

echo "Building JSON database"

./mkinfo -nofatal -quiet -json database.json `cat $terms`
