// Copyright (c) 2024 Gang Chen
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	mcastIface string
)

func main() {
	var httpAddr string
	var httpPort int
	flag.StringVar(&mcastIface, "m", "", "multicast interface")
	flag.StringVar(&httpAddr, "a", "127.0.0.1", "HTTP listen address")
	flag.IntVar(&httpPort, "p", 4022, "HTTP listen port")
	flag.Parse()

	if mcastIface == "" {
		log.Println("Missing required parameter: -m <interface>")
		flag.Usage()
		return
	}

	if httpPort <= 0 {
		log.Println("Missing or invalid required parameter: -p <port>")
		flag.Usage()
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGQUIT)

	ctx := context.Background()
	go func() {
		<-sigCh // receive signal
		webServer.Shutdown(ctx)
	}()

	runHttpServer(fmt.Sprintf("%s:%d", httpAddr, httpPort))
}
