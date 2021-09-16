package main

import (
	"log"
	"time"

	"github.com/arkhipovkm/musify/download"
)

func main() {
	for {
		t1 := time.Now()
		_, n, err := download.MP3File(
			// "https://psv4.vkuseraudio.net/audio/ee/bu_xR9d1CEKU214FiRfs5aSyK1IXjAd8-WsePA/a6Ojw4OTExPw/ddQ09LOFhTa2o-PVw/index.m3u8?extra=JpX3heC1rRR8scoEwp_2nuTRPGKmRbjxGuLfW33F-i1tD7_9_WQxsZ1-757aKAtJJnUldk5XewTbedmb2aBNl8P9_2xyOmciseBVzlWPfFPJP3K2nfI6Mne0rdo-2sHIPK55uQ-_QnU5Law6og",
			"https://psv4.vkuseraudio.net/c6187/u190985821/audios/a80d8cf8c3d1.mp3?extra=jTqp0A3ly2T6hxynbuvIDMpWYDL2NY74vzsVzwSftH6A5TQLkPfecuL1sClbTSv09IB0ApSCuQJK6Y2bKGarEzZ4naXoSBKSlQC7wtwvgibMLof4JkGP4B4F-5RpxZau76R4lKLbe0KPiEXDru3v-A&long_chunk=1",
			"test.mp3",
		)
		if err != nil {
			panic(err)
		}
		t2 := time.Now()
		log.Printf("Fetched audio: %d bytes in %.1f ms, %.1f MB/s\n", n, float64(t2.UnixNano()-t1.UnixNano())/float64(1e6), float64(n)/float64(1e6)/(float64(t2.UnixNano()-t1.UnixNano())/float64(1e9)))
	}
}
