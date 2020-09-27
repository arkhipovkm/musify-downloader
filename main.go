package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/arkhipovkm/id3-go"
	"github.com/arkhipovkm/musify/download"
	"github.com/arkhipovkm/musify/utils"
	"github.com/arkhipovkm/musify/vk"
)

func httpGET(uri string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func httpGETChan(uri string, dataChan chan []byte, errChan chan error) {
	data, err := httpGET(uri)
	dataChan <- data
	errChan <- err
}

func escapeWindowsPath(s string) string {
	forbidden := []string{
		"<",  // (less than)
		">",  // (greater than)
		":",  // (colon - sometimes works, but is actually NTFS Alternate Data Streams)
		"\"", // (double quote)
		"/",  // (forward slash)
		"\\", // (backslash)
		"|",  // (vertical bar or pipe)
		"?",  // (question mark)
		"*",  // (asterisk)
	}
	for _, char := range forbidden {
		s = strings.ReplaceAll(s, char, "_")
	}
	return s
}

func downloadAudioChan(i int, audio *vk.Audio, album *vk.Playlist, dirname string, apicCoverData, apicIconData []byte, errChan chan error) {
	var err error
	if audio.URL == "" {
		errChan <- err
		return
	}
	base := fmt.Sprintf("%02d", i+1) + " — " + escapeWindowsPath(audio.Performer) + " — " + escapeWindowsPath(audio.Title)
	if len([]rune(dirname))+len([]rune(base)) > 200 {
		base = string([]rune(base)[:200-len([]rune(dirname))])
	}
	filename := filepath.Join(
		dirname,
		base+".mp3",
	)
	// Download audio file
	if strings.Contains(audio.URL, ".m3u8") {
		re := regexp.MustCompile("/[0-9a-f]+(/audios)?/([0-9a-f]+)/index.m3u8")
		audio.URL = re.ReplaceAllString(audio.URL, "$1/$2.mp3")
		_, _, err = download.MP3File(audio.URL, filename)
	} else if strings.Contains(audio.URL, ".mp3") {
		_, _, err = download.MP3File(audio.URL, filename)
	} else {
		err = fmt.Errorf("Unsupported file type: %s", filepath.Base(filepath.Dir(audio.URL)))
		errChan <- err
		return
	}
	// End Download audio file

	// Handle trck tag
	var trck string
	if album != nil {
		trck = strconv.Itoa(i+1) + "/" + strconv.Itoa(album.TotalCount)
	}
	// End Handle trck tag

	// Write ID3 tags to file
	id3File, err := id3.Open(filename)
	if err != nil {
		errChan <- err
		return
	}
	defer id3File.Close()
	utils.SetID3Tag(
		id3File,
		album.AuthorName, // audio.Performer,
		audio.Title,
		album.Title,
		album.YearInfoStr,
		trck,
	)
	utils.SetID3TagAPICs(id3File, apicCoverData, apicIconData)
	id3File.Close()
	// Write ID3 tags to file

	errChan <- err
	return
}

func downloadApics(audio *vk.Audio, album *vk.Playlist) ([]byte, []byte) {
	// Handle APICs tags: cover and icon
	apicErrChan := make(chan error, 2)
	apicDataChan := make(chan []byte, 2)
	var apicCoverData, apicIconData []byte

	var apicCover string
	if album != nil && album.CoverURL != "" {
		apicCover = album.CoverURL
	} else {
		apicCover = audio.CoverURLp
	}

	if audio.CoverURLp != "" {
		go httpGETChan(apicCover, apicDataChan, apicErrChan)
	} else {
		apicErrChan <- nil
		apicDataChan <- nil
	}
	if audio.CoverURLs != "" {
		go httpGETChan(audio.CoverURLs, apicDataChan, apicErrChan)
	} else {
		apicErrChan <- nil
		apicDataChan <- nil
	}

	for i := 0; i < 2; i++ {
		err := <-apicErrChan
		if err != nil {
			log.Println("Error loading APICs: ", err)
		}
	}

	apic0 := <-apicDataChan
	apic1 := <-apicDataChan

	if len(apic0) < len(apic1) {
		apicCoverData = apic1
		apicIconData = apic0
	} else {
		apicCoverData = apic0
		apicIconData = apic1
	}
	// End Handle APICs tags: cover and icon
	return apicCoverData, apicIconData
}

func main() {
	vkUser := vk.NewDefaultUser()
	err := vkUser.Authenticate()
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadFile("playlists.txt")
	if err != nil {
		panic(err)
	}
	var albumIDs []string
	for _, uri := range strings.Split(string(body), "\r\n") {
		if uri != "" {
			parsedURI, err := url.Parse(uri)
			if err != nil {
				panic(err)
			}
			albumID := filepath.Base(parsedURI.Path)
			if albumID != "." {
				albumIDs = append(albumIDs, albumID)
			}
		}
	}
	for _, albumID := range albumIDs {
		playlist := vk.LoadPlaylist(albumID, vkUser)

		errChan := make(chan error, len(playlist.List))
		dirname := filepath.Join(
			"D:",
			"Musify",
			escapeWindowsPath(playlist.AuthorName),
			escapeWindowsPath(playlist.Title),
		)
		if _, err := os.Stat(dirname); os.IsNotExist(err) {
			log.Println("Starting download ", dirname)
			err = os.MkdirAll(
				dirname,
				os.ModePerm,
			)
			if err != nil {
				os.RemoveAll(dirname)
				panic(err)
			}
			err = playlist.AcquireURLs(vkUser)
			if err != nil {
				os.RemoveAll(dirname)
				panic(err)
			}
			playlist.DecypherURLs(vkUser)
			apicCoverData, apicIconData := downloadApics(playlist.List[0], playlist)
			log.Println("Downloaded APICs: ", len(apicCoverData), len(apicIconData))
			for i, audio := range playlist.List {
				go downloadAudioChan(i, audio, playlist, dirname, apicCoverData, apicIconData, errChan)
			}
			for i := 0; i < len(playlist.List); i++ {
				err := <-errChan
				if err != nil {
					os.RemoveAll(dirname)
					panic(err)
				}
			}
			log.Println("Finished download ", dirname)
		} else {
			log.Println(dirname, "already exists. Skipping..")
		}
	}
}
