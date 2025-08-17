# Gits
Gits is a lightweight library written in Go to develop git servers. It handles the core parts (advertising, uploading packs and receiving packs) effortlessly and that's the only goal of this library.


## Features

1. Advertising
2. Upload pack
3. Receive pack
4. Support custom filesystem

## API
```go
import (
    "gits"
)

// Creating a new repo.
repo, err := gits.InitRepo(&gits.Config{
    Dir:  "/path/to/base/dir",
    Name: "my-repo",
    FS:   nil, // Leaving this nil defaults to disk file system.
})

// Opening an existing repo.
repo, err := gits.OpenRepo(&gits.Config{
    Dir:  "/path/to/base/dir",
    Name: "my-repo",
    FS:   nil, // Leaving this nil defaults to disk file system.
})

// Advertisement.
// service = git-upload-pack or git-receive-pack
// Cb is called before sending advertisement. Can be used to send HTTP headers.
repo.Advertise(r io.Reader, w io.Writer, service string, cb func())

// Upload pack.
// Cb is called before unpacking starts.
repo.UploadPack(r io.Reader, w io.Writer, cb func())

// Receive pack.
// Cb is called before the report.
repo.ReceivePack(r io.Reader, w io.Writer, cb func())
```

## Sample HTTP Server
```go
func GitHTTPHandler(repo *gits.Repo) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/info/refs"):
			fmt.Println("Calling: info/refs")
			handleInfoRefs(w, r, repo)
			return
		case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
			fmt.Println("Calling: upload-pack")
			handleUploadPack(w, r, repo)
			return
		case strings.HasSuffix(r.URL.Path, "/git-receive-pack"):
			fmt.Println("Calling: receive-pack")
			handleReceivePack(w, r, repo)
			return
		default:
			http.NotFound(w, r)
			return
		}
	})
}

func handleInfoRefs(w http.ResponseWriter, r *http.Request, repo *gits.Repo) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	service := r.URL.Query().Get("service")
	if service == "" || (service != "git-upload-pack" && service != "git-receive-pack") {
		http.Error(w, "unsupported or missing service", http.StatusBadRequest)
		return
	}

	err := repo.Advertise(r.Body, w, service, func() {
		setNoCache(w)
		w.Header().Set("Content-Type", "application/x-"+service+"-advertisement")
		w.WriteHeader(http.StatusOK)
	})

	if err != nil {
		http.Error(w, "advertise error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleUploadPack(w http.ResponseWriter, r *http.Request, repo *gits.Repo) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Optional: enforce request content type (clients usually send this)
	// if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-git-upload-pack-request") {
	// 	http.Error(w, "bad content type", http.StatusUnsupportedMediaType)
	// 	return
	// }

	err := repo.UploadPack(r.Body, w, func() {
		setNoCache(w)
		w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		w.WriteHeader(http.StatusOK)
	})

	if err != nil {
		fmt.Println("Error uploading pack:", err)
		return
	}
}

func handleReceivePack(w http.ResponseWriter, r *http.Request, repo *gits.Repo) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := repo.ReceivePack(r.Body, w, func() {
		setNoCache(w)
		w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
		w.WriteHeader(http.StatusOK)
	})

	if err != nil {
		http.Error(w, "Error receiving pack: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func setNoCache(w http.ResponseWriter) {
	// Common git-http-backend headers
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
}

func main() {
	repo, err := gits.OpenRepo(&gits.Config{
		Dir:  "/path/to/repos",
        Name: "my-repo",
		FS:   gits.NewDiskFS,
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/", GitHTTPHandler(repo))

	addr := ":9191"
	log.Printf("git smart http listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
```

## Status

This library is still in development, so there are many things to be enhanced. However, it works and does the features mentioned above.
