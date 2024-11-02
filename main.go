package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"image"
	"image/jpeg"

	"github.com/nfnt/resize"
)

func main() {
	serveFlag := flag.Bool("serve", false, "Start the server")
	flag.Parse()
	if *serveFlag {
		serve()
		return
	}
	build()
	serve()
}

func serve() {
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", http.StripPrefix("/", fs))

	log.Println("Server starting at :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// Post represents the structure of each post in the JSON data.
type Post struct {
	Title   string `json:"title"`
	Caption string `json:"caption"`
	Image   string `json:"image"`
}

// PostsData represents the structure of the JSON data.
type PostsData struct {
	Posts []Post `json:"posts"`
}

func build() {
	// Define paths
	indexJSONPath := "source/index.json"
	imagesPath := "source/images"
	templatePath := "template/index.html"
	outputHTMLPath := "public/index.html"
	imagesOutputDir := "public/images"

	// Read and parse the JSON data
	var postsData PostsData
	jsonFile, err := os.Open(indexJSONPath)
	if err != nil {
		fmt.Printf("Error opening JSON file: %v\n", err)
		return
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)

	if err != nil {
		fmt.Printf("Error reading JSON file: %v\n", err)
		return
	}

	err = json.Unmarshal(byteValue, &postsData)

	if err != nil {
		fmt.Printf("Error parsing JSON data: %v\n", err)
		return
	}

	fmt.Printf("JSON data: %+v\n", postsData)

	// Parse the template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		return
	}

	// Create the output HTML file
	outputFile, err := os.Create(outputHTMLPath)
	if err != nil {
		fmt.Printf("Error creating output HTML file: %v\n", err)
		return
	}
	defer outputFile.Close()

	// Execute template with the data
	err = tmpl.Execute(outputFile, postsData)
	if err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		return
	}

	// Create the images output directory if it doesn't exist
	if _, err := os.Stat(imagesOutputDir); os.IsNotExist(err) {
		os.MkdirAll(imagesOutputDir, os.ModePerm)
	}

	// Copy and resize images
	for _, post := range postsData.Posts {
		srcImagePath := filepath.Join(imagesPath, post.Image)
		dstImagePath := filepath.Join(imagesOutputDir, filepath.Base(post.Image))

		// Open the source image
		srcImageFile, err := os.Open(srcImagePath)
		if err != nil {
			fmt.Printf("Error opening source image %s: %v\n", post.Image, err)
			continue
		}
		defer srcImageFile.Close()

		// Decode the image
		img, _, err := image.Decode(srcImageFile)
		if err != nil {
			fmt.Printf("Error decoding image %s: %v\n", post.Image, err)
			continue
		}

		// Resize the image (e.g., to 800x600 for web)
		resizedImg := resize.Resize(1440, 0, img, resize.Lanczos3)

		// Save the resized image
		dstImageFile, err := os.Create(dstImagePath)
		if err != nil {
			fmt.Printf("Error creating destination image %s: %v\n", dstImagePath, err)
			continue
		}
		defer dstImageFile.Close()

		err = jpeg.Encode(dstImageFile, resizedImg, nil)
		if err != nil {
			fmt.Printf("Error saving resized image %s: %v\n", dstImagePath, err)
		}

		fmt.Printf("Resized image saved to %s\n", dstImagePath)
	}

	fmt.Println("HTML and images have been generated successfully.")
}
