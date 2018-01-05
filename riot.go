package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/math/fixed"

	"github.com/golang/freetype"
	"github.com/nfnt/resize"
)

type cardTemplate struct {
	images map[string]imageData
	text   map[string]textData
}
type imageData struct {
	image   image.Image
	area    image.Rectangle
	point   image.Point
	hasMask bool
	mask    image.Image
}
type textData struct {
	text     string
	point    image.Point
	fontSize int
}
type summonerInfo struct {
	IconID    int    `json:"profileIconId"`
	Name      string `json:"name"`
	Level     int    `json:"summonerLevel"`
	AccountID int    `json:"accountId"`
	ID        int    `json:"id"`
}

type summonerLeagues struct {
	QueueType string `json:"queueType"`
	Rank      string `json:"rank"`
	Tier      string `json:"tier"`
	Wins      int    `json:"wins"`
	Losses    int    `json:"losses"`
}
type champMastery struct {
	Level  int `json:"championLevel"`
	ID     int `json:"championId"`
	Points int `json:"championPoints"`
}
type champion struct {
	Name string `json:"name"`
}
type leagueMatchList struct {
	Matches    []leagueMatchReference `json:"matches"`
	TotalGames int                    `json:"totalGames"`
	StartIndex int                    `json:"startIndex"`
	EndIndex   int                    `json:"endIndex"`
	isRanked   bool
}
type leagueMatchReference struct {
	Lane     string `json:"lane"`
	Champion int    `json:"champion"`
}
type leaguesResult []summonerLeagues
type masteryResult []champMastery

var riotKey string
var httpClient = &http.Client{Timeout: 10 * time.Second}
var riotChamps map[int]string
var riotRegions = map[string]string{
	"na":   "na1",
	"br":   "br1",
	"eune": "eun1",
	"euw":  "euw1",
	"jp":   "jp1",
	"kr":   "kr",
	"lan":  "la1",
	"las":  "la2",
	"oce":  "oc1",
	"tr":   "tr1",
	"ru":   "ru",
}

func summonerCardFront() cardTemplate {
	return cardTemplate{
		images: map[string]imageData{
			"background": imageData{
				area:  image.Rect(0, 0, 320, 570),
				point: image.Pt(80, 141),
			}, "border": imageData{
				area:  image.Rect(0, 0, 320, 570),
				point: image.ZP,
			}, "profileIcon": imageData{
				area:  image.Rect(110, 32, 210, 132),
				point: image.ZP,
			}, "insignia": imageData{
				area:  image.Rect(32, 190, 288, 225),
				point: image.ZP,
			}, "solo": imageData{
				area:  image.Rect(44, 230, 144, 330),
				point: image.ZP,
			}, "flex": imageData{
				area:  image.Rect(176, 230, 276, 330),
				point: image.ZP,
			}, "masteryBorder": imageData{
				area:  image.Rect(104, 380, 226, 520),
				point: image.ZP,
			}, "masteryChamp": imageData{
				area:    image.Rect(114, 386, 206, 479),
				point:   image.ZP,
				hasMask: true,
			},
		},
		text: map[string]textData{
			"name": textData{
				point:    image.Pt(160, 178),
				fontSize: 32,
			}, "Solo": textData{
				text:     "Solo",
				point:    image.Pt(74, 345),
				fontSize: 20,
			}, "Flex": textData{
				text:     "Flex",
				point:    image.Pt(208, 345),
				fontSize: 20,
			}, "soloRank": textData{
				point:    image.Pt(94, 365), // X is where the center of the text should be here
				fontSize: 16,
			}, "flexRank": textData{
				point:    image.Pt(226, 365), // X is where the center of the text should be here
				fontSize: 16,
			}, "masteryPoints": textData{
				point:    image.Pt(160, 540),
				fontSize: 16,
			},
		},
	}
}

func summonerCardBack() cardTemplate {
	return cardTemplate{
		images: map[string]imageData{
			"insignia": imageData{
				area:  image.Rect(32, 205, 288, 240),
				point: image.ZP,
			},
		},
		text: map[string]textData{
			"mainRole": textData{
				text:     "Main Role: ",
				point:    image.Pt(160, 260),
				fontSize: 16,
			}, "secondaryRole": textData{
				text:     "Secondary Role: ",
				point:    image.Pt(160, 280),
				fontSize: 16,
			},
		},
	}
}

func mostChampsTemplates(n int) map[string]imageData {
	return [3]map[string]imageData{
		map[string]imageData{
			"middle": imageData{
				area:  image.Rect(123, 64, 197, 139),
				point: image.ZP,
			},
		}, map[string]imageData{
			"left": imageData{
				area:  image.Rect(68, 64, 143, 139),
				point: image.ZP,
			}, "right": imageData{
				area:  image.Rect(177, 64, 252, 139),
				point: image.ZP,
			},
		}, map[string]imageData{
			"left": imageData{
				area:  image.Rect(33, 64, 107, 139),
				point: image.ZP,
			}, "middle": imageData{
				area:  image.Rect(123, 64, 197, 139),
				point: image.ZP,
			}, "right": imageData{
				area:  image.Rect(213, 64, 287, 139),
				point: image.ZP,
			},
		},
	}[n-1]
}

func mostChampsText() map[string]textData {
	return map[string]textData{
		"played": textData{
			text:     "Played: ",
			point:    image.Pt(0, 156),
			fontSize: 16,
		}, "winrate": textData{
			text:     "WR: ",
			point:    image.Pt(0, 176),
			fontSize: 16,
		}, "kda": textData{
			text:     "KDA: ",
			point:    image.Pt(0, 194),
			fontSize: 16,
		},
	}
}

// Initialize some data for use later
func riotInit(version string) error {
	data, err := ioutil.ReadFile("league/" + version + ".json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return err
	}
	riotChamps = make(map[int]string)
	err = json.Unmarshal(data, &riotChamps)
	if err != nil {
		fmt.Println("woops:", err)
		return err
	}
	return nil
}

// Get the body from a URL and return it only if it has a status code of 200
// This should probably be in a different file
// I should probably have this unmarshal the json data as well instead of doing it right after a call to this function (and return error instead of data)
func getURL(url string) []byte {
	resp, err := httpClient.Get(url)
	if err != nil {
		fmt.Println("Error getting a url:", err)
		return nil
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	return data
}

// Get basic player info. (see summonerLeagues and summonerInfo structs)
func riotPlayerInfo(name, region string) (summonerInfo, leaguesResult, leagueMatchList) {
	var sinfo summonerInfo
	var matchesResult leagueMatchList
	sleagues := new(leaguesResult)
	adjustedName := strings.Replace(name, " ", "%20", -1)
	data := getURL(fmt.Sprintf("https://%s.api.riotgames.com/lol/summoner/v3/summoners/by-name/%s?api_key=%s", region, adjustedName, riotKey))
	err := json.Unmarshal(data, &sinfo)
	if err != nil {
		fmt.Println("Error getting summoner id:", err)
		return sinfo, *sleagues, matchesResult
	}
	data = getURL(fmt.Sprintf("https://%s.api.riotgames.com/lol/league/v3/positions/by-summoner/%v?api_key=%s", region, sinfo.ID, riotKey))
	err = json.Unmarshal(data, &sleagues)
	if err != nil {
		fmt.Println("Error getting summoner leagues:", err, string(data))
		return sinfo, *sleagues, matchesResult
	}
	queueType := 400
	matchesResult.isRanked = false
	for _, v := range *sleagues {
		if v.QueueType == "RANKED_SOLO_5x5" {
			queueType = 420
			matchesResult.isRanked = true
		}
	}
	data = getURL(fmt.Sprintf("https://%s.api.riotgames.com/lol/match/v3/matchlists/by-account/%v?queue=%v&season=9&api_key=%s", region, sinfo.AccountID, queueType, riotKey))
	err = json.Unmarshal(data, &matchesResult)
	if err != nil {
		fmt.Println("Error getting summoner matches:", err, string(data))
		return sinfo, *sleagues, matchesResult
	}
	//fmt.Printf("%+v/n", matchesResult)
	return sinfo, *sleagues, matchesResult
}

func riotPlayerCard(playername, region string) *image.RGBA {
	// Gather palyer data
	sinfo, sleagues, smatches := riotPlayerInfo(playername, region)
	schamps := *new(masteryResult)
	data := getURL(fmt.Sprintf("https://%s.api.riotgames.com/lol/champion-mastery/v3/champion-masteries/by-summoner/%v?api_key=%s", region, sinfo.ID, riotKey))
	err := json.Unmarshal(data, &schamps)
	if err != nil {
		fmt.Println("Error getting summoner champion masteries:", err, string(data))
		return nil
	}
	var soloInfo, flexInfo summonerLeagues
	for _, v := range sleagues {
		if v.QueueType == "RANKED_FLEX_SR" {
			flexInfo = v
		} else if v.QueueType == "RANKED_SOLO_5x5" {
			soloInfo = v
		}
	}
	roleMatches := make(map[string]int)
	champMatches := make(map[int]int)
	var mainRoles [2]string
	var mainChamps []int
	for _, v := range smatches.Matches {
		roleMatches[v.Lane]++
		champMatches[v.Champion]++
	}
	for k, v := range roleMatches {
		if v > roleMatches[mainRoles[0]] {
			mainRoles[0], mainRoles[1] = k, mainRoles[0]
		} else if v > roleMatches[mainRoles[1]] {
			mainRoles[1] = k
		}
	}
	fmt.Printf("%+v\n", roleMatches)
	for i, v := range mainRoles {
		if v == "" {
			mainRoles[i] = "N/A"
		} else {
			mainRoles[i] = titlefy(mainRoles[i])
		}
	}
	switch {
	case len(champMatches) >= 3:
		mainChamps = make([]int, 3)
		for k, v := range champMatches {
			if v > champMatches[mainChamps[0]] {
				mainChamps[0], mainChamps[1], mainChamps[2] = k, mainChamps[0], mainChamps[1]
			} else if v > champMatches[mainChamps[1]] {
				mainChamps[1], mainChamps[2] = k, mainChamps[1]
			} else if v > champMatches[mainChamps[2]] {
				mainChamps[2] = k
			}
		}
	case len(champMatches) == 2:
		mainChamps = make([]int, 2)
		for k, v := range champMatches {
			if v > champMatches[mainChamps[0]] {
				mainChamps[0], mainChamps[1] = k, mainChamps[0]
			} else if v > champMatches[mainChamps[1]] {
				mainChamps[1] = k
			}
		}
	case len(champMatches) == 1:
		mainChamps = make([]int, 1)
		for k, v := range champMatches {
			if v > champMatches[mainChamps[0]] {
				mainChamps[0] = k
			}
		}
	}
	var champs []leagueMostChamps
	if region == "kr" {
		champs = opggRankedChamps(strconv.FormatInt(int64(sinfo.ID), 10), "www")
	} else {
		for k, v := range riotRegions {
			if v == region {
				champs = opggRankedChamps(strconv.FormatInt(int64(sinfo.ID), 10), k)
			}
		}
	}
	// Create images and freetype context
	fontFile, err := ioutil.ReadFile("league/FrizQuadrataTT.ttf")
	if err != nil {
		fmt.Println("error opening font:", err)
		return nil
	}
	f, err := freetype.ParseFont(fontFile)
	if err != nil {
		fmt.Println("error parsing font:", err)
		return nil
	}
	front := image.NewRGBA(image.Rect(0, 0, 320, 570))
	back := image.NewRGBA(image.Rect(0, 0, 320, 570))
	both := image.NewRGBA(image.Rect(0, 0, 640, 570))
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(f)
	c.SetClip(front.Bounds())
	c.SetSrc(image.White)
	c.SetDst(front)
	fmt.Println("--------\nName:", playername)
	fmt.Println("Soloq:", soloInfo)
	fmt.Println("Flexq:", flexInfo)
	fmt.Println("Champ:", riotChamps[schamps[0].ID])
	fmt.Println("Main role:", mainRoles[0])
	fmt.Println("Secondary role:", mainRoles[1])
	// Load the templates and fill in the missing info.
	cardFront := summonerCardFront()
	cardBack := summonerCardBack()
	imageInfo := cardFront.images["background"]
	imageInfo.image = loadImage("league/bg.png")
	cardFront.images["background"] = imageInfo
	draw.Draw(front, cardFront.images["background"].area, cardFront.images["background"].image, cardFront.images["background"].point, draw.Over)
	draw.Draw(back, cardFront.images["background"].area, cardFront.images["background"].image, cardFront.images["background"].point, draw.Over)
	delete(cardFront.images, "background")
	imageInfo = cardFront.images["border"]
	imageInfo.image = loadImage(fmt.Sprintf("league/rank_border/%sborder.png", strings.ToLower(soloInfo.Tier)))
	cardFront.images["border"] = imageInfo
	cardBack.images["border"] = imageInfo
	imageInfo = cardFront.images["profileIcon"]
	imageInfo.image = resize.Resize(100, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/profileicon/%v.png", sinfo.IconID)), resize.Lanczos3)
	cardFront.images["profileIcon"] = imageInfo
	imageInfo = cardFront.images["insignia"]
	imageInfo.image = resize.Resize(256, 0, loadImage(fmt.Sprintf("league/insignia/%sinsignia.png", strings.ToLower(soloInfo.Tier))), resize.Lanczos3)
	cardFront.images["insignia"] = imageInfo
	imageInfo = cardFront.images["solo"]
	imageInfo.image = resize.Resize(100, 0, loadImage(fmt.Sprintf("league/rank/%s_%s.png", strings.ToLower(soloInfo.Tier), soloInfo.Rank)), resize.Lanczos3)
	cardFront.images["solo"] = imageInfo
	imageInfo = cardFront.images["flex"]
	imageInfo.image = resize.Resize(100, 0, loadImage(fmt.Sprintf("league/rank/%s_%s.png", strings.ToLower(flexInfo.Tier), flexInfo.Rank)), resize.Lanczos3)
	cardFront.images["flex"] = imageInfo
	if soloInfo.Tier == "" {
		soloInfo.Tier = "Unranked"
	}
	if flexInfo.Tier == "" {
		flexInfo.Tier = "Unranked"
	}
	if len(schamps) > 0 {
		imageInfo = cardFront.images["masteryBorder"]
		imageInfo.image = loadImage(fmt.Sprintf("league/mastery_border/%v.png", schamps[0].Level))
		cardFront.images["masteryBorder"] = imageInfo

		imageInfo = cardFront.images["masteryChamp"]
		imageInfo.image = resize.Resize(93, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[schamps[0].ID])), resize.Lanczos3)
		imageInfo.mask = loadImage("league/champmask.png")
		cardFront.images["masteryChamp"] = imageInfo
		draw.DrawMask(front, cardFront.images["masteryChamp"].area, cardFront.images["masteryChamp"].image, image.ZP, cardFront.images["masteryChamp"].mask, image.ZP, draw.Over)
		delete(cardFront.images, "masteryChamp")
	} else {
		delete(cardFront.images, "masteryBorder")
		delete(cardFront.images, "masteryChamp")
	}
	textData := cardFront.text["name"]
	textData.text = sinfo.Name
	for {
		if width := textWidth(c, textData.text, textData.fontSize); width <= 256 {
			textData.point.X -= width / 2
			break
		}
		textData.fontSize--
	}
	cardFront.text["name"] = textData

	textData = cardFront.text["soloRank"]
	textData.text = fmt.Sprintf("%s %s", titlefy(soloInfo.Tier), soloInfo.Rank)
	textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
	cardFront.text["soloRank"] = textData

	textData = cardFront.text["flexRank"]
	textData.text = fmt.Sprintf("%s %s", titlefy(flexInfo.Tier), flexInfo.Rank)
	textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
	cardFront.text["flexRank"] = textData

	if len(schamps) > 0 {
		textData := cardFront.text["masteryPoints"]
		textData.text = commafy(strconv.FormatInt(int64(schamps[0].Points), 10))
		textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
		cardFront.text["masteryPoints"] = textData
	} else {
		delete(cardFront.images, "masteryPoints")
	}
	for _, v := range cardFront.images {
		draw.Draw(front, v.area, v.image, image.ZP, draw.Over)
	}
	for k, v := range cardFront.text {
		c.SetFontSize(float64(v.fontSize))
		if _, err := c.DrawString(v.text, pointToFixed(v.point)); err != nil {
			fmt.Println("Error writing text:", k, err)
		}
	}
	imageInfo = cardBack.images["insignia"]
	imageInfo.image = cardFront.images["insignia"].image
	cardBack.images["insignia"] = imageInfo

	textData = cardBack.text["mainRole"]
	textData.text += mainRoles[0]
	textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
	cardBack.text["mainRole"] = textData
	textData = cardBack.text["secondaryRole"]
	textData.text += mainRoles[1]
	textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
	cardBack.text["secondaryRole"] = textData
	if len(champs) > 0 {
		var images map[string]imageData
		text := mostChampsText()
		switch {
		case len(champs) >= 3:
			images = mostChampsTemplates(3)
			imageInfo = images["left"]
			imageInfo.image = loadImage(champs[0].imageURL)
			images["left"] = imageInfo
			imageInfo = images["middle"]
			imageInfo.image = loadImage(champs[1].imageURL)
			images["middle"] = imageInfo
			imageInfo = images["right"]
			imageInfo.image = loadImage(champs[2].imageURL)
			images["right"] = imageInfo

		case len(schamps) == 2:
			images = mostChampsTemplates(2)
			imageInfo = images["left"]
			imageInfo.image = loadImage(champs[0].imageURL)
			images["left"] = imageInfo
			imageInfo = images["right"]
			imageInfo.image = loadImage(champs[1].imageURL)
			images["right"] = imageInfo
			cardBack.images["left"] = images["left"]
			cardBack.images["right"] = images["right"]

		case len(schamps) == 1:
			images = mostChampsTemplates(1)
			imageInfo = images["middle"]
			imageInfo.image = loadImage(champs[1].imageURL)
			images["middle"] = imageInfo
			cardBack.images["middle"] = images["middle"]

		}
		i := 0
		for k, v := range images {
			cardBack.images[k] = v
			textData = text["played"]
			textData.text = champs[i].gamesPlayed
			textData.point.X = (images[k].area.Max.X-images[k].area.Min.X)/2 + images[k].area.Min.X
			for {
				if width := textWidth(c, textData.text, textData.fontSize); width <= 75 {
					textData.point.X -= width / 2
					break
				}
				textData.fontSize--
			}
			cardBack.text[k+"played"] = textData
			textData = text["kda"]
			textData.text = champs[i].kda
			textData.point.X = (images[k].area.Max.X-images[k].area.Min.X)/2 + images[k].area.Min.X
			for {
				if width := textWidth(c, textData.text, textData.fontSize); width <= 75 {
					textData.point.X -= width / 2
					break
				}
				textData.fontSize--
			}
			cardBack.text[k+"kda"] = textData
			textData = text["winrate"]
			textData.text = champs[i].winRate
			textData.point.X = (images[k].area.Max.X-images[k].area.Min.X)/2 + images[k].area.Min.X
			for {
				if width := textWidth(c, textData.text, textData.fontSize); width <= 75 {
					textData.point.X -= width / 2
					break
				}
				textData.fontSize--
			}
			cardBack.text[k+"winrate"] = textData
			i++
		}
	} else if len(mainChamps) > 0 {
		var images map[string]imageData
		text := mostChampsText()
		switch {
		case len(mainChamps) >= 3:
			images = mostChampsTemplates(3)
			imageInfo = images["left"]
			imageInfo.image = resize.Resize(75, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[0]])), resize.Lanczos3)
			images["left"] = imageInfo
			imageInfo = images["middle"]
			imageInfo.image = resize.Resize(75, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[1]])), resize.Lanczos3)
			images["middle"] = imageInfo
			imageInfo = images["right"]
			imageInfo.image = resize.Resize(75, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[2]])), resize.Lanczos3)
			images["right"] = imageInfo

		case len(mainChamps) == 2:
			images = mostChampsTemplates(2)
			imageInfo = images["left"]
			imageInfo.image = resize.Resize(75, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[0]])), resize.Lanczos3)
			images["left"] = imageInfo
			imageInfo = images["right"]
			imageInfo.image = resize.Resize(75, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[1]])), resize.Lanczos3)
			images["right"] = imageInfo
			cardBack.images["left"] = images["left"]
			cardBack.images["right"] = images["right"]

		case len(mainChamps) == 1:
			images = mostChampsTemplates(1)
			imageInfo = images["middle"]
			imageInfo.image = resize.Resize(75, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[0]])), resize.Lanczos3)
			images["middle"] = imageInfo
			cardBack.images["middle"] = images["middle"]

		}
		i := 0
		for k, v := range images {
			cardBack.images[k] = v
			textData = text["played"]
			textData.text += commafy(strconv.FormatInt(int64(champMatches[mainChamps[i]]), 10))
			textData.point.X = (images[k].area.Max.X-images[k].area.Min.X)/2 + images[k].area.Min.X
			for {
				if width := textWidth(c, textData.text, textData.fontSize); width <= 75 {
					textData.point.X -= width / 2
					break
				}
				textData.fontSize--
			}
			cardBack.text[k+"played"] = textData
			i++
		}

	}

	for _, v := range cardBack.images {
		draw.Draw(back, v.area, v.image, image.ZP, draw.Over)
	}
	c.SetDst(back)
	for k, v := range cardBack.text {
		c.SetFontSize(float64(v.fontSize))
		if _, err := c.DrawString(v.text, pointToFixed(v.point)); err != nil {
			fmt.Println("Error writing text:", k, err)
		}
	}
	/*
			var urls [5]string
			var files [7]string
			var images [12]image.Image
			urls[0] = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[schamps[0].ID])
			urls[1] = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/profileicon/%v.png", sinfo.IconID)
			if smatches.isRanked {
				urls[2] = champs[0].imageURL
				urls[3] = champs[1].imageURL
				urls[4] = champs[2].imageURL
			} else {

				urls[2] = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[0]])
				urls[3] = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[1]])
				urls[4] = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[mainChamps[2]])
			}
			files[0] = "league/bg.png"
			// Account for unranked players here. this could really be avoided by renaming "default.png" to "_.png"
			// I also need to store the directory path in a variable so when I change it I don't have to touch this code
			if soloInfo.Tier == "" {
				files[1] = "league/default.png"
				soloInfo.Tier = "Unranked"
				files[3] = "league/norankborder.png"
				files[4] = "league/norankinsignia.png"
			} else {
				files[1] = fmt.Sprintf("league/%s_%s.png", soloInfo.Tier, soloInfo.Rank)
				files[3] = fmt.Sprintf("league/%sborder.png", strings.ToLower(soloInfo.Tier))
				files[4] = fmt.Sprintf("league/%sinsignia.png", strings.ToLower(soloInfo.Tier))
			}
			if flexInfo.Tier == "" {
				files[2] = "league/default.png"
				flexInfo.Tier = "Unranked"
			} else {
				files[2] = fmt.Sprintf("league/%s_%s.png", flexInfo.Tier, flexInfo.Rank)
			}
			files[5] = fmt.Sprintf("league/level%vborder.png", schamps[0].Level)
			files[6] = "league/champmask.png"
			soloInfo.Tier = titlefy(soloInfo.Tier)
			flexInfo.Tier = titlefy(flexInfo.Tier)
			// Fill up the images array with the images. I need to label what each index is somewhere
			for i, v := range urls {
				if i >= 2 {
					break
				}
				url, err := http.Get(v)
				if err != nil || url.StatusCode != 200 {
					fmt.Println("error getting image from url:", url.StatusCode, v, err)
					return nil
				}
				images[i], err = png.Decode(url.Body)
				url.Body.Close()
				if err != nil {
					fmt.Println("error decoding image from url:", err)
					return nil
				}
			}
			for i, v := range files {
				file, err := ioutil.ReadFile(v)
				if err != nil {
					fmt.Println("error reading file:", err)
					return nil
				}
				images[i+2], err = png.Decode(bytes.NewReader(file))
				if err != nil {
					fmt.Println("error decoding image from file:", err)
					return nil
				}
			}
			for i, v := range urls {
				if i < 2 {
					continue
				}
				url, err := http.Get(v)
				if err != nil || url.StatusCode != 200 {
					fmt.Println("error getting image from url:", url.StatusCode, v, err)
					return nil
				}
				images[i+7], err = png.Decode(url.Body)
				url.Body.Close()
				if err != nil {
					fmt.Println("error decoding image from url:", err)
					return nil
				}
				images[i+7] = resize.Resize(75, 0, images[i+7], resize.Lanczos3)
			}
		fontFile, err := ioutil.ReadFile("league/FrizQuadrataTT.ttf")
		if err != nil {
			fmt.Println("error opening font:", err)
			return nil
		}
		f, err := freetype.ParseFont(fontFile)
		if err != nil {
			fmt.Println("error parsing font:", err)
			return nil
		}
		front := image.NewRGBA(image.Rect(0, 0, 320, 570))
		back := image.NewRGBA(image.Rect(0, 0, 320, 570))
		both := image.NewRGBA(image.Rect(0, 0, 640, 570))
		c := freetype.NewContext()
		c.SetDPI(72)
		c.SetFont(f)
		c.SetClip(front.Bounds())
		c.SetSrc(image.White)
		c.SetDst(front)
		// Draw image. There has to be some way I can tidy this up
		center := image.Pt(front.Bounds().Dx()/2, front.Bounds().Dy()/2)
		images[1] = resize.Resize(100, 0, images[1], resize.Lanczos3)
		images[3] = resize.Resize(100, 0, images[3], resize.Lanczos3)
		images[4] = resize.Resize(100, 0, images[4], resize.Lanczos3)
		images[0] = resize.Resize(93, 0, images[0], resize.Lanczos3)
		images[6] = resize.Resize(256, 0, images[6], resize.Lanczos3)
		draw.Draw(front, front.Bounds(), images[2], image.Pt(images[2].Bounds().Dx()/2-center.X, images[2].Bounds().Dy()/2-center.Y), draw.Src)
		draw.Draw(front, front.Bounds(), images[5], image.ZP, draw.Over)
		// rgba has the background and border, I need to make a copy of the image at this point to use for the "back" of the card
		draw.Draw(back, back.Bounds(), front, image.ZP, draw.Src)
		draw.Draw(back, image.Rect(center.X-75/2-15-75, 64, center.X+75/2-15-75, 64+75), images[9], image.ZP, draw.Over)
		fmt.Printf("left: %+v\n", image.Rect(center.X-75/2-15-75, 64, center.X+75/2-15-75, 64+75))
		fmt.Printf("middle: %+v\n", image.Rect(center.X-75/2, 64, center.X+75/2, 64+75))
		fmt.Printf("right: %+v\n", image.Rect(center.X-75/2+15+75, 64, center.X+75/2+15+75, 64+75))
		draw.Draw(back, image.Rect(center.X-75/2, 64, center.X+75/2, 64+75), images[10], image.ZP, draw.Over)
		draw.Draw(back, image.Rect(center.X-75/2+15+75, 64, center.X+75/2+15+75, 64+75), images[11], image.ZP, draw.Over)
		draw.Draw(back, image.Rect(center.X-128, 205, center.X+128,
			images[6].Bounds().Dy()+205), images[6], image.ZP, draw.Over)
		// ----------------------
		draw.Draw(front, image.Rect(center.X-116, 230, center.X-16, 330), images[3], image.ZP, draw.Over)
		draw.Draw(front, image.Rect(center.X+16, 230, center.X+116, 330), images[4], image.ZP, draw.Over)
		draw.Draw(front, image.Rect(center.X-images[1].Bounds().Dx()/2, 32, center.X+images[1].Bounds().Dx()/2, images[1].Bounds().Dy()+32), images[1], image.ZP, draw.Src)
		draw.DrawMask(front, image.Rect(center.X-images[0].Bounds().Dx()/2, 386, center.X+images[0].Bounds().Dx()/2, images[0].Bounds().Dy()+386),
			images[0], image.ZP, images[8], image.ZP, draw.Over)
		draw.Draw(front, image.Rect(center.X-images[7].Bounds().Dx()/2, 380, center.X+images[7].Bounds().Dx()/2, images[7].Bounds().Dy()+380),
			images[7], image.ZP, draw.Over)
		draw.Draw(front, image.Rect(center.X-128, 190, center.X-128+images[6].Bounds().Dx(),
			images[6].Bounds().Dy()+190), images[6], image.ZP, draw.Over)
		// Draw text. Probably needs to be cleaned more than anything else
		// TODO: make a function for width of a string: func(*context, string, fontsize)
		c := freetype.NewContext()
		c.SetDPI(72)
		c.SetFont(f)
		c.SetClip(front.Bounds())
		c.SetSrc(image.White)
		c.SetDst(front)
		for fontSize := 32; textWidth(c, playername, fontSize) > 256; fontSize -= 2 {
		}
		pt := freetype.Pt(center.X-textWidth(c, playername, 0)/2, 180)
		if _, err := c.DrawString(playername, pt); err != nil {
			fmt.Println("error1:", err)
		}

		c.SetFontSize(20)
		pt = freetype.Pt(center.X-16-50-textWidth(c, "Solo", 0)/2, 345)
		if _, err := c.DrawString("Solo", pt); err != nil {
			fmt.Println("error2:", err)
		}
		pt = freetype.Pt(center.X+16+50-textWidth(c, "Flex", 0)/2, 345)
		if _, err := c.DrawString("Flex", pt); err != nil {
			fmt.Println("error3:", err)
		}
		c.SetFontSize(16)
		pt = freetype.Pt(center.X-16-50-textWidth(c, fmt.Sprintf("%s %s", soloInfo.Tier, soloInfo.Rank), 0)/2, 345+20)
		if _, err := c.DrawString(fmt.Sprintf("%s %s", soloInfo.Tier, soloInfo.Rank), pt); err != nil {
			fmt.Println("error4:", err)
		}
		pt = freetype.Pt(center.X+16+50-textWidth(c, fmt.Sprintf("%s %s", flexInfo.Tier, flexInfo.Rank), 0)/2, 345+20)
		if _, err := c.DrawString(fmt.Sprintf("%s %s", flexInfo.Tier, flexInfo.Rank), pt); err != nil {
			fmt.Println("error5:", err)
		}

		pt = freetype.Pt(center.X-textWidth(c, commafy(strconv.FormatInt(int64(schamps[0].Points), 10)), 0)/2, 540)
		if _, err := c.DrawString(commafy(strconv.FormatInt(int64(schamps[0].Points), 10)), pt); err != nil {
			fmt.Println("error6:", err)
		}

		c.SetDst(back)
		//64+75
		// left
		if smatches.isRanked {
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[0].winRate, fontSize); width <= 75 {
					pt = freetype.Pt(center.X-15-75-width/2, 160)
					break
				}
			}
			if _, err := c.DrawString(champs[0].winRate, pt); err != nil {
				fmt.Println("error6:", err)
			}
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[0].gamesPlayed, fontSize); width <= 75 {
					pt = freetype.Pt(center.X-15-75-width/2, 176)
					break
				}
			}
			if _, err := c.DrawString(champs[0].gamesPlayed, pt); err != nil {
				fmt.Println("error6:", err)
			}
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[0].kda, fontSize); width <= 75 {
					pt = freetype.Pt(center.X-15-75-width/2, 194)
					break
				}
			}
			if _, err := c.DrawString(champs[0].kda, pt); err != nil {
				fmt.Println("error6:", err)
			}
			// Middle
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[1].winRate, fontSize); width <= 75 {
					pt = freetype.Pt(center.X-width/2, 160)
					break
				}
			}
			if _, err := c.DrawString(champs[1].winRate, pt); err != nil {
				fmt.Println("error6:", err)
			}
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[1].gamesPlayed, fontSize); width <= 75 {
					pt = freetype.Pt(center.X-width/2, 176)
					break
				}
			}
			if _, err := c.DrawString(champs[1].gamesPlayed, pt); err != nil {
				fmt.Println("error6:", err)
			}
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[1].kda, fontSize); width <= 75 {
					pt = freetype.Pt(center.X-width/2, 194)
					break
				}
			}
			if _, err := c.DrawString(champs[1].kda, pt); err != nil {
				fmt.Println("error6:", err)
			}
			// Right
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[2].winRate, fontSize); width <= 75 {
					pt = freetype.Pt(center.X+15+75-width/2, 160)
					break
				}
			}
			if _, err := c.DrawString(champs[2].winRate, pt); err != nil {
				fmt.Println("error6:", err)
			}
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[2].gamesPlayed, fontSize); width <= 75 {
					pt = freetype.Pt(center.X+15+75-width/2, 176)
					break
				}
			}
			if _, err := c.DrawString(champs[2].gamesPlayed, pt); err != nil {
				fmt.Println("error6:", err)
			}
			for fontSize := 14; true; fontSize-- {
				if width := textWidth(c, champs[2].kda, fontSize); width <= 75 {
					pt = freetype.Pt(center.X+15+75-width/2, 194)
					break
				}
			}
			if _, err := c.DrawString(champs[2].kda, pt); err != nil {
				fmt.Println("error6:", err)
			}
		}
		pt = freetype.Pt(center.X-textWidth(c, "Main Role: "+mainRole, 16)/2, 260)
		if _, err := c.DrawString("Main Role: "+mainRole, pt); err != nil {
			fmt.Println("error6:", err)
		}*/
	fmt.Println("Playercard created successfully!")

	draw.Draw(both, front.Bounds(), front, image.ZP, draw.Src)
	draw.Draw(both, front.Bounds().Add(image.Pt(321, 0)), back, image.ZP, draw.Src)
	return both
}

// Get the width in pixel of a string (if size is 0, use whatever was set previously)
// If font size is specified, it WILL change the font size for the given freetype *Context
func textWidth(c *freetype.Context, text string, size int) int {
	temp := image.NewRGBA(image.Rect(0, 0, 300, 100))
	pt := freetype.Pt(0, 50)
	if size > 0 {
		c.SetFontSize(float64(size))
	}
	ftcopy := *c
	ftcopy.SetDst(temp)
	endpoint, err := ftcopy.DrawString(text, pt)
	if err != nil {
		fmt.Println("Error getting text width:", err)
		return -1
	}
	return int(endpoint.X >> 6)
}

// Take a number and add commas every three digits, from the left
func commafy(s string) string {
	newLength := len(s) + (len(s)-1)/3
	newString := make([]byte, newLength)
	newLength--
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if count == 3 {
			count = 0
			newString[newLength] = ','
			newLength--
		}
		newString[newLength] = s[i]
		count++
		newLength--
	}
	return string(newString)
}

// Calling both functions from the strings package to get all caps into titles is annoying
func titlefy(text string) string {
	return strings.Title(strings.ToLower(text))
}

// Wrapper for converting image.Point to fixed.Point26_6 to make things a tad cleaner
func pointToFixed(point image.Point) fixed.Point26_6 {
	return freetype.Pt(point.X, point.Y)
}

func loadImage(path string) image.Image {
	if strings.Contains(path, "http") {
		response, err := http.Get(path)
		if err != nil || response.StatusCode != 200 {
			fmt.Println("error getting image from url:", response.StatusCode, path, err)
			return nil
		}
		image, err := png.Decode(response.Body)
		response.Body.Close()
		if err != nil {
			fmt.Println("error decoding image from url:", err)
			return nil
		}
		return image
	}
	file, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error reading file:", path, err)
		return nil
	}
	image, err := png.Decode(bytes.NewReader(file))
	if err != nil {
		fmt.Println("error decoding image from file:", err)
		return nil
	}
	return image
}
