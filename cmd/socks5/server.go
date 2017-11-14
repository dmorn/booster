package main

import (
	"flag"
	"log"
	"os"

	"github.com/danielmorandini/booster/socks5"
)

var (
	port = flag.Int("port", 1080, "SOCKS listening port")
)

func main() {
	flag.Parse()

	proxy := new(socks5.Socks5)
	logger := log.New(os.Stdout, "BOOSTER ", log.LstdFlags)
	proxy.Log = logger

	logger.Fatal(proxy.ListenAndServe(*port))
}
