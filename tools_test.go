package toolbox

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

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
