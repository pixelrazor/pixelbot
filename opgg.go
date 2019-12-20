package main

import (
	"fmt"
	"github.com/golang/freetype"
	"github.com/nfnt/resize"
	"image"
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

// Attempt to get a Player's Solo rank
func opggPastRanks(c *freetype.Context, username string, region region.Region) cardTemplate {
	var ranks cardTemplate
	ranks.images = make(map[string]imageData)
	ranks.text = make(map[string]textData)
	var seasonRanks []string
	var seasons []int
	res, err := http.Get(opggLink(region) + "summoner/userName=" + strings.Replace(username, " ", "+", -1))
	if err != nil {
		logger.Println("Error getting past ranks:", err)
		return ranks
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logger.Println("Error getting past ranks:", err)
		return ranks
	}
	doc.Find(".PastRankList li").Each(func(i int, s *goquery.Selection) {
		var (
			season int
			tier   string
		)
		fmt.Sscanf(strings.TrimSpace(s.Text()), "S%d %s", &season, &tier)
		tier = strings.ToUpper(tier)
		seasonRanks = append(seasonRanks, tier)
		seasons = append(seasons, season)
	})
	var ranksImages []imageData
	if len(seasonRanks) > 3 {
		ranksImages = mostChampsTemplates(3)
		seasonRanks = seasonRanks[len(seasonRanks)-3:]
		seasons = seasons[len(seasons)-3:]
	} else if len(seasonRanks) > 0 {
		ranksImages = mostChampsTemplates(len(seasonRanks))
	}
	for i, v := range seasonRanks {
		var text textData
		ranksImages[i].area = image.Rect(ranksImages[i].area.Min.X, 350, ranksImages[i].area.Max.X, 425)
		ranksImages[i].image = resize.Resize(75, 0, loadImage(fmt.Sprintf("league/rank/%s_I.png", v)), resize.Lanczos3)
		ranks.images[fmt.Sprintf("%v%s", seasons[i], v)] = ranksImages[i]
		text.fontSize = 12
		text.text = fmt.Sprintf("Season %v", seasons[i])
		text.point = image.Pt((ranksImages[i].area.Min.X+ranksImages[i].area.Max.X)/2-textWidth(c, text.text, text.fontSize)/2, 435)
		ranks.text[fmt.Sprintf("%v%s", seasons[i], v)] = text
	}
	return ranks
}

func opggRankedChamps(summonerName string, region region.Region) []leagueMostChamps {
	var champs []leagueMostChamps
	fmt.Println(opggLink(region) + "summoner/champions/userName=" + url.QueryEscape(summonerName))
	res, err := http.Get(opggLink(region) + "summoner/champions/userName=" + url.QueryEscape(summonerName))
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
	doc.Find(".TopRanker").Each(func(i int, s *goquery.Selection) {
		champ := leagueMostChamps{}
		kda, _ := s.Find(".KDA").Attr("data-value")
		champ.kda = "KDA: " + kda
		wins, losses := 0, 0
		fmt.Sscanf(s.Find(".WinRatioGraph .Text.Left").Text(), "%dW", &wins)
		fmt.Sscanf(s.Find(".WinRatioGraph .Text.Right").Text(), "%dL", &losses)
		champ.winRate = "WR: " + fmt.Sprint(int(float64(wins)/float64(losses)*100)) + "%"
		champ.gamesPlayed = "Played: " + fmt.Sprint(wins+losses)
		cname, _ := s.Find(".ChampionName").Attr("data-value")
		cname = titlefy(strings.ReplaceAll(cname, "'", ""))
		champ.imageURL = "https://opgg-static.akamaized.net/images/lol/champion/" + cname + ".png"
		for k, v := range riotChamps {
			if v == cname {
				champ.imageURL = fmt.Sprintf("league/champion/%d.png", k)
			}
		}
		champs = append(champs, champ)
	})
	return champs
}
