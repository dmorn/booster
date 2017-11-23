package main

import (
	"flag"
	"log"
	"os"

	"github.com/danielmorandini/booster-network/socks5"
)

var (
	port = flag.Int("port", 1080, "SOCKS listening port")
)

func main() {
	flag.Parse()

	proxy := new(socks5.Socks5)
	proxy.Logger = log.New(os.Stdout, "SOCKS5 ", log.LstdFlags)

	proxy.Fatal(proxy.ListenAndServe(*port))
}
