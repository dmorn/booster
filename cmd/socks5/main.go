package main

import (
	"flag"

	"github.com/danielmorandini/booster-network/socks5"
)

var (
	port = flag.Int("port", 1080, "SOCKS listening port")
)

func main() {
	flag.Parse()

	proxy := socks5.SOCKS5()
	proxy.Fatal(proxy.ListenAndServe(*port))
}
