package toolbox

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PostJSONWithClient(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		// test request parameters
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}

	})
	var testTools Tools
	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"
	_, _, err := testTools.PostJSONWithClient("http://example.com/some/path", foo, client)
	if err != nil {
		t.Error("failed to call remote url:", err)
	}
}

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Errorf("wrong length. wanted=%d, got=%d", 10, len(s))
	}

}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up a pipe to avoid buffering
		pipeRead, pipeWrite := io.Pipe()
		writer := multipart.NewWriter(pipeWrite)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer writer.Close()
			defer wg.Done()

			// create form data field 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding the image:", err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()
		// read from the pipe that receives data
		req := httptest.NewRequest("POST", "/", pipeRead)
		req.Header.Add("Content-Type", writer.FormDataContentType())
		var testTools Tools
		testTools.UploadedFile.AllowedFileTypes = e.allowedTypes
		uploadedFiles, err := testTools.UploadFiles(req, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}
		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}
			// clean up
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))

		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but non received", e.name)
		}
		wg.Wait()
	}
	cleanDirectory("./testdata/uploads")
}

func cleanDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = os.Remove(fmt.Sprintf("%s/%s", path, file))
		if err != nil {
			return err
		}
	}

	return nil
}

func TestTools_UploadAFile(t *testing.T) {
	for _, e := range uploadTests {
		// set up a pipe to avoid buffering
		pipeRead, pipeWrite := io.Pipe()
		writer := multipart.NewWriter(pipeWrite)
		go func() {
			defer writer.Close()

			// create form data field 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding the image:", err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()
		// read from the pipe that receives data
		req := httptest.NewRequest("POST", "/", pipeRead)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools

		uploadedFiles, err := testTools.UploadAFile(req, "./testdata/uploads/", true)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
			t.Errorf("expected file to exist: %s", err.Error())
		}

		// clean up
		_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))

	}
	cleanDirectory("./testdata/uploads")
}

func TestTools_MakeDirIfNotExists(t *testing.T) {
	var testTool Tools

	err := testTool.MakeDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	err = testTool.MakeDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "test the slug time", expected: "test-the-slug-time", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "Test + the & SLUG TiMe &^42", expected: "test-the-slug-time-42", errorExpected: false},
	{name: "japanese string", s: "スラグタイムをテストする", expected: "", errorExpected: true},
	{name: "japanese string and roman characters", s: "hello world スラグタイムをテストする", expected: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools

	for _, e := range slugTests {
		slug, err := testTool.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}

		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned. wanted=%s, got=%s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools
	testTool.DownloadStaticFile(rr, req, "./testdata", "img.png", "landscape.jpg")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "534283" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"landscape.jpg\"" {
		t.Error("wrong content disposition")
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "valid json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "invalid json", json: `{"foo": }`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "invalid type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "1"}{"alpha":"beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax json error", json: `{"foo": "1"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field", json: `{"fooo": "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "allow unknown fields in json", json: `{"fooo": "1"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 5, allowUnknown: true},
	{name: "not json", json: `foo bar`, errorExpected: true, maxSize: 1024, allowUnknown: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, e := range jsonTests {
		// set max file size
		testTool.MaxJSONSize = e.maxSize

		// allow/disallow unknown fields
		testTool.AllowUnknownFields = e.allowUnknown

		// declare var to read decoded json
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create request with body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		// create recorder
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one received: %s", e.name, err.Error())
		}
		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools
	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("an error message"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error when decoding JSON", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, and it should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; wanted=503, got=%d", rr.Code)
	}
}
