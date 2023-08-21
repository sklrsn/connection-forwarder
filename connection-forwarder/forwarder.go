package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	mrand "math/rand"
	"net"
	"os"
	"sync"
	"time"
)

func init() {
	log.SetFlags(log.LUTC | log.Llongfile)
	log.SetPrefix("=> ")
}

type x509certificate struct {
	certificate []byte
	priv        []byte
	pub         []byte
}

func generateCertificate() (*x509certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	x509Cert := x509.Certificate{
		SerialNumber: big.NewInt(mrand.Int63n(math.MaxInt64)),
		Subject: pkix.Name{
			Organization:  []string{"private, INC."},
			Country:       []string{"FI"},
			Province:      []string{"Uusimaa"},
			Locality:      []string{"Helsinki"},
			StreetAddress: []string{""},
			PostalCode:    []string{"0200"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certificate, err := x509.CreateCertificate(rand.Reader, &x509Cert, &x509Cert, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}
	return &x509certificate{
		certificate: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate}),
		priv:        pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}),
	}, nil
}
func main() {
	cert, err := generateCertificate()
	if err != nil {
		log.Fatalf("forwarder: error occurred %v", err)
	}
	tlsCertificate, err := tls.X509KeyPair(cert.certificate, cert.priv)
	if err != nil {
		log.Fatalf("forwarder: error occurred %v", err)
	}
	var lr net.Listener
	switch protocol := os.Getenv("PROTOCOL"); protocol {
	case "RDP":
		lr, err = tls.Listen("tcp", ":3389", &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{tlsCertificate},
		})
	case "VNC":
		lr, err = net.Listen("tcp", ":5900")
	default:
		log.Fatal()
	}
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
		sessionID: fmt.Sprintf("%v", mrand.Int63n(math.MaxInt64)),
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
