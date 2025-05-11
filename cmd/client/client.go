package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/compspec/compat-lib/pkg/client"
)

var (
	host string
)

func main() {
	flag.StringVar(&host, "host", ":50051", "Server address (host:port)")
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		log.Fatal("Please provide a compatibility artifact to compare with the host.")
	}
	client, err := client.NewClient(host)
	if err != nil {
		fmt.Println(err)
		log.Fatal("Issue creating client")
	}
	client.CheckCompatibility(context.Background(), args[0])
}
