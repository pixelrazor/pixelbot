package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yuhanfang/riot/apiclient"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuhanfang/riot/constants/champion"

	"github.com/yuhanfang/riot/constants/lane"
	"github.com/yuhanfang/riot/constants/tier"

	"golang.org/x/image/math/fixed"

	"github.com/golang/freetype"
	"github.com/nfnt/resize"

	"github.com/yuhanfang/riot/constants/region"
	"github.com/yuhanfang/riot/ratelimit"
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
type errChan struct {
	value interface{}
	err   error
}

var riotKey string
var httpClient = &http.Client{Timeout: 10 * time.Second}
var ctx = context.Background()
var limiter = ratelimit.NewLimiter()
var riotClient apiclient.Client
var riotChamps map[int]string
var riotRegions = map[string]region.Region{
	"na":   region.NA1,
	"br":   region.BR1,
	"eune": region.EUN1,
	"euw":  region.EUW1,
	"jp":   region.JP1,
	"kr":   region.KR,
	"lan":  region.LA1,
	"las":  region.LA2,
	"oce":  region.OC1,
	"tr":   region.TR1,
	"ru":   region.RU,
}
var riotRanks = map[tier.Tier]int{
	"BRONZE":     1,
	"SILVER":     2,
	"GOLD":       3,
	"PLATINUM":   4,
	"DIAMOND":    5,
	"MASTER":     6,
	"CHALLENGER": 7,
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

func mostChampsTemplates(n int) []imageData {
	return [3][]imageData{
		[]imageData{
			imageData{
				area:  image.Rect(123, 64, 197, 139),
				point: image.ZP,
			},
		}, []imageData{
			imageData{
				area:  image.Rect(68, 64, 143, 139),
				point: image.ZP,
			}, imageData{
				area:  image.Rect(177, 64, 252, 139),
				point: image.ZP,
			},
		}, []imageData{
			imageData{
				area:  image.Rect(33, 64, 107, 139),
				point: image.ZP,
			}, imageData{
				area:  image.Rect(123, 64, 197, 139),
				point: image.ZP,
			}, imageData{
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
	riotClient = apiclient.New(riotKey, httpClient, limiter)
	return nil
}

// Get basic player info. (see summonerLeagues and summonerInfo structs)

// Attempt to get a Player's Solo rank
func riotPastRanks(c *freetype.Context, username, region string) cardTemplate {
	var ranks cardTemplate
	ranks.images = make(map[string]imageData)
	ranks.text = make(map[string]textData)
	var seasonRanks []string
	var seasons []int
	res, err := http.Get("http://" + region + ".op.gg/summoner/userName=" + strings.Replace(username, " ", "+", -1))
	if err != nil {
		fmt.Println("Error getting past ranks")
		return ranks
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println("Error getting past ranks")
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

// Create and return a playercard
func riotPlayerCard(playername *string, region region.Region) (*image.RGBA, error) {
	// Gather playyer data
	//sinfo, sleagues, smatches := riotPlayerInfo(*playername, region)
	masteryChan := make(chan errChan, 1)
	leaguesChan := make(chan errChan, 1)
	matchesChan := make(chan errChan, 1)
	opggchampsChan := make(chan []leagueMostChamps, 1)
	sinfo, err := riotClient.GetBySummonerName(ctx, region, *playername)
	if err != nil {
		return nil, errors.New("Couldn't find summoner '" + *playername + "'")
	}
	*playername = sinfo.Name
	go func() {
		stuff, err := riotClient.GetAllChampionMasteries(ctx, region, sinfo.ID)
		masteryChan <- errChan{stuff, err}
		close(masteryChan)
	}()
	go func() {
		stuff, err := riotClient.GetAllLeaguePositionsForSummoner(ctx, region, sinfo.ID)
		leaguesChan <- errChan{stuff, err}
		close(leaguesChan)
	}()
	go func() {
		stuff, err := riotClient.GetMatchlist(ctx, region, sinfo.AccountID, nil)
		matchesChan <- errChan{stuff, err}
		close(matchesChan)
	}()
	go func() {
		if region == "kr" {
			opggchampsChan <- opggRankedChamps(strconv.FormatInt(int64(sinfo.ID), 10), "www")
		} else {
			for k, v := range riotRegions {
				if v == region {
					opggchampsChan <- opggRankedChamps(strconv.FormatInt(int64(sinfo.ID), 10), k)
				}
			}
		}
		close(opggchampsChan)
	}()
	result, open := <-masteryChan
	if result.err != nil || !open {
		return nil, errors.New("Unknown issue, try again later")
	}
	schamps := result.value.([]apiclient.ChampionMastery)
	if len(schamps) == 0 {
		return nil, errors.New("Account is too old/unused")
	}
	result = <-leaguesChan
	if result.err != nil {
		return nil, errors.New("Unknown issue, try again later")
	}
	sleagues := result.value.([]apiclient.LeaguePosition)
	var soloInfo, flexInfo apiclient.LeaguePosition
	for _, v := range sleagues {
		if v.QueueType == "RANKED_FLEX_SR" || v.QueueType == "RANKED_FLEX_TT" {
			if riotRanks[v.Tier] > riotRanks[flexInfo.Tier] {
				flexInfo = v
			}
		} else if v.QueueType == "RANKED_SOLO_5x5" {
			soloInfo = v
		}
	}
	result = <-matchesChan
	if result.err != nil {
		return nil, errors.New("Unknown issue, try again later")
	}
	smatches := result.value.(*apiclient.Matchlist)
	imagesChan := make(chan image.Image, 4)
	defer close(imagesChan)
	go func() {
		imagesChan <- loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/8.8.2/img/profileicon/%v.png", sinfo.ProfileIconID))
		imagesChan <- loadImage(fmt.Sprintf("league/insignia/%sinsignia.png", strings.ToLower(string(soloInfo.Tier))))
		imagesChan <- loadImage(fmt.Sprintf("league/rank/%s_%s.png", soloInfo.Tier, soloInfo.Rank))
		imagesChan <- loadImage(fmt.Sprintf("league/rank/%s_%s.png", flexInfo.Tier, flexInfo.Rank))
		if len(schamps) > 0 {
			imagesChan <- loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/8.8.2/img/champion/%s.png", riotChamps[int(schamps[0].ChampionID)]))
		}
	}()
	roleMatches := make(map[lane.Lane]int)
	champMatches := make(map[champion.Champion]int)
	var mainRoles [2]lane.Lane
	var mainChamps []struct {
		champ champion.Champion
		n     int
	}
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
	for k, v := range champMatches {
		mainChamps = append(mainChamps, struct {
			champ champion.Champion
			n     int
		}{k, v})
	}
	sort.Slice(mainChamps, func(i, j int) bool { return mainChamps[i].n < mainChamps[j].n })
	for i, v := range mainRoles {
		if v == "" {
			mainRoles[i] = "N/A"
		} else {
			mainRoles[i] = lane.Lane(titlefy(string(mainRoles[i])))
		}
	}
	var champs []leagueMostChamps
	champs = <-opggchampsChan
	// Create images and freetype context
	fontFile, err := ioutil.ReadFile("league/FrizQuadrataTT.ttf")
	if err != nil {
		fmt.Println("error opening font:", err)
		return nil, nil
	}
	f, err := freetype.ParseFont(fontFile)
	if err != nil {
		fmt.Println("error parsing font:", err)
		return nil, nil
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
	fmt.Println("--------\nName:", *playername)
	fmt.Println("Soloq:", soloInfo)
	fmt.Println("Flexq:", flexInfo)
	fmt.Println("Champ:", schamps[0].ChampionID.String())
	fmt.Println("Main role:", mainRoles[0])
	fmt.Println("Secondary role:", mainRoles[1])
	// Load the templates and fill in the missing info.
	oldRanksChan := make(chan cardTemplate, 1)
	defer close(oldRanksChan)
	go func() {
		if region == "kr" {
			oldRanksChan <- riotPastRanks(c, sinfo.Name, "www")
		} else {
			for k, v := range riotRegions {
				if v == region {
					oldRanksChan <- riotPastRanks(c, sinfo.Name, k)
				}
			}
		}
		oldRanksChan <- *new(cardTemplate)
	}()
	cardFront := summonerCardFront()
	cardBack := summonerCardBack()
	imageInfo := cardFront.images["background"]
	imageInfo.image = loadImage("league/bg.png")
	cardFront.images["background"] = imageInfo
	draw.Draw(front, cardFront.images["background"].area, cardFront.images["background"].image, cardFront.images["background"].point, draw.Over)
	draw.Draw(back, cardFront.images["background"].area, cardFront.images["background"].image, cardFront.images["background"].point, draw.Over)
	delete(cardFront.images, "background")
	imageInfo = cardFront.images["border"]
	imageInfo.image = loadImage(fmt.Sprintf("league/rank_border/%sborder.png", strings.ToLower(string(soloInfo.Tier))))
	cardFront.images["border"] = imageInfo
	cardBack.images["border"] = imageInfo
	imageInfo = cardFront.images["profileIcon"]
	imageInfo.image = resize.Resize(100, 0, <-imagesChan, resize.Lanczos3)
	cardFront.images["profileIcon"] = imageInfo
	imageInfo = cardFront.images["insignia"]
	imageInfo.image = resize.Resize(256, 0, <-imagesChan, resize.Lanczos3)
	cardFront.images["insignia"] = imageInfo
	imageInfo = cardFront.images["solo"]
	imageInfo.image = resize.Resize(100, 0, <-imagesChan, resize.Lanczos3)
	cardFront.images["solo"] = imageInfo
	imageInfo = cardFront.images["flex"]
	imageInfo.image = resize.Resize(100, 0, <-imagesChan, resize.Lanczos3)
	cardFront.images["flex"] = imageInfo
	if soloInfo.Tier == "" {
		soloInfo.Tier = "Unranked"
	}
	if flexInfo.Tier == "" {
		flexInfo.Tier = "Unranked"
	}
	if len(schamps) > 0 {
		imageInfo = cardFront.images["masteryBorder"]
		imageInfo.image = loadImage(fmt.Sprintf("league/mastery_border/%v.png", schamps[0].ChampionLevel))
		cardFront.images["masteryBorder"] = imageInfo

		imageInfo = cardFront.images["masteryChamp"]
		imageInfo.image = resize.Resize(93, 0, <-imagesChan, resize.Lanczos3)
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
	textData.text = fmt.Sprintf("%s %s", titlefy(string(soloInfo.Tier)), soloInfo.Rank)
	textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
	cardFront.text["soloRank"] = textData
	if flexInfo.QueueType == "RANKED_FLEX_TT" {
		textData = cardFront.text["Flex"]
		textData.text = "Flex 3v3"
		textData.point.X = 226 - textWidth(c, textData.text, textData.fontSize)/2
		cardFront.text["Flex"] = textData
	}
	textData = cardFront.text["flexRank"]
	textData.text = fmt.Sprintf("%s %s", titlefy(string(flexInfo.Tier)), flexInfo.Rank)
	textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
	cardFront.text["flexRank"] = textData
	if len(schamps) > 0 {
		textData := cardFront.text["masteryPoints"]
		textData.text = commafy(strconv.FormatInt(int64(schamps[0].ChampionPoints), 10))
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
	imageInfo.area = image.Rect(imageInfo.area.Min.X, 290, imageInfo.area.Max.X, 325)
	cardBack.images["insignia2"] = imageInfo
	textData = cardBack.text["mainRole"]
	textData.text += string(mainRoles[0])
	textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
	cardBack.text["mainRole"] = textData
	textData = cardBack.text["secondaryRole"]
	textData.text += string(mainRoles[1])
	textData.point.X -= textWidth(c, textData.text, textData.fontSize) / 2
	cardBack.text["secondaryRole"] = textData
	if len(champs) > 0 {
		var images []imageData
		text := mostChampsText()
		if len(champs) > 3 {
			images = mostChampsTemplates(3)
		} else {
			images = mostChampsTemplates(len(champs))
		}
		for i := range images {
			images[i].image = loadImage(champs[i].imageURL)
		}
		i := 0
		for k, v := range images {
			cardBack.images[strconv.FormatInt(int64(k), 10)] = v
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
			cardBack.text[fmt.Sprintf("%v%s", k, "played")] = textData
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
			cardBack.text[fmt.Sprintf("%v%s", k, "kda")] = textData
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
			cardBack.text[fmt.Sprintf("%v%s", k, "winrate")] = textData
			i++
		}
	} else if len(mainChamps) > 0 {
		var images []imageData
		text := mostChampsText()
		if len(mainChamps) > 3 {
			images = mostChampsTemplates(3)
		} else {
			images = mostChampsTemplates(len(mainChamps))
		}
		syncChan := make(chan image.Image, 3)
		for i := range images {
			go func(i int) {
				syncChan <- resize.Resize(75, 0, loadImage(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/8.8.2/img/champion/%s.png", riotChamps[int(mainChamps[i].champ)])), resize.Lanczos3)
			}(i)
		}
		for i := range images {
			images[i].image = <-syncChan
		}
		close(syncChan)
		i := 0
		for k, v := range images {
			cardBack.images[strconv.FormatInt(int64(k), 10)] = v
			textData = text["played"]
			textData.text += commafy(strconv.FormatInt(int64(champMatches[mainChamps[i].champ]), 10))
			textData.point.X = (images[k].area.Max.X-images[k].area.Min.X)/2 + images[k].area.Min.X
			for {
				if width := textWidth(c, textData.text, textData.fontSize); width <= 75 {
					textData.point.X -= width / 2
					break
				}
				textData.fontSize--
			}
			cardBack.text[fmt.Sprintf("%v%s", k, "played")] = textData
			i++
		}
	}
	oldRanks := <-oldRanksChan
	if len(oldRanks.images) > 0 {
		textData.text = "Previous ranks"
		textData.fontSize = 20
		textData.point = image.Pt(160-textWidth(c, textData.text, textData.fontSize)/2, 348)
		cardBack.text["previous"] = textData
	}
	for k, v := range oldRanks.images {
		cardBack.images[k] = v
	}
	for k, v := range oldRanks.text {
		cardBack.text[k] = v
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
	fmt.Println("Playercard created successfully!")
	draw.Draw(both, front.Bounds(), front, image.ZP, draw.Src)
	draw.Draw(both, front.Bounds().Add(image.Pt(321, 0)), back, image.ZP, draw.Src)
	return both, nil
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
