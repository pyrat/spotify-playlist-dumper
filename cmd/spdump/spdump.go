// Copy the existing spotify implementation and port to be
// a CLI for dumping all playlists for a user. This will be
// just a JSON dump which can then be piped into the torrent.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/pelletier/go-toml"
	"github.com/pyrat/spd/internal/spotify"
	flag "github.com/spf13/pflag"
)

func main() {
	// implement the cli here
	// Define flags
	// playlistPtr := flag.String("playlist", "", "Playlist to dump")
	var playlistPtr *string = flag.StringP("playlist", "p", "3rpdjX0UZGjjmk3A86FrU3", "playlist_id to dump")

	// Parse command line arguments
	flag.Parse()

	// Read the TOML file
	tomlData, err := ioutil.ReadFile("config.toml")
	if err != nil {
		panic(err)
	}

	// Parse the TOML data
	config, err := toml.Load(string(tomlData))
	if err != nil {
		panic(err)
	}

	// Get the Spotify client ID and secret
	clientID := config.Get("spotify.client_id").(string)
	clientSecret := config.Get("spotify.client_secret").(string)

	log.Println("clientID: ", clientID)

	sp, err := spotify.NewSpotify(clientID, clientSecret)
	if err != nil {
		panic(err)
	}

	// Get the user's playlists
	playlist, err := sp.PlaylistFromID(*playlistPtr)
	if err != nil {
		panic(err)
	}

	mp := spotify.ConvertToMusicPlaylist(playlist)

	// Print the playlist
	bytes, _ := json.Marshal(mp)
	fmt.Println(string(bytes))

}
