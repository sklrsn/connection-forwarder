package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/google/uuid"
)

func init() {
	log.SetFlags(log.LUTC | log.Lshortfile)
}

func main() {
	log.Println("starting exporter at 9900")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/storage", http.StatusMovedPermanently)
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
	in, err := os.Open(avi.storageID)
	if err != nil {
		return err
	}

	out, err := os.Create(fmt.Sprintf("%v.avi", avi.enrichmentID))
	if err != nil {
		return err
	}

	encCmd := exec.Command(fmt.Sprintf("ffmpeg -i pipe: -c:v %v -f avi pipe:", avi.codec))
	encCmd.Stdin = in
	encCmd.Stdout = out

	if err := encCmd.Run(); err != nil {
		return err
	}

	return nil
}
