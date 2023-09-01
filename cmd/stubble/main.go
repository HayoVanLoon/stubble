package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/HayoVanLoon/stubble"
)

func parseArgs() (files []string) {
	for i := 1; i < len(os.Args); i += 1 {
		files = append(files, os.Args[i])
	}
	return files
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	files := parseArgs()

	h, err := stubble.FromFiles(files...)
	if err != nil {
		log.Fatal(err.Error())
	}
	s := &http.Server{
		Handler:           h,
		ReadTimeout:       60 * time.Second,
		ReadHeaderTimeout: 1 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	l, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Printf("Stubble server listening on port %s", port)
	if err = s.Serve(l); err != nil {
		log.Printf(err.Error())
	}
	log.Printf("Shutting down server")

	err = h.Close()
	if err != nil {
		log.Fatal(err)
	}
}
