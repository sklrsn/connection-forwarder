package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

func init() {
	log.SetFlags(log.LUTC | log.Lshortfile)
	log.SetPrefix("=>")
}

func main() {
	log.Println("starting exporter at 9900")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/storage", http.StatusMovedPermanently)
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		response := make(map[string]interface{}, 0)
		response["time"] = time.Now().UTC()

		bs, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n, err := w.Write(bs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("wrote %d bytes", n)
	})

	http.Handle("/storage", http.StripPrefix("/storage", http.FileServer(http.Dir("/opt/storage"))))
	http.Handle("/enriched", http.StripPrefix("/enriched", http.FileServer(http.Dir("/opt/downloads"))))

	http.HandleFunc("/transform", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			storageID := r.URL.Query().Get("storage_id")
			format := r.URL.Query().Get("format")

			var tfr VideoTransformer
			switch format {
			case "avi":
				tfr = Avi{
					storageID: storageID,
					codec:     "h264",
				}
			default:
				tfr = Avi{
					storageID:    storageID,
					codec:        "h264",
					enrichmentID: uuid.NewString(),
				}
			}
			if err := tfr.Transform(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Printf("Received storage id %v for encode", storageID)
			w.WriteHeader(http.StatusOK)
			return
		} else {
			http.Error(w, "http method not supported", http.StatusBadRequest)
			return
		}
	})

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			videoID := r.URL.Query().Get("video_id")

			log.Printf("Received download %v for download", videoID)
			w.WriteHeader(http.StatusOK)

			return
		} else {
			http.Error(w, "unsupported http method", http.StatusBadRequest)
			return
		}
	})

	log.Fatalf("%v", http.ListenAndServe(":9900", nil))

}

type VideoTransformer interface {
	Transform() error
}

type Avi struct {
	storageID    string
	enrichmentID string
	codec        string
}

func (avi Avi) Transform() error {
	in, err := os.Open(fmt.Sprintf("/opt/storage/%v", avi.storageID))
	if err != nil {
		return err
	}

	out, err := os.Create(fmt.Sprintf("/opt/downloads/%v", avi.enrichmentID))
	if err != nil {
		return err
	}

	cmdStr := strings.Builder{}
	if _, err := cmdStr.Write([]byte("-i pipe: ")); err != nil {
		return err
	}
	if _, err := cmdStr.Write([]byte(fmt.Sprintf("-c:v %v ", avi.codec))); err != nil {
		return err
	}
	if _, err := cmdStr.Write([]byte("-f avi pipe:")); err != nil {
		return err
	}
	log.Printf("Executing command ffmpeg :%v", cmdStr.String())

	var errBuffer bytes.Buffer
	encCmd := exec.Command("/usr/local/bin/ffmpeg", cmdStr.String())
	encCmd.Stdin = in
	encCmd.Stdout = out
	encCmd.Stderr = &errBuffer
	if err := encCmd.Run(); err != nil {
		if errBuffer.Available() > 0 {
			return fmt.Errorf("%v", errBuffer.String())
		}
		return err
	}

	return nil
}
