# booster-network
[![Build Status](https://travis-ci.org/danielmorandini/booster-network.svg?branch=master)](https://travis-ci.org/danielmorandini/booster-network)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/danielmorandini/booster-network/node)

## Abstract

At my parent's home we have a slow Internet connection. I noticed that we also
have four smartphones, all of them 4G enabled. This made me think about this
configuration a bit. We have four fast network interfaces, all of them with a
limited Internet usage per month, and a slow Internet connection.

Even if in need, we always use just one device at once, and usually it is the
slowest one.

This is were **booster-network** comes into play. Its aim is to create a network
of peer nodes, each of them with an active Internet connection, that balance the
network usage between their interfaces.

Each booster-node is actually composed by a transparent forward proxy and a
booster instance. Both parts will be described in detail later on, but, in
summary, the former one is a SOCKS5 proxy that keeps track of the number of open
channels of data that is proxing. When we want to enable booster on a device, we
just have to make it exploit the booster-proxy when networking, and use the
booster instance to connect nodes to it, and manage them. This way we can use
our device as always, while transparently in the background **booster-network**
will balance the network usage between each node, with all the consequences that
come along with it.

**My current configuration:** I have a booster instance on my mac always
running, using the advanced settings in the network section of system
preferences, I set the proxy setting to use the SOCKS5 booster-proxy, so all my
network traffic passes through it. I have another booster instance running on a
Nexus 5X (thanks to termux), which is connected to my mac's (step described
later). With just this simple steps, my network traffic is balanced between our
slow home's Internet connection and the NEXUS 5's 4G!

When I need a further boost, I even connect my brother's phone, my mum's, and
the phone of whoever wants to **boost** me! :tada:

## Usage
`booster help`
```
Usage:
  booster [command]

Available Commands:
  connect     connect two nodes together
  disconnect  disconnect two nodes
  help        Help about any command
  inspect     inspects the node's activity
  start       starts a booster node
  version     prints booster version

Flags:
  -h, --help   help for booster

Use "booster [command] --help" for more information about a command.
```

`booster help version`
```
prints booster version

Usage:
  booster version [flags]

Flags:
  -h, --help   help for version
```

`booster help start`
```
starts a booster proxy and node. Both are tcp servers, their listening port will be logged

Usage:
  booster start [flags]

Flags:
      --bport int   booster listening port (default 4884)
  -h, --help        help for start
      --pport int   proxy listening port (default 1080)
```

`booster help connect`
```
connect asks (by default) the local node to perform the necessary steps required to connect
an external node to itself. Returns the added node identifier if successfull. You can use
the 'inspect' command to monitor node activity.

Usage:
  booster connect [host:port] [flags]

Flags:
  -b, --baddr string   booster address (default ":4884")
  -h, --help           help for connect
```

`booster help disconnect`
```
disconnect aks (by default) the local node to perform the necessary steps required to disconnect
completely a node from itself.

Usage:
  booster disconnect [id] [flags]

Flags:
  -b, --baddr string   booster address (default ":4884")
  -h, --help           help for disconnect
```

`booster help inspect`
```
inspect listents (by default) on the local node for each node activity update, and logs it.

Usage:
  booster inspect [flags]

Flags:
  -b, --baddr string   booster address (default ":4884")
  -h, --help           help for inspect
```

## Booster protocol specification (todo)

## Connecting through bluetooth (todo)
