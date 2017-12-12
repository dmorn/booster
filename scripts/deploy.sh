#!/bin/bash
# deploy.sh used to deploy this application to
# a target raspberry pi

set -e

. io.sh

# First check that we have a target
target=$2
user=$1
dbase="gostore-app"

if [ -z $user ]; then
  error "User required. example \"root\""
  exit -1
fi

if [ -z $target ]; then
  error "Target required. example 127.0.0.2"
  exit -1
fi

info "User $user"
info "Target $target"

info "Compiling app"
app="api-$( git rev-parse --verify HEAD )"

# Rasp
# env GOOS=linux GOARCH=arm GOARM=6 go build -o $app api.go

# Linux digitalocean
env GOOS=linux GOARCH=amd64 go build -o $app ../api.go

info "Sending files to $target"
scp ../.env $user@$target:./$dbase/server/
scp configure.sh $user@$target:./$dbase/
scp $app $user@$target:./$dbase/server/
scp ../conf/gostore-app.conf root@$target:/etc/supervisor/conf.d/
scp ../conf/redis.conf root@$target:/etc/

info "Cleaning up"
rm $app

info "Done"
exit 0
