// Simple, sexy and easy golang module for TheTVDB.
package tvdb

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	// TheTVDB API key.
	API_KEY = "DECE3B6B5464C552"

	// Default language
	DEFAULT_LANGUAGE = "en"

	// Protocol and Host Prefix
	HOST = "http://thetvdb.com"

	// URL used to get basic series information by name.
	GET_SERIES_URL = HOST + "/api/GetSeries.php?seriesname=%v"

	// URL used to get basic series information by ID.
	GET_SERIES_BY_ID_URL = HOST + "/api/%v/series/%v/%v.xml"

	// URL used to get detailed series/episode information by ID.
	GET_DETAIL_URL = HOST + "/api/%v/series/%v/all/%v.xml"

	// URL used for series web searches.
	SEARCH_SERIES_URL = HOST + "/?string=%v&searchseriesid=&tab=listseries&function=Search"

	// URL used for series web search matching.
	SEARCH_SERIES_REGEX = `(?P<before><a href="/\?tab=series&amp;id=)(?P<seriesId>\d+)(?P<after>\&amp;lid=\d*">)`
)

// Regex used for series web search matching.
var SearchSeriesRegex = regexp.MustCompile(SEARCH_SERIES_REGEX)

// Type representing pipe-separated string values.
type PipeList []string

// Unmarshal an XML element with string value into a pip-separated list of strings.
func (pipeList *PipeList) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) (err error) {
	content := ""

	if err = decoder.DecodeElement(&content, &start); err != nil {
		return err
	}

	*pipeList = strings.Split(strings.Trim(content, "|"), "|")

	return
}

// Episode represents a TV show episode on TheTVDB.
type Episode struct {
	Id                    uint64   `xml:"id"`
	CombinedEpisodeNumber string   `xml:"Combined_episodenumber"`
	CombinedSeason        uint64   `xml:"Combined_season"`
	DvdChapter            string   `xml:"DVD_chapter"`
	DvdDiscId             string   `xml:"DVD_discid"`
	DvdEpisodeNumber      string   `xml:"DVD_episodenumber"`
	DvdSeason             string   `xml:"DVD_season"`
	Director              PipeList `xml:"Director"`
	EpImgFlag             string   `xml:"EpImgFlag"`
	EpisodeName           string   `xml:"EpisodeName"`
	EpisodeNumber         uint64   `xml:"EpisodeNumber"`
	FirstAired            string   `xml:"FirstAired"`
	GuestStars            string   `xml:"GuestStars"`
	ImdbId                string   `xml:"IMDB_ID"`
	Language              string   `xml:"Language"`
	Overview              string   `xml:"Overview"`
	ProductionCode        string   `xml:"ProductionCode"`
	Rating                string   `xml:"Rating"`
	RatingCount           string   `xml:"RatingCount"`
	SeasonNumber          uint64   `xml:"SeasonNumber"`
	Writer                PipeList `xml:"Writer"`
	AbsoluteNumber        string   `xml:"absolute_number"`
	Filename              string   `xml:"filename"`
	LastUpdated           string   `xml:"lastupdated"`
	SeasonId              uint64   `xml:"seasonid"`
	SeriesId              uint64   `xml:"seriesid"`
	ThumbAdded            string   `xml:"thumb_added"`
	ThumbHeight           string   `xml:"thumb_height"`
	ThumbWidth            string   `xml:"thumb_width"`
}

// Series represents TV show on TheTVDB.
type Series struct {
	Id            uint64   `xml:"id"`
	Actors        PipeList `xml:"Actors"`
	AirsDayOfWeek string   `xml:"Airs_DayOfWeek"`
	AirsTime      string   `xml:"Airs_Time"`
	ContentRating string   `xml:"ContentRating"`
	FirstAired    string   `xml:"FirstAired"`
	Genre         PipeList `xml:"Genre"`
	ImdbId        string   `xml:"IMDB_ID"`
	Language      string   `xml:"Language"`
	Network       string   `xml:"Network"`
	NetworkId     string   `xml:"NetworkID"`
	Overview      string   `xml:"Overview"`
	Rating        string   `xml:"Rating"`
	RatingCount   string   `xml:"RatingCount"`
	Runtime       string   `xml:"Runtime"`
	SeriesId      string   `xml:"SeriesID"`
	SeriesName    string   `xml:"SeriesName"`
	Status        string   `xml:"Status"`
	Added         string   `xml:"added"`
	AddedBy       string   `xml:"addedBy"`
	Banner        string   `xml:"banner"`
	Fanart        string   `xml:"fanart"`
	LastUpdated   string   `xml:"lastupdated"`
	Poster        string   `xml:"poster"`
	Zap2ItId      string   `xml:"zap2it_id"`
	Seasons       map[uint64][]Episode
}

// SeriesList represents a list of TV shows, often used for returning search results.
type SeriesList struct {
	Series []Series `xml:"Series"`
}

// EpisodeList represents a list of TV show episodes.
type EpisodeList struct {
	Episodes []Episode `xml:"Episode"`
}

// Create an initialise a new TV series object from XML data.
func NewSeries(data []byte) (*Series, error) {
	series := Series{}

	series.Seasons = make(map[uint64][]Episode)

	if err := xml.Unmarshal(data, &series); err != nil {
		return nil, err
	}

	return &series, nil
}

// Tvdb query handler
type Tvdb struct {
	ApiKey   string
	Language string
}

// Tvdb query configuration
type TvdbConfig struct {
	ApiKey   string
	Language string
}

// Tvdb with a configuration
func NewTvdbWithConfig(config TvdbConfig) *Tvdb {
	return &Tvdb{config.ApiKey, config.Language}
}

// Tvdb with default configuration
func NewTvdb() *Tvdb {
	return NewTvdbWithConfig(TvdbConfig{API_KEY, DEFAULT_LANGUAGE})
}

// Get more detail for all TV shows in a list.
func (seriesList *SeriesList) GetDetail(config TvdbConfig) (err error) {
	for seriesIndex := range seriesList.Series {
		if err = seriesList.Series[seriesIndex].GetDetail(config); err != nil {
			return
		}
	}

	return
}

// Get more detail for a TV show, including information on it's episodes.
func (series *Series) GetDetail(config TvdbConfig) (err error) {
	response, err := http.Get(fmt.Sprintf(GET_DETAIL_URL, config.ApiKey, strconv.FormatUint(series.Id, 10), config.Language))

	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return
	}

	if err = xml.Unmarshal(data, series); err != nil {
		return
	}

	episodeList := EpisodeList{}

	if err = xml.Unmarshal(data, &episodeList); err != nil {
		return
	}

	if series.Seasons == nil {
		series.Seasons = make(map[uint64][]Episode)
	}

	for _, episode := range episodeList.Episodes {
		series.Seasons[episode.SeasonNumber] = append(series.Seasons[episode.SeasonNumber], episode)
	}

	return
}

// Get a list of TV series by name, by performing a simple search.
func GetSeries(name string) (seriesList SeriesList, err error) {
	response, err := http.Get(fmt.Sprintf(GET_SERIES_URL, url.QueryEscape(name)))

	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return
	}

	err = xml.Unmarshal(data, &seriesList)

	return
}

// Get a TV series by ID.
func (mytvdb *Tvdb) GetSeriesById(id uint64) (series Series, err error) {
	response, err := http.Get(fmt.Sprintf(GET_SERIES_BY_ID_URL, mytvdb.ApiKey, id, mytvdb.Language))

	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return
	}

	seriesList := SeriesList{}

	if err = xml.Unmarshal(data, &seriesList); err != nil {
		return
	}

	if len(seriesList.Series) != 1 {
		err = errors.New("incorrect number of series")

		return
	}

	series = seriesList.Series[0]

	return
}

// Search for TV shows by name, using the more sophisticated search on TheTVDB's homepage.
// This is the recommended search method.
func (mytvdb *Tvdb) SearchSeries(name string, maxResults int) (seriesList SeriesList, err error) {
	response, err := http.Get(fmt.Sprintf(SEARCH_SERIES_URL, url.QueryEscape(name)))

	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return
	}

	groups := SearchSeriesRegex.FindAllSubmatch(buf, -1)
	doneSeriesIds := make(map[uint64]struct{})

	for _, group := range groups {
		seriesId := uint64(0)
		series := Series{}
		seriesId, err = strconv.ParseUint(string(group[2]), 10, 64)

		if _, ok := doneSeriesIds[seriesId]; ok {
			continue
		}

		if err != nil {
			return
		}

		series, err = mytvdb.GetSeriesById(seriesId)

		if err != nil {
			// Some series can't be found, so we will ignore these.
			if _, ok := err.(*xml.SyntaxError); ok {
				err = nil
				continue
			} else {
				return
			}
		}

		seriesList.Series = append(seriesList.Series, series)
		doneSeriesIds[seriesId] = struct{}{}

		if len(seriesList.Series) == maxResults {
			break
		}
	}

	return
}
