package main

import (
	"archive/zip"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func openBrowser(url string) { //Open the browser with the url as parameter (localhost:8080)
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

const port = ":8080"

func runDelayed() {
	//Wait 2 seconds before opening the browser
	time.Sleep(2 * time.Second)
	openBrowser("http://localhost" + port + "/")
}

//go:embed pages.zip
var archive []byte

const fName = "tmp.zip"

func autoExtract() {
	//Check if folder "pages" exists
	if _, err := os.Stat("./pages/"); os.IsNotExist(err) {
		fmt.Println("Self-Extracting webpages please wait...")
		//Create the file
		file, err := os.Create(fName)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				log.Fatal(err)
				return
			}
			err = os.Remove(fName)
			if err != nil {
			}
		}(file)
		//Write the file
		_, err = file.Write(archive)
		if err != nil {
			log.Fatal(err)
			return
		}
		//Unzip the file
		fmt.Println("Unzipping " + fName)
		err = Unzip(fName, "./pages/")
		if err != nil {
			log.Fatal(err)
			return
		}
	}
}

func cleanup() {
	//delete folder "pages"
	err := os.RemoveAll("./pages/")
	if err != nil {
	}
}

func main() {
	autoExtract()

	defer func() {
		cleanup()
	}()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(-1)
	}()

	fmt.Printf("Server running on 8080\n")

	//Serve files from folder "pages"
	fileServer := http.FileServer(http.Dir("./pages"))
	http.Handle("/", fileServer)
	//Run the browser
	go runDelayed()
	//Start the server
	fmt.Println("Stop the server with Ctrl+C")
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
