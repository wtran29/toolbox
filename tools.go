package toolbox

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var randomRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+_1234567890")

// Tools is the type used to instantiate module. Any variable of this type
// will have access to the methods with receiver *Tools.
type Tools struct {
	UploadedFile UploadedFile
}

// RandomString generates a random string of length using characters from randomRunes
func (t *Tools) RandomString(n int) string {
	runes := []rune(randomRunes)
	result := make([]rune, n)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(runes))))
		result[i] = runes[num.Int64()]
	}
	return string(result)
}

// UploadedFile is a struct represents saved information about an uploaded file
type UploadedFile struct {
	NewFileName      string
	OrigFileName     string
	FileSize         int64
	MaxFileSize      int
	AllowedFileTypes []string
}

func (t *Tools) UploadFile(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile
	if t.UploadedFile.MaxFileSize == 0 {
		t.UploadedFile.MaxFileSize = 1024 * 1024 * 1024
	}

	err := r.ParseMultipartForm(int64(t.UploadedFile.MaxFileSize))
	if err != nil {
		return nil, errors.New("uploaded file is too big")
	}

	for _, fileHeaders := range r.MultipartForm.File {
		for _, h := range fileHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile
				inFile, err := h.Open()
				if err != nil {
					return nil, err
				}
				defer inFile.Close()

				// buffer of 512 bytes
				buffer := make([]byte, 512)
				_, err = inFile.Read(buffer)
				if err != nil {
					return nil, err
				}

				// TODO: check if file type is permitted
				allowed := false
				fileType := http.DetectContentType(buffer)

				if len(t.UploadedFile.AllowedFileTypes) > 0 {
					for _, t := range t.UploadedFile.AllowedFileTypes {
						if strings.EqualFold(fileType, t) {
							allowed = true
						}
					}
				} else {
					allowed = true
				}
				if !allowed {
					return nil, errors.New("uploaded file type not permitted")
				}
				_, err = inFile.Seek(0, 0)
				if err != nil {
					return nil, err
				}
				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(h.Filename))
				} else {
					uploadedFile.NewFileName = h.Filename
				}
				var outFile *os.File
				defer outFile.Close()

				if outFile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outFile, inFile)
					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = fileSize
				}
				uploadedFiles = append(uploadedFiles, &uploadedFile)
				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}
	return uploadedFiles, nil
}
