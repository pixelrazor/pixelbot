package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/yuhanfang/riot/constants/region"

	"github.com/PuerkitoBio/goquery"
)

type leagueMostChamps struct {
	imageURL    string
	gamesPlayed string
	winRate     string
	kda         string
}

func opggLink(region region.Region) string {
	if region == "kr" {
		return "http://op.gg/"
	}
	for k, v := range riotRegions {
		if v == region {
			return "http://" + k + ".op.gg/"
		}
	}
	return "http://na.op.gg/"
}
/*
<button class="Button SemiRound Green" id="SummonerRefreshButton" onclick="$.OP.GG.summoner.renewBtn.start(this, '51630642');">Updated</button>


*/
func opggRankedChamps(summonerID, region string) []leagueMostChamps {
	var champs []leagueMostChamps
	var champ *leagueMostChamps
	res, err := http.PostForm("http://"+region+".op.gg/summoner/ajax/championMost/renew.json/", url.Values{"summonerId": {summonerID}})
	if err != nil {
		logger.Println("Error refreshing summoner info:", err)
	} else {
		res.Body.Close()
	}
	res, err = http.Get("http://" + region + ".op.gg/summoner/champions/ajax/champions.most/summonerId=" + summonerID + "&season=11")
	if err != nil {
		logger.Println("Error getting summoner mostChamps:", err)
		return champs
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logger.Println("Error getting summoner mostChamps:", err)
		return champs
	}
	doc.Find(".ChampionBox").Each(func(i int, s *goquery.Selection) {
		champ = new(leagueMostChamps)
		ci := s.Find(".ChampionImage")
		image, _ := ci.Attr("src")
		champ.imageURL = "http:" + strings.Replace(string(image), "45", "75", 1)
		kdaSelect := s.Find(".KDA > .KDA")
		if kdaSelect.Size() != 0 {
			champ.kda = "KDA: " + strings.TrimSuffix(strings.TrimSpace(kdaSelect.Text()), ":1")
		} else {
			champ.kda = "KDA: Perfect"
		}
		champ.winRate = "WR: " + strings.TrimSpace(s.Find(".WinRatio").Text())
		champ.gamesPlayed = "Played: " + strings.TrimSpace(strings.TrimSuffix(s.Find(".Played .Title").Text(), "Played"))
		champs = append(champs, *champ)
	})
	return champs
}
