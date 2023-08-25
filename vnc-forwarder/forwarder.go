package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/google/uuid"
)

func init() {
	log.SetFlags(log.LUTC | log.Llongfile)
	log.SetPrefix("=> ")
}

func main() {
	var lr net.Listener
	var err error
	lr, err = net.Listen("tcp", ":5900")
	if err != nil {
		log.Fatalf("forwarder: error occurred %v", err)
	}

	log.Printf("listener is running at %v", lr.Addr())

	for {
		srcConn, err := lr.Accept()
		if err != nil {
			log.Printf("error occurred %v", err)
			continue
		}
		targetConn, err := net.Dial("tcp", "vnc-server:5901")
		if err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			continue
		}
		go serve(srcConn, targetConn)
	}
}

func serve(srcConn, targetConn net.Conn) {
	defer func() {
		_ = srcConn.Close()
		_ = targetConn.Close()
	}()

	sr := &SessionRecorder{
		sessionID: fmt.Sprintf("%v", uuid.NewString()),
	}
	if err := os.MkdirAll(os.Getenv("STORAGE_LOCATION"), 0777); err != nil {
		log.Printf("%v", err)
		return
	}
	storage, err := os.Create(fmt.Sprintf("%v/%v", os.Getenv("STORAGE_LOCATION"), sr.sessionID))
	if err != nil {
		return
	}
	sr.storage = storage

	handleConnection(srcConn, targetConn, sr)
}

func handleConnection(srcConn, targetConn net.Conn, sr *SessionRecorder) (err error) {
	var wg sync.WaitGroup
	defer func() {
		_ = srcConn.Close()
		_ = targetConn.Close()
		_ = sr.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		n, err := io.Copy(io.MultiWriter(sr, targetConn), srcConn)
		if err != nil {
			log.Printf("forward: connection error %v", err)
			return
		}
		log.Printf("forward: wrote %d bytes", n)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		n, err := io.Copy(io.MultiWriter(sr, srcConn), targetConn)
		if err != nil {
			log.Printf("reverse: connection error %v", err)
			return
		}
		log.Printf("reverse: wrote %d bytes", n)
	}()
	wg.Wait()

	return
}

type SessionRecorder struct {
	sessionID string
	storage   *os.File
}

func (sr *SessionRecorder) Write(b []byte) (n int, err error) {
	log.Printf("%v", hex.Dump(b))
	return sr.storage.Write(b)
}

func (sr *SessionRecorder) Read(b []byte) (n int, err error) {
	return sr.storage.Read(b)
}

func (sr *SessionRecorder) Close() (err error) {
	if sr.storage != nil {
		err = sr.storage.Close()
	}
	return
}
