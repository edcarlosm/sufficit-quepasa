package library

import (
	"mime"
	"net/http"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// Validate email string
func IsValidEMail(s string) bool {
	var rx = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

	if len(s) < 255 && rx.MatchString(s) {
		return true
	}

	return false
}

// Returns a string representation of source type interface
func GetTypeString(myvar interface{}) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	} else {
		return t.Name()
	}
}

func GetMimeTypeFromContent(content []byte, filename string) string {
	mimeType := http.DetectContentType(content)
	if mimeType == "application/octet-stream" && len(filename) > 0 {
		extension := filepath.Ext(filename)
		newMimeType := mime.TypeByExtension(extension)
		if len(newMimeType) > 0 {
			mimeType = newMimeType
		}
	}
	return mimeType
}

func GenerateFileNameFromMimeType(mimeType string) string {

	const layout = "20060201150405"
	t := time.Now().UTC()
	filename := "file-" + t.Format(layout)

	// get file extension from mime type
	extension, _ := mime.ExtensionsByType(mimeType)
	if len(extension) > 0 {
		filename = filename + extension[0]
	}

	return filename
}

// Get the first discovered extension from a given mime type (with dot = {.ext})
func TryGetExtensionFromMimeType(mimeType string) (exten string, err error) {
	extensions, err := mime.ExtensionsByType(mimeType)
	if len(extensions) > 0 {
		exten = extensions[0]
		if !strings.HasPrefix(exten, ".") {
			exten = "." + exten
		}
	}
	return
}

// Force the recognition of some types of mime string
func EnsureMimesMapping() {
	_ = mime.AddExtensionType(".webp", "image/webp")
	_ = mime.AddExtensionType(".mp4", "video/mp4")
}
