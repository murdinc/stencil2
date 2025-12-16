package media

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"

	"github.com/nfnt/resize"
)

func ProxyAndResizeImage(imageURL string, targetWidth int, w http.ResponseWriter, acceptWebP bool) error {
	// Download the image from the URL
	response, err := http.Get(imageURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch image: %s", response.Status)
	}

	// Read the image data into memory
	imageData, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	// Determine the input image format
	format := http.DetectContentType(imageData)

	// Return the original file if the format is not GIF or PNG
	if format != "image/jpeg" && format != "image/png" {
		w.Header().Set("Content-Type", format)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(imageData)))
		w.Header().Set("Content-Disposition", "inline")

		bufferedWriter := bufio.NewWriter(w)
		_, err := bufferedWriter.Write(imageData)
		if err != nil {
			return err
		}
		bufferedWriter.Flush()

		return nil
	}

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return err
	}

	// Resize the image
	resizedImage := resize.Resize(uint(targetWidth), 0, img, resize.Lanczos3)

	// Encode the resized image
	imageBytes, err := encodeImage(resizedImage, format)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Set the appropriate headers for the HTTP response
	w.Header().Set("Content-Type", format)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(imageBytes)))

	// Use a buffered writer for improved writing performance
	bufferedWriter := bufio.NewWriter(w)
	_, err = bufferedWriter.Write(imageBytes)
	if err != nil {
		return err
	}
	bufferedWriter.Flush()

	return nil
}

// Function to read a portion of the response body for MIME type detection
func readSniffData(reader io.Reader) ([]byte, error) {
	// Read a limited amount of bytes for content type detection
	sniffSize := 512
	sniffedData := make([]byte, sniffSize)
	_, err := io.ReadFull(reader, sniffedData)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return sniffedData, nil
}

// Function to check if the MIME type is convertible (e.g., PNG or GIF)
func isConvertibleMIMEType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/png") || strings.HasPrefix(contentType, "image/jpeg")
}

func getImageFormat(data []byte) string {
	// Use the image package to determine the format
	_, format, _ := image.DecodeConfig(bytes.NewReader(data))
	return format
}

func encodeImage(img image.Image, format string) ([]byte, error) {
	var imageBytes []byte
	buffer := new(bytes.Buffer)

	// Encode the image based on the format
	switch strings.ToLower(format) {
	case "image/jpeg":
		err := jpeg.Encode(buffer, img, nil)
		if err != nil {
			return nil, err
		}
	case "image/png":
		err := png.Encode(buffer, img)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	imageBytes = buffer.Bytes()
	return imageBytes, nil
}
