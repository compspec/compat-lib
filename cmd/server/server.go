package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/compspec/compat-lib/pkg/server"
)

const (
	serverName = "compatServer"
)

var (
	host string
)

func main() {
	flag.StringVar(&host, "host", ":50051", "Server address (host:port)")
	flag.Parse()

	s := server.NewServer(serverName)
	log.Printf("🧩 starting compatibility server: %s", s.String())
	if err := s.Start(context.Background(), host); err != nil {
		fmt.Println(err)
		log.Fatal("error while running compatibility server")
	}
	log.Printf("🧩 done 🧩")
}
