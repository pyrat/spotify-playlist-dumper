# Spotify Playlist Dumper

This CLI dumps all the songs from a Spotify playlist into a JSON file.

## Installation

```bash
go install github.com/pyrat/spotify-playlist-dumper@latest
```

Also you need to copy the config.toml.example to config.toml and fill in the client id and client secret from the Spotify developer dashboard.

When you run the spdump command do it in the same directory as the config.toml file.

## Usage

```bash 
spdump -playlist <playlist_id> > playlist.json
```
