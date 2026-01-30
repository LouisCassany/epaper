package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

var dev = false

//go:embed image.py
var imagePy string

//go:embed index.html
var indexHTML []byte

// Global variable to store picture list with mutex for safe concurrent access
var (
	pictureList   []string
	pictureMux    sync.RWMutex
	currentPic    int
	currentPicMux sync.RWMutex
	lastRotation  time.Time
	rotationTime  = 2 * time.Hour
)

func main() {
	// Initialize picture list on startup
	updatePictureList()
	startPictureRotation()

	// Serve embedded index.html at "/"
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexHTML)
	})

	// Serve /static/ files from disk
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Register API endpoints
	http.HandleFunc("/list-pictures", listPicturesHandler)
	http.HandleFunc("/upload-picture", uploadPictureHandler)
	http.HandleFunc("/delete-picture", deletePictureHandler)
	http.HandleFunc("/display-picture", displayPictureHandler)

	// Start the server
	log.Println("Serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// updatePictureList reads the pictures directory and updates the global pictureList
func updatePictureList() error {
	picturesDir := "./static"
	entries, err := os.ReadDir(picturesDir)
	if err != nil {
		return fmt.Errorf("failed to read pictures directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	// Update the global picture list with write lock
	pictureMux.Lock()
	pictureList = files
	pictureMux.Unlock()

	return nil
}

// listPicturesHandler returns the current list of pictures
func listPicturesHandler(w http.ResponseWriter, r *http.Request) {
	// Update the picture list to ensure it's current
	if err := updatePictureList(); err != nil {
		http.Error(w, "Failed to update pictures list", http.StatusInternalServerError)
		return
	}

	// Get a read lock on the picture list
	pictureMux.RLock()
	files := pictureList
	pictureMux.RUnlock()

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
	dstPath := filepath.Join("static", filepath.Base(handler.Filename))
	if err := imaging.Save(processedImage, dstPath); err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	// Update the picture list after adding a new picture
	updatePictureList()

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

	// check if the image is in portrait mode and rotate it if so
	if srcW < srcH {
		srcImg = imaging.Rotate(srcImg, 90, color.Transparent)
		srcBounds = srcImg.Bounds()
		srcW = srcBounds.Dx()
		srcH = srcBounds.Dy()
	}

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

	p := filepath.Join("static", filepath.Base(name))
	if err := os.Remove(p); err != nil {
		http.Error(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}

	// Update the picture list after deleting a picture
	updatePictureList()

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

	// Get the picture index from the global picture list
	index := -1
	for i, p := range pictureList {
		if p == name {
			index = i
			break
		}
	}
	if index == -1 {
		http.Error(w, "Picture not found", http.StatusNotFound)
		return
	}

	if err := displayPicture(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Display successful"))
}

func displayPicture(index int) error {
	if dev {
		currentPicMux.Lock()
		currentPic = index
		currentPicMux.Unlock()
		fmt.Println("Displaying picture:", pictureList[index], index)
		return nil
	}

	// Get the picture path from the global picture list
	picturePath := filepath.Join("static", pictureList[index])

	absPath, err := filepath.Abs(picturePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Print the path to the server logs
	log.Printf("Displaying picture: %s", absPath)

	pythonPath := "/home/louis/.virtualenvs/pimoroni/bin/python"
	cmd := exec.Command(pythonPath, "-", "--file", absPath)
	cmd.Stdin = bytes.NewBufferString(imagePy)

	_, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Python error:", err)
		return err
	}

	currentPicMux.Lock()
	currentPic = index
	currentPicMux.Unlock()
	return nil
}

func startPictureRotation() {
	go func() {
		for {
			now := time.Now()
			hour := now.Hour()

			if hour >= 8 && hour <= 20 {
				if now.Sub(lastRotation) >= rotationTime {
					pictureMux.RLock()
					count := len(pictureList)
					pictureMux.RUnlock()

					if count > 0 {
						currentPicMux.Lock()
						index := (currentPic + 1) % count
						currentPic = index
						currentPicMux.Unlock()

						if err := displayPicture(index); err != nil {
							log.Printf("Failed to display picture: %v", err)
						} else {
							log.Printf("Rotated to picture #%d: %s", index, pictureList[index])
							lastRotation = now
						}
					}
				}
			}

			time.Sleep(10 * time.Minute)
		}
	}()
}
