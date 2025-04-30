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
	"os/exec"
	"path/filepath"

	"github.com/disintegration/imaging"
)

func main() {
	// Set up file server for static content
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// Register API endpoints
	http.HandleFunc("/list-pictures", listPicturesHandler)
	http.HandleFunc("/upload-picture", uploadPictureHandler)
	http.HandleFunc("/delete-picture", deletePictureHandler)
	http.HandleFunc("/display-picture", displayPictureHandler)

	// Start the server
	log.Println("Serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// listPicturesHandler returns a list of all pictures in the pictures directory
func listPicturesHandler(w http.ResponseWriter, r *http.Request) {
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
}

// uploadPictureHandler processes uploaded images and adds aspect ratio padding to 800x480
func uploadPictureHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB max
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

	processedImage, err := processImageWithAspectRatio(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Save the processed image
	dstPath := filepath.Join("static", "pictures", filepath.Base(handler.Filename))
	if err := imaging.Save(processedImage, dstPath); err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Upload and aspect-ratio padding successful"))
}

// processImageWithAspectRatio adds padding to maintain 800x480 aspect ratio
func processImageWithAspectRatio(imageData []byte) (image.Image, error) {
	srcImg, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, err
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

	return imaging.Paste(canvas, srcImg, image.Pt(offsetX, offsetY)), nil
}

// deletePictureHandler removes a picture file
func deletePictureHandler(w http.ResponseWriter, r *http.Request) {
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
}

// displayPictureHandler prints the path of the requested picture
func displayPictureHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing file name", http.StatusBadRequest)
		return
	}

	// Create the full path to the picture
	picturePath := filepath.Join("static", "pictures", filepath.Base(name))

	// Get absolute path for demonstration purposes
	absPath, err := filepath.Abs(picturePath)
	if err != nil {
		http.Error(w, "Failed to get absolute path", http.StatusInternalServerError)
		return
	}

	// Print the path to the server logs
	log.Printf("Displaying picture: %s", absPath)

	pythonPath := "/home/louis/.virtualenvs/pimoroni/bin/python"

	scriptPath := "./static/image.py"

	cmd := exec.Command(pythonPath, scriptPath, "--file", picturePath)
	if err := cmd.Run(); err != nil {
		http.Error(w, "Failed to run image.py", http.StatusInternalServerError)
		println(err.Error())
		return
	}

	w.Write([]byte("Display successful"))
}
