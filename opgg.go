package main

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type leagueMostChamps struct {
	imageURL    string
	gamesPlayed string
	winRate     string
	kda         string
}

func opggRankedChamps(summonerID, region string) []leagueMostChamps {
	var champs []leagueMostChamps
	var champ *leagueMostChamps
	response, err := http.Get("http://" + region + ".op.gg/summoner/ajax/championMost/renew.json/summonerId=" + summonerID)
	if err != nil {
		fmt.Println("Error refreshing summoner info:", err)
		return champs
	}
	response.Body.Close()
	response, err = http.Get("http://" + region + ".op.gg/summoner/champions/ajax/champions.most/summonerId=" + summonerID + "&season=7")
	if err != nil {
		fmt.Println("Error getting summoner mostChamps:", err)
		return champs
	}
	defer response.Body.Close()
	tokens := html.NewTokenizer(response.Body)
	for {
		token := tokens.Next()
		if token == html.ErrorToken {
			return champs
		} else if token == html.StartTagToken {
			_, hasAttrs := tokens.TagName()

			for hasAttrs {
				key, value, hasMore := tokens.TagAttr()
				if string(key) == "src" {
					champ.imageURL = "http:" + strings.Replace(string(value), "45", "75", 1)
				}
				switch string(value) {
				case "Face":
					champ = new(leagueMostChamps)
				case "KDA":
					token = tokens.Next()
					champ.kda = "KDA: " + strings.TrimSuffix(strings.TrimSpace(string(tokens.Text())), ":1")
				case "Played":
					_ = tokens.Next()
					_ = tokens.Next()
					_ = tokens.Next()
					champ.winRate = "WR: " + strings.TrimSpace(string(tokens.Text()))
					_ = tokens.Next()
					_ = tokens.Next()
					_ = tokens.Next()
					_ = tokens.Next()
					hasMore = false
					champ.gamesPlayed = strings.TrimSpace(strings.TrimSuffix(string(tokens.Text()), "Played"))
					champ.gamesPlayed = "Played: " + commafy(champ.gamesPlayed)
					champs = append(champs, *champ)
				}
				if !hasMore {
					break
				}
			}
		}
	}
}
