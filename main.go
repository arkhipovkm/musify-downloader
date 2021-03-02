package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/arkhipovkm/id3-go"
	"github.com/arkhipovkm/musify/download"
	"github.com/arkhipovkm/musify/utils"
	"github.com/arkhipovkm/musify/vk"
)

var BASE_PATH string = ""

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

func downloadAudio(i int, audio *vk.Audio, album *vk.Playlist, dirname string, apicCoverData, apicIconData []byte) error {
	var err error
	if audio.URL == "" {
		return err
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
		return err
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
		return err
	}
	defer func() {
		err = id3File.Close()
		if err != nil {
			log.Println(err)
		}
	}()
	utils.SetID3TagAPICs(id3File, apicCoverData, apicIconData)
	utils.SetID3Tag(
		id3File,
		album.AuthorName, // audio.Performer,
		audio.Title,
		album.Title,
		album.YearInfoStr,
		trck,
	)
	// Write ID3 tags to file

	return err
}

func main() {

	playlistsFilePath := os.Args[1]
	BASE_PATH = os.Args[2]

	vkUser := vk.NewDefaultUser()
	err := vkUser.Authenticate("", "")
	if err != nil {
		log.Printf("%#v\n", vkUser)
		panic(err)
	}
	body, err := ioutil.ReadFile(playlistsFilePath)
	if err != nil {
		panic(err)
	}
	var failedAlbums []string
	for _, albumID := range strings.Split(string(body), "\r\n") {
		if albumID[0] == '#' {
			continue
		}
		playlist := vk.LoadPlaylist(albumID, vkUser)
		if playlist == nil {
			log.Println("Nil Playlist. Continuing..")
			continue
		}
		dirname := filepath.Join(
			BASE_PATH,
			escapeWindowsPath(playlist.AuthorName),
			escapeWindowsPath(playlist.Title),
		)
		log.Println("Starting download ", dirname)
		err = os.MkdirAll(
			dirname,
			os.ModePerm,
		)
		if err != nil {
			panic(err)
		}
		playlist.AcquireURLs(vkUser)
		playlist.DecypherURLs(vkUser)
		playlist.CoverURL = ""
		apicCoverData, apicIconData := download.DownloadAPICs(playlist.List[0], playlist)
		log.Println("Downloaded APICs: ", len(apicCoverData), len(apicIconData))
		for i, audio := range playlist.List {
			err = downloadAudio(i, audio, playlist, dirname, apicCoverData, apicIconData)
			if err != nil {
				log.Println(err)
				failedAlbums = append(failedAlbums, albumID)
				break
			}
		}
		log.Println("Finished download ", dirname)
	}
	log.Println(failedAlbums)
}
