package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"
)

func init() {
	log.SetFlags(log.LUTC | log.Lshortfile)
	log.SetPrefix("=>")
}

func main() {
	log.Println("starting exporter at 9900")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/recordings", http.StatusMovedPermanently)
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

	http.Handle("/recordings", http.StripPrefix("/recordings", http.FileServer(http.Dir("/opt/recordings"))))
	http.Handle("/enriched", http.StripPrefix("/enriched", http.FileServer(http.Dir("/opt/downloads"))))

	http.HandleFunc("/transform", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			recordID := r.URL.Query().Get("record_id")

			go func() {
				cmdArgs := make([]string, 0)
				cmdArgs = append(cmdArgs, "-s 640x480")
				cmdArgs = append(cmdArgs, fmt.Sprintf("/opt/recordings/%v", recordID))

				var buff bytes.Buffer
				log.Printf("Executing command with guacenc :%v", cmdArgs)
				encCmd := exec.Command("/usr/local/bin/guacenc", cmdArgs...)
				encCmd.Stderr = &buff
				encCmd.Stdout = &buff
				defer func() {
					log.Println(buff.String())
				}()
				if err := encCmd.Run(); err != nil {
					log.Printf("error occurred %v", err)
					return
				}

			}()

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
