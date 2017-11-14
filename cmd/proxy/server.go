package main

import (
	"flag"
	"log"
	"os"

	"github.com/danielmorandini/booster/proxy"
)

var (
	port = flag.Int("port", 1080, "PROXY listening port")
)

func main() {
	flag.Parse()

	proxy := proxy.NewProxyServer(*port)
	proxy.Logger = log.New(os.Stdout, "BOOSTER ", log.LstdFlags)

	proxy.Fatal(proxy.ListenAndServe(*port))
}
