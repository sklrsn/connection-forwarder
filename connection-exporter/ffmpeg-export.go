package main

import (
	"log"
)

func init() {
	log.SetFlags(log.LUTC | log.Lshortfile)
}

func main() {
	log.Println("starting exporter")
}
