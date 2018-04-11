# booster
[![Build Status](https://travis-ci.org/danielmorandini/booster.svg?branch=master)](https://travis-ci.org/danielmorandini/booster)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/danielmorandini/booster)
[![Go Report Card](https://goreportcard.com/badge/github.com/danielmorandini/booster)](https://goreportcard.com/report/github.com/danielmorandini/booster)

# Table of Contents

- [Overview](#overview)
- [Installation](#installation)
  * [From source](#from-source)
  * [Homebrew](#homebrew)
  * [Binary](#binary)
- [Usage](#usage)

## Overview

At my parent's home we have a slow Internet connection. I noticed that we also
have four smartphones, all of them 4G enabled. This made me think about this
configuration a bit. We have four fast network interfaces, all of them with a
limited Internet usage per month, and a slow Internet connection.

Even if in need, we always use just one device at once, and usually it is the
slowest one.

This is were **booster** comes into play. Its aim is to create a network
of peer nodes, each of them with an active Internet connection, that balance the
network usage between their interfaces.

Each booster-node is actually composed by a transparent forward proxy and a
booster instance. Both parts will be described in detail later on, but, in
summary, the former one is a SOCKS5 proxy that keeps track of the number of open
channels of data that is proxing. When we want to enable booster on a device, we
just have to make it exploit the booster-proxy when networking, and use the
booster instance to connect nodes to it, and manage them. This way we can use
our device as always, while transparently in the background **booster**
will balance the network usage between each node, with all the consequences that
come along with it.

**My current configuration:** I have a booster instance on my mac always
running, using the advanced settings in the network section of system
preferences, I set the proxy setting to use the SOCKS5 booster-proxy, so all my
network traffic passes through it. I have another booster instance running on a
Nexus 5X (thanks to termux), which is connected to my mac's (step described
later). With just this simple steps, my network traffic is balanced between our
slow home's Internet connection and the NEXUS 5X's 4G!

When I need a further boost, I even connect my brother's phone, my mum's, and
the phone of whoever wants to **boost** me! :tada:

## Installation
#### From source:
With a [correctly configured](https://golang.org/doc/code.html#GOPATH) Go installation,

get the repo with:
```
go get -u github.com/danielmorandini/booster/node
```

build the executable:
```
cd $GOPATH/src/github.com/danielmorandini/booster
go build -o booster cmd/booster/main.go
```

With Makefile (thanks to [Filippo's](https://github.com/FiloSottile) [hellogopher](https://github.com/cloudflare/hellogopher)),

clone the repo:
```
git clone https://github.com/danielmorandini/booster.git
```

test & build:
```
cd booster
make setup
make test
make
```

a `bin` folder will be produced inside the repo.

#### Homebrew:
Add tap (has only to be done once):
```
brew tap danielmorandini/tap
```
Install booster:
```
brew install danielmorandini/tap/booster
```

#### Binary:
Pick your [release!](https://github.com/danielmorandini/booster/releases)

## Usage
Booster comes with a great CLI powered by [cobra](https://github.com/spf13/cobra).

Find more with `bin/booster help`
