#!/bin/bash
# deploy.sh used to deploy this application to
# a target raspberry pi

set -e

. io.sh

# First check that we have a target
target=$2
user=$1

if [ -z $user ]; then
  error "user required. example \"root\""
  exit -1
fi

if [ -z $target ]; then
  error "target file required."
  exit -1
fi

info "user $user"
info "target $target"

info "compiling app"
app="booster_arm-$( git rev-parse --verify HEAD )"

# Rasp
env GOOS=linux GOARCH=arm GOARM=6 go build -o $app ../cmd/node/main.go

info "sending files to $target"

info "cleaning up"
rm $app

info "done"
exit 0
