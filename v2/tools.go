package toolbox

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var randomRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+_1234567890")

// Tools is the type used to instantiate module. Any variable of this type
// will have access to the methods with receiver *Tools.
type Tools struct {
	UploadedFile       UploadedFile
	MaxJSONSize        int
	AllowUnknownFields bool
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

// UploadAFile is a convenience method that calls UploadFiles, only one file is uploaded
func (t *Tools) UploadAFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}
	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}
	return files[0], nil
}

// UploadFiles uploads one or more files to a specific directory and generates a renames each file to a random filename.
// The function returns a slice of newly named files, the original file names, the file size, max file size of files set to 1 GiB
// and the allowed file types, and possible error.
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile
	if t.UploadedFile.MaxFileSize == 0 {
		t.UploadedFile.MaxFileSize = 1024 * 1024 * 1024
	}

	err := t.MakeDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(int64(t.UploadedFile.MaxFileSize))
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

				uploadedFile.OrigFileName = h.Filename
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

// MakeDirIfNotExist creates a directory, and all necessary parents, if it does not exist
func (t *Tools) MakeDirIfNotExist(path string) error {
	// Octal representation of file permission
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanDirectory removes all files in a directory. os.RemoveAll is a similar function but
// removes everything and its path.
func (t *Tools) CleanDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, f := range files {
		err = os.Remove(fmt.Sprintf("%s/%s", path, f))
		if err != nil {
			return err
		}
	}
	return nil
}

// SLugify creates a URL-friendly "slug" fro ma given string.
func (t *Tools) Slugify(s string) (string, error) {
	if s == "" {
		return "", errors.New("input string cannot be empty")
	}
	// regex expression to match non-alphanumeric characters
	re := regexp.MustCompile(`[^a-z\d]+`)
	// convert the string to lowercase and replace non-alphanumeric characters with hyphens
	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(slug) == 0 {
		return "", errors.New("slug is empty after character removal")
	}
	return slug, nil
}

// DownloadStaticFile handles the download of a file from the server.
// It sets the appropriate headers to force the browser to download the file
// instead of displaying it inline. This function allows specifying a custom
// display name for the downloaded file.
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, pathName, displayName string) {

	if _, err := os.Stat(pathName); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, pathName)
}

// JSONResponse is a struct used to pass JSON data around
type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReadJSON reads the JSON body of a request into a provided data structure.
// It handles various error cases and limits the size of the JSON body to prevent
// denial-of-service attacks. If AllowUnknownFields is set to false, it disallows
// unknown fields in the JSON. It returns an error if any issues are encountered
// during decoding.
func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1024 * 1024
	if t.MaxJSONSize != 0 {
		maxBytes = t.MaxJSONSize
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)

	if !t.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}

	err := dec.Decode(data)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("the request body contains malformed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("the request body contains incomplete JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("the request body contains an incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("the request body contains an incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("the request body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("the request body contains an unknown key %s", fieldName)
		case err.Error() == "http: request body too large":
			return fmt.Errorf("the request body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("unable to unmarshal the JSON request body: %s", err.Error())
		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must contain only one JSON value")
	}
	return nil
}

// WriteJSON writes a response status code, arbitrary data and JSON.
func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for k, v := range headers[0] {
			w.Header()[k] = v
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}
	return nil
}

// ErrorJSON takes an error, optional status code and returns a possible error
func (t *Tools) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()

	return t.WriteJSON(w, statusCode, payload)
}

// PostJSONWithClient sends a JSON-encoded request to a specified URI using an HTTP POST method.
// It allows for an optional custom HTTP client and returns the HTTP response, status code,
// and an error if any occurred during the process.
func (t *Tools) PostJSONWithClient(uri string, data interface{}, client ...*http.Client) (*http.Response, int, error) {
	// Create json
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, 0, err
	}
	// check custom http client
	httpClient := &http.Client{}
	if len(client) > 0 {
		httpClient = client[0]
	}
	// build request and set header
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	// call remote uri
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if res.Body != nil {
			res.Body.Close()
		}
	}()
	// send response back
	return res, res.StatusCode, nil
}
