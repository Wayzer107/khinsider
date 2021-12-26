package scrape

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pterm/pterm"

	"github.com/marcus-crane/khinsider/pkg/types"
)

const (
	AlbumBase  = "https://downloads.khinsider.com/game-soundtracks/album/"
	LetterBase = "https://downloads.khinsider.com/game-soundtracks/browse/"
)

func DownloadPage(url string) (*http.Response, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, err
	}
	return res, err
}

func GetResultsForLetter(letter string) (types.SearchResults, error) {
	url := fmt.Sprintf("%s%s", LetterBase, letter)
	res, err := DownloadPage(url)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(res.Body)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	results := make(types.SearchResults)
	doc.Find("#EchoTopic p[align='left'] a").Each(func(i int, s *goquery.Selection) {
		title := s.Text()
		results[title] = "#"
		trackUrl, exists := s.Attr("href")
		if exists {
			results[title] = trackUrl
		} else {
			results[title] = "#"
		}
	})
	return results, nil
}

func DownloadAlbum(slug string) (types.Album, error) {
	var album types.Album
	albumUrl := fmt.Sprintf("%s%s", AlbumBase, slug)
	res, err := DownloadPage(albumUrl)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(res.Body)
	if err != nil {
		return album, err
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return album, err
	}
	metadata := doc.Find("#EchoTopic p[align='left'] b")
	if metadata.Length() == 5 {
		album.FlacAvailable = true
	}
	metadata.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			album.Name = s.Text()
		}
		if i == 1 {
			album.FileCount, err = strconv.Atoi(s.Text())
			if err != nil {
				album.FileCount = 0
			}
		}
		if i == 2 {
			album.MP3FileSize = s.Text()
		}
		if i == 3 && album.FlacAvailable {
			album.FlacFileSize = s.Text()
		}
	})
	flacLabel := ""
	if album.FlacAvailable {
		flacLabel = "[FLAC]"
	}
	pterm.Info.Printfln(
		"Found %s (%d tracks) %s %s",
		album.Name,
		album.FileCount,
		"[MP3]",
		flacLabel)
	_ = doc.Find("#EchoTopic table:not(#songList) tr div a").Each(func(i int, s *goquery.Selection) {
		coverUrl, exists := s.Attr("href")
		if exists {
			// TODO: Use proper escaping. Tried stdlib but it escaped everything
			coverUrl = strings.ReplaceAll(coverUrl, " ", "%20")
			album.Covers = append(album.Covers, coverUrl)
		}
	})
	songMeta := make(map[int]string)
	_ = doc.Find("#songlist_header th").Each(func(i int, s *goquery.Selection) {
		header := strings.TrimSpace(s.Text())
		if header == "CD" {
			songMeta[i] = "CD"
		}
		if header == "#" {
			songMeta[i] = "TrackNumber"
		}
		if header == "Song Name" {
			songMeta[i] = "SongName"
		}
		if header == "MP3" {
			songMeta[i] = "TrackLength"
			songMeta[i+1] = "MP3FileSize"
		}
		if header == "FLAC" {
			songMeta[i+1] = "FlacFileSize"
		}
	})
	fmt.Println(songMeta)
	doc.Find("#songlist tr:not(#songlist_header, #songlist_footer)").Each(func(i int, s *goquery.Selection) {
		var track types.Track
		s.Find("td").Each(func(i int, s *goquery.Selection) {
			if songMeta[i] == "CD" {
				track.CDNumber = strings.TrimSpace(s.Text())
			}
			if songMeta[i] == "TrackNumber" {
				track.Number = strings.TrimSpace(s.Text())
			}
			if songMeta[i] == "SongName" {
				track.Name = strings.TrimSpace(s.Text())
			}
			if songMeta[i] == "TrackLength" {
				track.Duration = strings.TrimSpace(s.Text())
			}
			if songMeta[i] == "MP3FileSize" {
				track.MP3FileSize = strings.TrimSpace(s.Text())
			}
			if songMeta[i] == "FlacFileSize" {
				track.FlacFileSize = strings.TrimSpace(s.Text())
			}
		})
		fmt.Println(track)
		album.Tracks = append(album.Tracks, track)
	})
	return album, nil
}
