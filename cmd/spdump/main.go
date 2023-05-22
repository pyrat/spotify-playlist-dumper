// Copy the existing spotify implementation and port to be
// a CLI for dumping all playlists for a user. This will be
// just a JSON dump which can then be piped into the torrent.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/pelletier/go-toml"
	"github.com/pyrat/spd/internal/spotify"
)

func main() {
	// implement the cli here
	// Define flags
	userPtr := flag.String("user", "", "User to dump playlists for")
	playlistPtr := flag.String("playlist", "", "Playlist to dump")

	// Parse command line arguments
	flag.Parse()

	// Access parsed values
	user := *inputPtr
	playlist := *playlistPtr

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

	sp, err := spotify.NewSpotify(clientID, clientSecret)
	if err != nil {
		panic(err)
	}

	// Get the user's playlists
	playlists, err := sp.MyPlaylists()
	if err != nil {
		panic(err)
	}

	// Print the playlists
	for _, playlist := range playlists {
		fmt.Println(playlist.Name)
	}

}
