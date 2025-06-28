package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"image"
	"image/jpeg"

	"github.com/nfnt/resize"
)

func main() {
	build()
	serve()
}

func serve() {
	fs := http.FileServer(http.Dir("docs"))
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
	outputHTMLPath := "docs/index.html"
	imagesOutputDir := "docs/images"

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

	// Create the images output directory if it doesn't exist
	if _, err := os.Stat(imagesOutputDir); os.IsNotExist(err) {
		os.MkdirAll(imagesOutputDir, os.ModePerm)
	}

	// Find unused images
	unusedImages, err := findUnusedImages(postsData, imagesPath)
	if err != nil {
		fmt.Printf("Error finding unused images: %v\n", err)
		return
	}

	if len(unusedImages) > 0 {
		fmt.Println("Adding new images to the index json...")
		slices.Reverse(unusedImages)
		newPosts := make([]Post, 0)
		for _, image := range unusedImages {
			fmt.Printf("Adding image: %s\n", image)
			newPosts = append(newPosts, Post{
				Title:   "New",
				Caption: "Meaningful caption",
				Image:   image,
			})
		}
		postsData.Posts = append(newPosts, postsData.Posts...)
		postsDataJSON, err := json.MarshalIndent(postsData, "", "  ")
		if err != nil {
			fmt.Printf("Error marshalling updated JSON data: %v\n", err)
			return
		}
		err = os.WriteFile(indexJSONPath, postsDataJSON, 0644)
		if err != nil {
			fmt.Printf("Error writing updated JSON data to file: %v\n", err)
			return
		}
		fmt.Println("Updated index.json with new images.")
	}

	// Execute template with the data
	err = tmpl.Execute(outputFile, postsData)
	if err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		return
	}

	// Copy and resize images
	for _, post := range postsData.Posts {
		srcImagePath := filepath.Join(imagesPath, post.Image)
		dstImagePath := filepath.Join(imagesOutputDir, filepath.Base(post.Image))

		if _, err := os.Stat(dstImagePath); err == nil {
			fmt.Printf("Image %s already exists, skipping...\n", post.Image)
			continue
		}

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

func findUnusedImages(postsData PostsData, imagesPath string) ([]string, error) {
	usedImages := make(map[string]bool)
	for _, post := range postsData.Posts {
		usedImages[post.Image] = true
	}

	var unusedImages []string
	err := filepath.Walk(imagesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".jpg" || filepath.Ext(path) == ".jpeg") {
			imageName := filepath.Base(path)
			if !usedImages[imageName] {
				unusedImages = append(unusedImages, imageName)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return unusedImages, nil
}
