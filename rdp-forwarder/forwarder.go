package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	mrand "math/rand"
	"net"
	"os"
	"strings"
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
	var lr net.Listener
	lr, err := net.Listen("tcp", ":3389")
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

		fc, err := guacd.NewForwarderConnection(srcConn, guacdConn)
		if err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

	loop:
		for {
			msg, err := fc.Reverse.ReadGuacamoleMessage()
			if err != nil {
				log.Printf("error occurred %v", err)
				_ = srcConn.Close()
				_ = guacdConn.Close()
				break loop
			}

			if msg.OpCode == "connect" {
				log.Printf("%#v", msg)
				log.Println("received connect message from client")
				break loop
			}

			if msg.OpCode == "select" {
				if err := fc.Reverse.WriteGuacamoleMessage(guacd.GuacamoleMessage{
					OpCode: "args",
					Args:   []string{"VERSION_1_5_0", "hostname", "port", "username", "password"},
				}); err != nil {
					_ = srcConn.Close()
					_ = guacdConn.Close()
					break loop
				}
			}
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

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "size",
			Args:   []string{"3360", "1706", "192"},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "audio",
			Args:   []string{"audio/L8", "audio/L16"},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "video",
			Args:   nil,
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "image",
			Args:   []string{"image/png", "image/jpeg", "image/webp"},
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
			OpCode: "name",
			Args:   []string{"guacadmin"},
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		connArgs := make([]string, 0)
		for _, arg := range msg.Args {
			switch arg {
			case "VERSION_1_5_0", "VERSION_1_3_0", "VERSION_1_1_0", "VERSION_1_0_0":
				connArgs = append(connArgs, arg)
			case "hostname":
				connArgs = append(connArgs, strings.Split(os.Getenv("TARGET_ADDR"), ":")[0])
			case "port":
				connArgs = append(connArgs, strings.Split(os.Getenv("TARGET_ADDR"), ":")[1])
			case "password":
				connArgs = append(connArgs, "guest")
			default:
				connArgs = append(connArgs, "")
			}
		}

		if err := fc.Forward.WriteGuacamoleMessage(guacd.GuacamoleMessage{
			OpCode: "connect",
			Args:   connArgs,
		}); err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		msg, err = fc.Forward.ReadGuacamoleMessage()
		if err != nil {
			log.Printf("error occurred %v", err)
			_ = srcConn.Close()
			_ = guacdConn.Close()
			continue
		}

		if msg.OpCode == "ready" {
			if err := fc.Reverse.WriteGuacamoleMessage(msg); err != nil {
				log.Printf("error occurred %v", err)
				_ = srcConn.Close()
				_ = guacdConn.Close()
			}
		}

		go func() {
			_ = serve(fc.Forward.GetRawConn(), fc.Reverse.GetRawConn())
		}()
	}
}

func serve(srcConn, targetConn net.Conn) error {
	defer func() {
		_ = srcConn.Close()
		_ = targetConn.Close()
	}()

	sr := &SessionRecorder{
		sessionID: fmt.Sprintf("%v", uuid.NewString()),
	}
	if err := os.MkdirAll(os.Getenv("STORAGE_LOCATION"), 0777); err != nil {
		log.Printf("%v", err)
		return err
	}
	storage, err := os.Create(fmt.Sprintf("%v/%v", os.Getenv("STORAGE_LOCATION"), sr.sessionID))
	if err != nil {
		return err
	}
	sr.storage = storage

	return handleConnection(srcConn, targetConn, sr)
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
