package main

import (
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.LUTC | log.Lshortfile)
}

func main() {
	log.Println("starting exporter at 9900")

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	http.Handle("/list", http.FileServer(http.Dir("/opt/storage")))

	http.HandleFunc("/encode", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			storageID := r.URL.Query().Get("storage_id")

			log.Printf("Received storage id %v for encode", storageID)
			w.WriteHeader(http.StatusOK)

			return
		} else {
			http.Error(w, "unsupported http method", http.StatusBadRequest)
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
