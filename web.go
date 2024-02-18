// Copyright (c) 2024 Gang Chen
package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
)

var (
	urlPattern = regexp.MustCompile(`/?(?:udp|rtp)/([0-9.]+:[0-9]+)`)
	webServer  *http.Server
)

type webHandler struct {
}

func replyString(resp http.ResponseWriter, status int, format string, args ...any) {
	header := resp.Header()
	header.Add("Content-Type", "text/plain")

	resp.WriteHeader(status)
	fmt.Fprintf(resp, format, args...)
}

func (h *webHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	m := urlPattern.FindStringSubmatch(req.URL.Path)
	if m == nil {
		replyString(resp, http.StatusBadRequest, "Invalid request: %s\n", req.URL.Path)
		return
	}

	addr := m[1]

	tp, err := newTransport(mcastIface, addr)
	if err != nil {
		replyString(resp, http.StatusBadRequest, "failed initialize transport: %s", err)
		return
	}

	defer tp.Close()
	if ch, err := tp.start(req.Context()); err == nil {
		resp.WriteHeader(http.StatusOK)

		for pkt := range ch {
			if err = pkt.Write(resp); err == nil {
				tp.release(pkt)
			} else {
				log.Printf("error occurs while sending response: %s", err)
				break
			}
		}
	} else {
		replyString(resp, http.StatusInternalServerError, "failed to establish connection: %s", err)
	}
}

func runHttpServer(addr string) {
	log.Printf("Listen at %s\n", addr)
	webServer = &http.Server{Addr: addr, Handler: &webHandler{}}
	if err := webServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("Server closed with error: %s\n", err)
	}
}
