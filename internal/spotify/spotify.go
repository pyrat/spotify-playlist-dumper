package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/opentracing/opentracing-go/log"
)

// Spotify is the struct to control spotify api interactions.
type Spotify struct {
	Token        string
	ClientID     string
	ClientSecret string
}

type spotifyTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// SpotifyTrackSearchResult Struct for storing API search response.
type SpotifyTrackSearchResult struct {
	ResultCollection   SpotifyTracksResult    `json:"tracks"`
	AlbumCollection    SpotifyAlbumsResult    `json:"albums"`
	PlaylistCollection SpotifyPlaylistsResult `json:"playlists"`
}

// SpotifyTracksResult is just a container struct.
type SpotifyTracksResult struct {
	Items []SpotifyTrack `json:"items"`
}

// SpotifyPlaylistTracks is a container struct for playlist tracks parsing.
type SpotifyPlaylistTracks struct {
	Items []SpotifyPlaylistTrack `json:"items"`
}

// SpotifyPlaylistTrack is a container struct for playlist tracks parsing.
type SpotifyPlaylistTrack struct {
	Track SpotifyTrack `json:"track"`
}

// SpotifyAlbumsResult is also a container struct
type SpotifyAlbumsResult struct {
	Items []SpotifyAlbum `json:"items"`
}

// SpotifyPlaylistsResult is also a container struct
type SpotifyPlaylistsResult struct {
	Items []SpotifyPlaylist `json:"items"`
}

// SpotifyTrack describes a spotify track.
type SpotifyTrack struct {
	Album         SpotifyAlbum       `json:"album"`
	Name          string             `json:"name"`
	PreviewURL    string             `json:"preview_url"`
	TrackURI      string             `json:"uri"`
	IntegrationID string             `json:"id"`
	DurationMS    int                `json:"duration_ms"`
	ExternalURL   SpotifyExternalURL `json:"external_urls"`
	Artists       []SpotifyArtist    `json:"artists"`
}

// ImageURLs Returns a space separated list of image urls in decreasing size.
func (o *SpotifyTrack) ImageURLs() (urls string) {
	for _, image := range o.Album.Images {
		urls += image.URL + " "
	}
	return strings.TrimSpace(urls)
}

// CombineArtists combines the artists to fit in a db field for DB
func (o *SpotifyTrack) CombineArtists() (artists string) {
	var artistNames []string
	for _, artist := range o.Artists {
		artistNames = append(artistNames, artist.Name)
	}
	return strings.Join(artistNames, ", ")
}

// SpotifyAlbum describes a spotify album.
type SpotifyAlbum struct {
	Name             string              `json:"name"`
	Images           []SpotifyAlbumImage `json:"images"`
	URI              string              `json:"uri"`
	ExternalURL      SpotifyExternalURL  `json:"external_urls"`
	IntegrationID    string              `json:"id"`
	ReleaseDate      string              `json:"release_date"`
	Artists          []SpotifyArtist     `json:"artists"`
	TracksCollection SpotifyTracksResult `json:"tracks"`
}

// ImageURLs Returns a space separated list of image urls in decreasing size.
func (o *SpotifyAlbum) ImageURLs() (urls string) {
	for _, image := range o.Images {
		urls += image.URL + " "
	}
	return strings.TrimSpace(urls)
}

// SpotifyAlbumImage describes a spotify album image.
type SpotifyAlbumImage struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	URL    string `json:"url"`
}

// SpotifyPlaylist describes a spotify playlist.
type SpotifyPlaylist struct {
	Name             string                 `json:"name"`
	Images           []SpotifyPlaylistImage `json:"images"`
	URI              string                 `json:"uri"`
	ExternalURL      SpotifyExternalURL     `json:"external_urls"`
	IntegrationID    string                 `json:"id"`
	TracksCollection SpotifyPlaylistTracks  `json:"tracks"`
}

// SpotifyPlaylistImage describes a spotify playlist image.
type SpotifyPlaylistImage struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	URL    string `json:"url"`
}

// SpotifyExternalURL describes a spotify external url.
type SpotifyExternalURL struct {
	Spotify string `json:"spotify"`
}

// SpotifyArtist describes a spotify artist.
type SpotifyArtist struct {
	Name          string `json:"name"`
	IntegrationID string `json:"id"`
}

// MusicSearchResult is a generic struct
type MusicSearchResult struct {
	Tracks    []MusicTrack
	Albums    []MusicAlbum
	Playlists []MusicPlaylist
}

// MusicTrack stores the spotify result in a format which can be easily Marshaled.
type MusicTrack struct {
	Name             string
	PreviewURL       string
	AlbumName        string
	AlbumArt         []SpotifyAlbumImage
	AlbumReleaseDate string
	IntegrationID    string
	Source           string
	ExternalURL      string
	Artists          []MusicArtist `json:",omitempty"`
}

// MusicAlbum stores details of Albums for further browsing.
type MusicAlbum struct {
	Name          string
	AlbumArt      []SpotifyAlbumImage
	ReleaseDate   string
	Artists       []MusicArtist `json:",omitempty"`
	Tracks        []MusicTrack  `json:",omitempty"`
	IntegrationID string
}

// MusicPlaylist stores details of Playlist for further browsing.
type MusicPlaylist struct {
	Name          string
	PlaylistArt   []SpotifyPlaylistImage
	Tracks        []MusicTrack `json:",omitempty"`
	IntegrationID string
}

// MusicArtist describes a music artist in a generic way.
type MusicArtist struct {
	Name          string
	IntegrationID string
}

// NewMusicSearchResultFromSpotify converts a spotify result to music search result
func NewMusicSearchResultFromSpotify(spotifyResult SpotifyTrackSearchResult) *MusicSearchResult {
	msr := &MusicSearchResult{}

	for _, track := range spotifyResult.ResultCollection.Items {
		mt := MusicTrack{
			Name:             track.Name,
			PreviewURL:       track.PreviewURL,
			AlbumName:        track.Album.Name,
			AlbumArt:         track.Album.Images,
			AlbumReleaseDate: track.Album.ReleaseDate,
			IntegrationID:    track.IntegrationID,
			Source:           "spotify",
			ExternalURL:      track.ExternalURL.Spotify,
		}

		for _, artist := range track.Album.Artists {
			a := MusicArtist{
				Name:          artist.Name,
				IntegrationID: artist.IntegrationID,
			}
			mt.Artists = append(mt.Artists, a)
		}
		msr.Tracks = append(msr.Tracks, mt)
	}

	// Add albums and artists of that album.
	for _, album := range spotifyResult.AlbumCollection.Items {

		ma := MusicAlbum{
			Name:          album.Name,
			AlbumArt:      album.Images,
			ReleaseDate:   album.ReleaseDate,
			IntegrationID: album.IntegrationID,
		}

		for _, artist := range album.Artists {
			a := MusicArtist{
				Name:          artist.Name,
				IntegrationID: artist.IntegrationID,
			}
			ma.Artists = append(ma.Artists, a)
		}
		// need to add artists in a loop here.
		// wondering whether to add
		msr.Albums = append(msr.Albums, ma)
	}

	// Add playlists.
	for _, playlist := range spotifyResult.PlaylistCollection.Items {

		mp := MusicPlaylist{
			Name:          playlist.Name,
			PlaylistArt:   playlist.Images,
			IntegrationID: playlist.IntegrationID,
		}
		msr.Playlists = append(msr.Playlists, mp)
	}

	return msr
}

// NewSpotify initialises a Spotify API struct. This requests a access token if
// it does not current have a valid token cached.
func NewSpotify(clientID string, clientSecret string) (*Spotify, error) {
	sp := &Spotify{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	token, err := sp.getToken()

	if err != nil {
		log.Error("Unable to get token for API access.", err)
		return nil, err
	}

	sp.Token = token
	return sp, nil
}

// getToken gets the token for Spotify API access.
// Sets it with an expiry of 55 minutes in redis. (Tokens are typically valid for 60 minutes)
func (o *Spotify) getToken() (string, error) {
	if len(o.Token) > 0 {
		return o.Token, nil
	}
	return o.refreshSpotifyToken()
}

// refreshSpotifyToken hits spotify API to get a new token and
// stores it in redis with a 55 minute expiry if successful.
func (o *Spotify) refreshSpotifyToken() (string, error) {

	// get body
	body := url.Values{}
	body.Set("grant_type", "client_credentials")
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(body.Encode()))
	if err != nil {
		log.Error("net/http error")
		return "", err
	}

	req.SetBasicAuth(o.ClientID, o.ClientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		log.Error("Error hitting spotify to refresh token")
		return "", errors.New("spotify token error")
	}

	defer resp.Body.Close()
	respbody, _ := ioutil.ReadAll(resp.Body)
	spotTokenResp := spotifyTokenResponse{}
	err = json.Unmarshal(respbody, &spotTokenResp)

	if spotTokenResp.AccessToken == "" {
		errmsg := "Problems getting spotify access token from JSON"
		log.Error(errmsg, spotTokenResp)
		return "", errors.New(errmsg)
	}

	o.Token = spotTokenResp.AccessToken
	return spotTokenResp.AccessToken, nil
}

func (o *Spotify) getSearchResultsFromCache(query string) ([]byte, error) {
	val, err := o.Cache.Get("spotify:search:" + query).Result()
	if err == redis.Nil {
		return []byte{}, nil
	} else if err != nil {
		log.Error("Error hitting redis for spotify token" + err.Error())
		return []byte{}, err
	}
	return []byte(val), nil
}

// Search makes a call to the Spotify search endpoint returning a struct which implements
// the TrackSearchResult interface.
func (o *Spotify) Search(query string, market string) (SpotifyTrackSearchResult, error) {
	searchResult := SpotifyTrackSearchResult{}
	var cached bool

	if len(market) == 0 {
		market = "US"
	}

	respbody, err := o.getSearchResultsFromCache(market + "/" + query)
	if err != nil {
		return searchResult, err
	}

	// Hit Spotify if there was a cache miss.
	if len(respbody) == 0 {
		searchURL := "https://api.spotify.com/v1/search"
		params := url.Values{}
		params.Add("q", query)
		params.Add("type", "track,album,playlist")
		params.Add("market", market)
		params.Add("limit", "50")

		client := &http.Client{}
		req, err := http.NewRequest("GET", searchURL+"?"+params.Encode(), nil)
		if err != nil {
			log.Error("net/http error")
			return searchResult, err
		}

		// Always get the token before making the request
		// to avoid making a request with an expired token.
		token, err := o.getToken()
		if err != nil {
			log.Error("error getting token")
			return searchResult, err
		}

		req.Header.Add("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			log.Error("Error making call to spotify error:", err)
			return searchResult, errors.New("error making call to spotify")
		}

		defer resp.Body.Close()
		respbody, _ = ioutil.ReadAll(resp.Body)

		if resp.StatusCode != 200 {
			log.Error("Error making call to spotify", string(respbody[:]))
			return searchResult, errors.New("error making call to spotify")
		}

		// store raw results in the cache
		err = o.Cache.Set("spotify:search:"+market+"/"+query, respbody, 7*24*time.Hour).Err()
		if err != nil {
			log.Error("Error caching spotify search result.")
			return searchResult, err
		}
	} else {
		// set cached to true. dont generally like flags but easy in this case.
		cached = true
	}

	err = json.Unmarshal(respbody, &searchResult)

	if err != nil {
		log.Error("Invalid JSON response from Spotify", err)
		return searchResult, err
	}

	if cached == true {
		// Let the client know it is a cached result.
		searchResult.Cached = true
	}

	// All is well.
	return searchResult, nil
}

// TrackFromID hits the Spotify API to get Track information.
func (o *Spotify) TrackFromID(ID string) (SpotifyTrack, error) {
	st := SpotifyTrack{}

	trackURL := "https://api.spotify.com/v1/tracks/" + ID

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", trackURL, nil)
	if err != nil {
		log.Error("net/http error")
		return st, err
	}

	// Always get the token before making the request
	// to avoid making a request with an expired token.
	token, err := o.getToken()
	if err != nil {
		log.Error("error getting token")
		return st, err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error making call to spotify error:", err)
		return st, fmt.Errorf("error making call to spotify to get track information : %s", ID)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Error("Error making call to spotify", string(body[:]))
		return st, fmt.Errorf("error making call to spotify to get track information : %s", ID)
	}

	// load the response into the required object,
	// translate to a music track also required
	err = json.Unmarshal(body, &st)
	if err != nil {
		log.Error("Invalid JSON response from Spotify", err)
		return st, err
	}

	return st, nil
}

// AlbumFromID hits the Spotify API to get Album information.
func (o *Spotify) AlbumFromID(ID string) (SpotifyAlbum, error) {
	album := SpotifyAlbum{}

	trackURL := "https://api.spotify.com/v1/albums/" + ID

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest("GET", trackURL, nil)
	if err != nil {
		log.Error("net/http error")
		return album, err
	}

	// Always get the token before making the request
	// to avoid making a request with an expired token.
	token, err := o.getToken()
	if err != nil {
		log.Error("error getting token")
		return album, err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error making call to spotify error:", err)
		return album, errors.New("error making call to spotify to get album information")
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Error("Error making call to spotify", string(body[:]))
		return album, errors.New("error making call to spotify to get album information")
	}

	// load the response into the required object,
	err = json.Unmarshal(body, &album)
	if err != nil {
		log.Error("Invalid JSON response from Spotify", err)
		return album, err
	}

	return album, nil
}

// PlaylistFromID hits the Spotify API to get Playlist information.
func (o *Spotify) PlaylistFromID(ID string) (SpotifyPlaylist, error) {
	playlist := SpotifyPlaylist{}

	trackURL := "https://api.spotify.com/v1/playlists/" + ID

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest("GET", trackURL, nil)
	if err != nil {
		log.Error("net/http error")
		return playlist, err
	}

	// Always get the token before making the request
	// to avoid making a request with an expired token.
	token, err := o.getToken()
	if err != nil {
		log.Error("error getting token")
		return playlist, err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Error making call to spotify error:", err)
		return playlist, errors.New("error making call to spotify to get album information")
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		log.Error("Error making call to spotify", string(body[:]))
		return playlist, errors.New("error making call to spotify to get album information")
	}

	// load the response into the required object,
	err = json.Unmarshal(body, &playlist)
	if err != nil {
		log.Error("Invalid JSON response from Spotify", err)
		return playlist, err
	}

	return playlist, nil
}

// ConvertToMusicAlbum converts a SpotifyAlbum struct to a MusicAlbum struct
func ConvertToMusicAlbum(sa SpotifyAlbum) MusicAlbum {
	ma := MusicAlbum{
		Name:          sa.Name,
		IntegrationID: sa.IntegrationID,
		AlbumArt:      sa.Images,
		ReleaseDate:   sa.ReleaseDate,
	}

	ma.Artists = []MusicArtist{}

	for _, artist := range sa.Artists {
		musicArtist := MusicArtist{
			Name:          artist.Name,
			IntegrationID: artist.IntegrationID,
		}
		ma.Artists = append(ma.Artists, musicArtist)
	}

	if len(sa.TracksCollection.Items) > 0 {
		for _, track := range sa.TracksCollection.Items {
			track.Album = sa
			ma.Tracks = append(ma.Tracks, ConvertToMusicTrack(track))
		}
	}

	return ma
}

// ConvertToMusicPlaylist converts a SpotifyPlaylist struct to a MusicPlaylist struct
func ConvertToMusicPlaylist(sp SpotifyPlaylist) MusicPlaylist {
	playlist := MusicPlaylist{
		Name:          sp.Name,
		IntegrationID: sp.IntegrationID,
		PlaylistArt:   sp.Images,
	}

	if len(sp.TracksCollection.Items) > 0 {
		for _, track := range sp.TracksCollection.Items {
			playlist.Tracks = append(playlist.Tracks, ConvertToMusicTrack(track.Track))
		}
	}

	return playlist
}

// ConvertToMusicTrack converts a SpotifyTrack struct to a MusicTrack struct
func ConvertToMusicTrack(st SpotifyTrack) MusicTrack {
	musicTrack := MusicTrack{
		Name:             st.Name,
		PreviewURL:       st.PreviewURL,
		AlbumName:        st.Album.Name,
		AlbumArt:         st.Album.Images,
		AlbumReleaseDate: st.Album.ReleaseDate,
		IntegrationID:    st.IntegrationID,
		Source:           "spotify",
		ExternalURL:      st.ExternalURL.Spotify,
	}
	musicTrack.Artists = []MusicArtist{}

	for _, artist := range st.Artists {
		musicArtist := MusicArtist{
			Name:          artist.Name,
			IntegrationID: artist.IntegrationID,
		}
		musicTrack.Artists = append(musicTrack.Artists, musicArtist)
	}

	return musicTrack
}