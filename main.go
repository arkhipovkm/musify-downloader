package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	for {
		t1 := time.Now()
		// _, n, err := download.MP3File(
		// 	os.Args[1],
		// 	"test.mp3",
		// )
		uri := os.Args[1]
		resp, err := http.Get(uri)
		defer resp.Body.Close()
		t1_5 := time.Now()
		log.Printf("First response in %.1f ms", float64(t1_5.UnixNano()-t1.UnixNano())/float64(1e6))
		b, err := ioutil.ReadAll(resp.Body)
		n := len(b)
		if err != nil {
			panic(err)
		}
		t2 := time.Now()
		log.Printf("Fetched audio: %d bytes in %.1f ms, %.1f MB/s\n", n, float64(t2.UnixNano()-t1.UnixNano())/float64(1e6), float64(n)/float64(1e6)/(float64(t2.UnixNano()-t1.UnixNano())/float64(1e9)))
	}
}
