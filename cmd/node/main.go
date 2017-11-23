package main

import (
	"flag"
	"log"
	"os"

	"github.com/danielmorandini/booster-network/node"
)

var (
	pport = flag.Int("pport", 1080, "PROXY listening port")
	bport = flag.Int("bport", 4884, "BOOSTER listening port")
)

func main() {
	flag.Parse()

	b := node.NewBooster()
	b.Logger = log.New(os.Stdout, "BOOSTER ", log.LstdFlags)

	go func() {
		log.Fatal(b.Proxy.ListenAndServe(*pport))
	}()
	log.Fatal(b.ListenAndServe(*bport))
}
