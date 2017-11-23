package main

import (
	"flag"
	"log"
	"os"

	"github.com/danielmorandini/booster-network/node"
)

var (
	port = flag.Int("port", 1080, "PROXY listening port")
)

func main() {
	flag.Parse()

	b := node.NewBooster()
	b.Logger = log.New(os.Stdout, "BOOSTER ", log.LstdFlags)

	go func() {
		log.Fatal(b.Proxy.ListenAndServe(*port))
	}()
	log.Fatal(b.ListenAndServe())
}
