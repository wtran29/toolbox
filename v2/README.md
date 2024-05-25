# Toolbox

 <b>Toolbox</b> is a convenience package created for the foundational layer of Go projects.

## Purpose

This package was created to limit rewriting functions or repeated code for projects. It is a personal library of reusable functions commonly used in development.

## Features
- [X] <b>Random String Generator</b>: Generates a random string of specified length.
- [X] <b>File Upload</b>: Uploads files to a specified directory with support for MIME type and file size validation.
- [X] <b>Directory Creator</b>: Creates directories for non-existent paths.
- [X] <b>Directory Cleaner</b>: Removes all files in a specified directory while preserving the directory itself.
- [X] <b>Slug Generator</b>: Generates URL-safe slugs from strings.
- [X] <b>File Downloader</b>: Downloads files with a specified name, forcing the browser to avoid displaying them in the browser window.
- [X] <b>JSON Reader</b>: Reads JSON data from an HTTP request and decodes it into a specified struct.
- [X] <b>JSON Writer</b>: Encodes data to JSON and writes it to an HTTP response.
- [X] <b>Post JSON with Client</b>: Sends a JSON-encoded HTTP POST request to a remote service.

## Installation

To install the Toolbox package, run:

```
go get -u github.com/wtran29/toolbox/v2
```

## Usage Examples

### Random String Generator

```
tools := toolbox.Tools{}
randomString := tools.RandomString(10)
fmt.Println(randomString)
```

### File Upload

```
http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
    tools := toolbox.Tools{}
    files, err := tools.UploadFiles(r, "./uploads", true)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    fmt.Fprintf(w, "Files uploaded: %v", files)
})
```

### Directory Creator

```
tools := toolbox.Tools{}
err := tools.CreateDirIfNotExist("./new_directory")
if err != nil {
    log.Fatal(err)
}
```

### Directory Cleaner

```
tools := toolbox.Tools{}
err := tools.CleanDirectory("./uploads")
if err != nil {
    log.Fatal(err)
}
```

### Slug Generator

```
tools := toolbox.Tools{}
slug, err := tools.Slugify("Example String!")
if err != nil {
    log.Fatal(err)
}
fmt.Println(slug) // Output: example-string
```

### File Downloader

```
http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
    tools := toolbox.Tools{}
    tools.DownloadStaticFile(w, r, "./files/example.txt", "example_download.txt")
})
```

### JSON Reader

```
http.HandleFunc("/readjson", func(w http.ResponseWriter, r *http.Request) {
    tools := toolbox.Tools{}
    var data struct {
        Name string `json:"name"`
    }
    err := tools.ReadJSON(w, r, &data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    fmt.Fprintf(w, "Received: %v", data)
})
```

### JSON Writer

```
http.HandleFunc("/writejson", func(w http.ResponseWriter, r *http.Request) {
    tools := toolbox.Tools{}
    payload := toolbox.JSONResponse{
        Error:   false,
        Message: "Success",
    }
    err := tools.WriteJSON(w, http.StatusOK, payload)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
})
```

### Post JSON with Client

```
tools := toolbox.Tools{}
data := map[string]string{"foo": "bar"}
res, statusCode, err := tools.PostJSONWithClient("http://example.com/api", data)
if err != nil {
    log.Fatal(err)
}
defer res.Body.Close()
fmt.Printf("Status Code: %d\n", statusCode)
```

## Contributing
Feel free to open issues or submit pull requests if you have suggestions for improvements or new features.

## License
This project is licensed under the MIT License. See the LICENSE file for details.