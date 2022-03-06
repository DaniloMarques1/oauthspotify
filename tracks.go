package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// recently played tracks response
type Response struct {
	Items []Item `json:"items"`
}

type Item struct {
	Track     Track  `json:"track"`
	Played_at string `json:"played_at"`
}

type Track struct {
	Album Album  `json:"album"`
	Name  string `json:"name"`
}

type Album struct {
	Artists []Artist `json:"artists"`
	Name    string   `json:"name"`
}

type Artist struct {
	Name string `json:"name"`
}

// loops over the response from spotify and shows
// recently played tracks
func showTracks(bytes []byte) {
	var response Response
	if err := json.Unmarshal(bytes, &response); err != nil {
		log.Fatal(err)
	}

	itens := response.Items
	var size int
	if len(itens) < 10 {
		size = len(itens)
	} else {
		size = 10
	}
	for i := 0; i < size; i++ {
		item := itens[i]
		dt, err := time.Parse("2006-01-02T15:04:05.000Z", item.Played_at)
		loc, err := time.LoadLocation("America/Recife")
		if err != nil {
			log.Fatalf("Error loading location %v", err)
		}
		fmt.Printf("Music name: %s\n", item.Track.Name)
		fmt.Printf("Album name: %s\n", item.Track.Album.Name)
		if err == nil {
			dtStr := dt.In(loc).Format(time.UnixDate)
			fmt.Printf("Played at: %v\n", dtStr)
		}
		fmt.Println("Artits (just the first 2):")
		artits := item.Track.Album.Artists
		var artistsSize int
		if len(artits) < 2 {
			artistsSize = len(artits)
		} else {
			artistsSize = 2
		}
		for j := 0; j < artistsSize; j++ {
			fmt.Printf("   %d- %v\n", j+1, artits[j].Name)
		}
		fmt.Println("==================")
		fmt.Println()
	}
}
