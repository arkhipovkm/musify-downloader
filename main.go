package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arkhipovkm/id3-go"
	"github.com/arkhipovkm/musify/download"
	"github.com/arkhipovkm/musify/utils"
	"github.com/arkhipovkm/musify/vk"
)

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
	dirnameRune := []rune(dirname)
	baseRune := []rune(base)
	if len(dirnameRune)+len(baseRune) > 200 {
		base = string(baseRune)[:200-len(dirnameRune)]
	}
	filename := filepath.Join(
		dirname,
		base+".mp3",
	)
	err = download.Download(audio, filename)
	if err != nil {
		errChan <- err
		return
	}
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
		if playlist == nil {
			log.Println("Nil Playlist. Continuing..")
			continue
		}
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
			apicCoverData, apicIconData := download.DownloadAPICs(playlist.List[0], playlist)
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
