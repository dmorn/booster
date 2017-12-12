#!/bin/bash
# Some IO functions

function info {
  printf "\n==================================================\n"
  printf "INFO ---> $1"
  printf "\n==================================================\n"
}

function fatalerror {
  printf "ERR ---> $1\n"
  exit -1
}

function getEnv {
  cat ../.env | grep $1 | awk -F"=" '{ print $2 }'
}

function getTargets {

}
