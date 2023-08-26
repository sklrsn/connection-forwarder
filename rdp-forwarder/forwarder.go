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

	"github.com/google/uuid"
	"github.com/sklrsn/video-convertor/rdp-forwarder/guacd"
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
	lr, err = tls.Listen("tcp", ":3389", &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{tlsCertificate},
	})
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

		guacdConn, err := net.Dial("tcp", os.Getenv("GUACD_ADDR"))
		if err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			continue
		}

		fc, err := guacd.NewForwarderConnection(srcConn, guacdConn, os.Getenv("TARGET_ADDR"))
		if err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(
			guacd.GuacamoleMessage{
				OpCode: "select",
				Args:   []string{"vnc"},
			}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		msg, err := fc.Forward.ReadGuacamoleMessage()
		if err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}
		log.Println(msg)

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "size",
			Args:   []string{"1024", "768", "96"},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "audio",
			Args:   []string{"audio/ogg"},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "video",
			Args:   []string{},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "image",
			Args:   []string{"image/png", "image/jpeg"},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "timezone",
			Args:   []string{"Europe/Helsinki"},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "connect",
			Args:   []string{"Europe/Helsinki"},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

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
