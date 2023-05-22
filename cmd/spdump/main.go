// Copy the existing spotify implementation and port to be
// a CLI for dumping all playlists for a user. This will be
// just a JSON dump which can then be piped into the torrent.

package main

import "flag"

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

}
