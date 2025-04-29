package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// List files
	http.HandleFunc("/list-pictures", func(w http.ResponseWriter, r *http.Request) {
		picturesDir := "./static/pictures"
		entries, err := os.ReadDir(picturesDir)
		if err != nil {
			http.Error(w, "Failed to read pictures directory", http.StatusInternalServerError)
			return
		}

		var files []string
		for _, entry := range entries {
			if !entry.IsDir() {
				files = append(files, entry.Name())
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	})

	// Upload picture
	http.HandleFunc("/upload-picture", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}
		file, handler, err := r.FormFile("picture")
		if err != nil {
			http.Error(w, "Error retrieving file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		dstPath := filepath.Join("static", "pictures", filepath.Base(handler.Filename))
		dst, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, "Could not save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Upload successful"))
	})

	// Delete picture
	http.HandleFunc("/delete-picture", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "Missing file name", http.StatusBadRequest)
			return
		}
		p := filepath.Join("static", "pictures", filepath.Base(name))
		if err := os.Remove(p); err != nil {
			http.Error(w, "Failed to delete file", http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Delete successful"))
	})

	log.Println("Serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
