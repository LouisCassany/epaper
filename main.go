package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
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

	// Upload picture with aspect ratio padding to 800x480
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

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		srcImg, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			http.Error(w, "Invalid image file", http.StatusBadRequest)
			return
		}

		srcBounds := srcImg.Bounds()
		srcW := srcBounds.Dx()
		srcH := srcBounds.Dy()

		targetAspect := 800.0 / 480.0
		srcAspect := float64(srcW) / float64(srcH)

		var canvasW, canvasH int

		if srcAspect > targetAspect {
			// Image is too wide: match width, pad height
			canvasW = srcW
			canvasH = int(float64(srcW) / targetAspect)
		} else {
			// Image is too tall: match height, pad width
			canvasH = srcH
			canvasW = int(float64(srcH) * targetAspect)
		}

		// Create padded canvas
		canvas := imaging.New(canvasW, canvasH, color.NRGBA{255, 255, 255, 255})

		offsetX := (canvasW - srcW) / 2
		offsetY := (canvasH - srcH) / 2
		result := imaging.Paste(canvas, srcImg, image.Pt(offsetX, offsetY))

		// Save the result
		dstPath := filepath.Join("static", "pictures", filepath.Base(handler.Filename))
		if err := imaging.Save(result, dstPath); err != nil {
			http.Error(w, "Failed to save image", http.StatusInternalServerError)
			return
		}

		w.Write([]byte("Upload and aspect-ratio padding successful"))
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
