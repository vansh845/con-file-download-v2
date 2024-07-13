package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	url        = "https://example.com/myvideo.mp4"
	workerPool = 10
)

type Chunk struct {
	Start int64
	End   int64
}

func DownloadFile() {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	fd, err := os.Create("newFile.mp4")
	if err != nil {
		log.Fatalln(err)
	}

	w, err := io.Copy(fd, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d bytes written\n", w)
}

func mergeFilePieces(directory, outputFile string) error {
	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	files, err := filepath.Glob(filepath.Join(directory, "*.part"))
	if err != nil {
		return err
	}

	for _, file := range files {
		in, err := os.Open(file)
		if err != nil {
			return err
		}
		defer in.Close()

		_, err = io.Copy(out, in)
		if err != nil {
			return err
		}
	}

	return nil
}

func DownloadChunk(wg *sync.WaitGroup, chunk Chunk) {
	defer wg.Done()

	fmt.Printf("Downloading chunk from %d - %d\n", chunk.Start, chunk.End)

	client := http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", chunk.Start, chunk.End))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	fd, err := os.Create(fmt.Sprintf("./parts/%d-%d.part", chunk.Start, chunk.End))
	if err != nil {
		log.Fatalln(err)
	}
	_, err = io.Copy(fd, resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("finished chunk %d - %d\n", chunk.Start, chunk.End)
}

func main() {
	s := time.Now()
	wg := sync.WaitGroup{}
	// getting the content length
	os.Mkdir("parts", os.ModePerm)
	resp, err := http.Head(url)
	if err != nil {
		log.Fatalln("Error while head request", err.Error())
	}

	contentLength := resp.ContentLength
	chunkSize := contentLength / workerPool

	for i := 0; i < workerPool; i++ {
		var chunk Chunk
		chunk.Start = int64(i) * chunkSize

		if i == workerPool-1 {
			chunk.End = contentLength - 1
		} else {
			chunk.End = chunk.Start + chunkSize - 1
		}
		wg.Add(1)
		go DownloadChunk(&wg, chunk)

	}
	wg.Wait()
	l := len(url) - 1

	for url[l] != '.' {
		l--
	}
	typ := url[l+1:]
	err = mergeFilePieces("parts", fmt.Sprintf("downloaded_file.%s", typ))
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("time took - %q\n", time.Since(s))
}
