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
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pixelrazor/pixelbot/league"

	"github.com/boltdb/bolt"

	"github.com/PuerkitoBio/goquery"

	"github.com/yuhanfang/riot/apiclient"
	"github.com/yuhanfang/riot/constants/champion"
	"github.com/yuhanfang/riot/constants/lane"
	"github.com/yuhanfang/riot/constants/region"
	"github.com/yuhanfang/riot/ratelimit"

	"golang.org/x/image/math/fixed"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/nfnt/resize"
)

var riotKey, riotPatch string
var httpClient = &http.Client{Timeout: 10 * time.Second}
var ctx = context.Background()
var limiter = ratelimit.NewLimiter()
var riotClient apiclient.Client
var riotChamps map[champion.Champion]string
var riotDB *bolt.DB

// Initialize some data for use later
func riotInit() error {
	riotVerified = make(map[riotVerifyKey]string)
	var err error
	riotDB, err = bolt.Open("league/riot.db", 0666, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		fmt.Println("Error opening riot.db")
		return err
	}
	err = riotDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("quotes"))
		if err != nil {
			fmt.Println("Error making quotes bucket")
			return err
		}
		return nil
	})
	riotClient = apiclient.New(riotKey, httpClient, limiter)
	var patches []string
	res, err := http.Get("https://ddragon.leagueoflegends.com/api/versions.json")
	if err != nil {
		return err
	}
	asd, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(asd, &patches)
	res.Body.Close()
	riotPatch = patches[0]
	res, err = http.Get("http://ddragon.leagueoflegends.com/cdn/" + riotPatch + "/data/en_US/champion.json")
	if err != nil {
		return err
	}
	ddchamps := new(ddchampions)
	asd, _ = ioutil.ReadAll(res.Body)
	json.Unmarshal(asd, ddchamps)
	res.Body.Close()
	riotChamps = ddchamps.toMap()
	go func() {
		for true {
			var patches []string
			<-time.After(time.Hour)
			res, err := http.Get("https://ddragon.leagueoflegends.com/api/versions.json")
			if err != nil {
				continue
			}
			asd, _ := ioutil.ReadAll(res.Body)
			json.Unmarshal(asd, &patches)
			if riotPatch != patches[0] {
				res, err = http.Get("http://ddragon.leagueoflegends.com/cdn/" + riotPatch + "/data/en_US/champion.json")
				if err != nil {
					continue
				}
				ddchamps := new(ddchampions)
				asd, _ = ioutil.ReadAll(res.Body)
				json.Unmarshal(asd, ddchamps)
				res.Body.Close()
				riotChamps = ddchamps.toMap()
				ddragon.GetChamps(patches[0])
				ddragon.GetIcons(patches[0])
				ddragon.GetRunes(patches[0])
				ddragon.GetSumms(patches[0])
			}
			riotPatch = patches[0]
			res.Body.Close()
		}
	}()
	return nil
}

func randString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[random.Intn(len(letterBytes))]
	}
	return string(b)
}
func riotVerify(playername, discordID string, region region.Region) (string, error) {
	summoner, err := riotClient.GetBySummonerName(ctx, region, playername)
	if err != nil {
		return "", err
	}
	key := riotVerifyKey{discordID, summoner.ID}
	code := randString(13)
	riotVerified[key] = code
	go func() {
		<-time.After(10 * time.Minute)
		_, ok := riotVerified[key]
		if ok {
			delete(riotVerified, key)
		}
	}()
	return code, nil
}

func riotSetQuote(discordID, pname, quote string, region region.Region) error {
	if len(quote) > 96 {
		return errors.New("Error: Quote too long")
	}
	summoner, err := riotClient.GetBySummonerName(ctx, region, pname)
	if err != nil {
		return errors.New("Error: Could not find summoner " + pname)
	}
	code, ok := riotVerified[riotVerifyKey{discordID, summoner.ID}]
	if !ok {
		return errors.New("Error: You are not currently verified for this account")
	}
	pcode, err := riotClient.GetThirdPartyCodeByID(ctx, region, summoner.ID)
	if err != nil {
		return errors.New("Unknown error occured")
	}
	if pcode != code {
		return errors.New("Error: your verification code does not match the summoner's")
	}
	err = riotDB.Update(func(tx *bolt.Tx) error {
		quotes := tx.Bucket([]byte("quotes"))
		if quotes == nil {
			return errors.New("Unknown database error occured")
		}
		err := quotes.Put([]byte(fmt.Sprint("%v%v", summoner.AccountID, region)), []byte(quote))
		if err != nil {
			return errors.New("Unknown database error occured")
		}
		return nil
	})
	return err
}

// Attempt to get a Player's Solo rank
func riotPastRanks(c *freetype.Context, username, region string) cardTemplate {
	var ranks cardTemplate
	ranks.images = make(map[string]imageData)
	ranks.text = make(map[string]textData)
	var seasonRanks []string
	var seasons []int
	res, err := http.Get("http://" + region + ".op.gg/summoner/userName=" + strings.Replace(username, " ", "+", -1))
	if err != nil {
		logger.Println("Error getting past ranks")
		return ranks
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		logger.Println("Error getting past ranks")
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

func riotAddQuote(quote string, tdata map[string]textData, c *freetype.Context) {
	fixed := strings.Replace(quote, "\n", " ", -1)
	lines := make([]string, 0)
	for len(fixed) > 32 {
		line := fixed[:32]
		if fixed[31] != ' ' && fixed[32] != ' ' {
			line += "-"
		}
		lines = append(lines, line)
		fixed = fixed[32:]
	}
	lines = append(lines, fixed)
	for i, v := range lines {
		tdata[fmt.Sprintf("qline%d", i)] = textData{
			v,
			image.Pt(160-textWidth(c, v, 14)/2, 475+i*22),
			14,
		}
	}
	return
}
func riotMakeInGame(playername string, region region.Region) (*image.RGBA, string, int, error) {
	summoner, err := riotClient.GetBySummonerName(ctx, region, playername)
	if err != nil {
		return nil, "", 0, errors.New("Summoner not found: " + err.Error())
	}
	info, err := riotClient.GetCurrentGameInfoBySummoner(ctx, region, summoner.ID)
	if err != nil {
		return nil, "", 0, errors.New("Summoner not in game: " + err.Error())
	}
	fontFile, err := ioutil.ReadFile("league/FrizQuadrataTT.TTF")
	if err != nil {
		return nil, "", 0, errors.New("Error opening font: " + err.Error())
	}
	f, err := freetype.ParseFont(fontFile)
	if err != nil {
		return nil, "", 0, errors.New("Error parsing font: " + err.Error())
	}
	names := make([]int64, len(info.Participants))
	blue := 0
	left := make(chan riotInGameCH)
	right := make(chan riotInGameCH)
	defer close(left)
	defer close(right)
	participants := make([]apiclient.CurrentGameParticipant, 0)
	for _, v := range info.Participants {
		if v.TeamId == 100 {
			participants = append([]apiclient.CurrentGameParticipant{v}, participants...)
			blue++
		} else {
			participants = append(participants, v)

		}
	}
	for i, v := range info.Participants {
		if v.TeamId == 100 {
			go riotMakeParticipant(left, f, v, i+1)
		} else {
			go riotMakeParticipant(right, f, v, i+1)
		}
		names[i] = v.SummonerId
	}
	var ingame *image.RGBA
	if len(info.Participants)-blue > blue {
		ingame = image.NewRGBA(image.Rect(0, 0, 700, (len(info.Participants)-blue)*64))
	} else {
		ingame = image.NewRGBA(image.Rect(0, 0, 700, (blue)*64))
	}
	bg := loadImage("league/ingame.png")
	draw.Draw(ingame, ingame.Bounds(), bg, image.ZP, draw.Src)
	for i := 0; i < blue; i++ {
		player := <-left
		draw.Draw(ingame, player.card.Bounds().Add(image.Pt(0, 64*(player.num-1))), player.card, image.ZP, draw.Over)
	}
	for i := 0; i < len(info.Participants)-blue; i++ {
		player := <-right
		draw.Draw(ingame, player.card.Bounds().Add(image.Pt(366, 64*(player.num-1-blue))), player.card, image.ZP, draw.Over)
	}
	/*file, _ := os.Create("test.png")
	defer file.Close()
	png.Encode(file, ingame)*/
	filename := string(region)
	for _, v := range names {
		filename += fmt.Sprintf("_%v", v)
	}
	filename += ".png"
	return ingame, filename, len(info.Participants), nil
}
func riotMakeParticipant(done chan riotInGameCH, font *truetype.Font, player apiclient.CurrentGameParticipant, num int) {
	styles := [][]image.Point{
		{
			{270, 0},
			{238, 0},
			{238, 32},
			{206, 0},
			{206, 32},
			{106, 36},
			{4, 36},
		},
		{
			{0, 0},
			{64, 0},
			{64, 32},
			{96, 0},
			{96, 32},
			{228, 36},
			{320, 36},
		},
	}
	style := styles[player.TeamId/100-1]
	card := image.NewRGBA(image.Rect(0, 0, 334, 64))
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(font)
	c.SetClip(card.Bounds())
	c.SetSrc(image.White)
	c.SetDst(card)
	champ := resize.Resize(64, 0, loadImage(fmt.Sprintf("league/champion/%v.png", player.ChampionId)), resize.Lanczos3)
	s1 := resize.Resize(32, 0, loadImage(fmt.Sprintf("league/summoners/%v.png", player.Spell1Id)), resize.Lanczos3)
	s2 := resize.Resize(32, 0, loadImage(fmt.Sprintf("league/summoners/%v.png", player.Spell2Id)), resize.Lanczos3)
	r1 := resize.Resize(32, 0, loadImage(fmt.Sprintf("league/runes/%v.png", player.Perks.PerkStyle)), resize.Lanczos3)
	r2 := resize.Resize(32, 0, loadImage(fmt.Sprintf("league/runes/%v.png", player.Perks.PerkSubStyle)), resize.Lanczos3)
	draw.Draw(card, champ.Bounds().Add(style[0]), champ, image.ZP, draw.Over)
	draw.Draw(card, s1.Bounds().Add(style[1]), s1, image.ZP, draw.Over)
	draw.Draw(card, s2.Bounds().Add(style[2]), s2, image.ZP, draw.Over)
	draw.Draw(card, r1.Bounds().Add(style[3]), r1, image.ZP, draw.Over)
	draw.Draw(card, r2.Bounds().Add(style[4]), r2, image.ZP, draw.Over)
	c.SetFontSize(16)
	c.DrawString(player.SummonerName, freetype.Pt(style[5].X-textWidth(c, player.SummonerName, 16)/2, style[5].Y))
	c.SetFontSize(12)
	c.DrawString(fmt.Sprintf("%v", num), freetype.Pt(style[6].X, style[6].Y))
	done <- riotInGameCH{
		card,
		num,
	}
}

// Create and return a playercard
func riotPlayerCard(playername *string, region region.Region) (*image.RGBA, error) {
	// Gather player data
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
		logger.Println("mastery error", result.err, open)
		return nil, errors.New("Unknown issue, try again later")
	}
	schamps := result.value.([]apiclient.ChampionMastery)
	if len(schamps) == 0 {
		return nil, errors.New("Account is too old/unused")
	}
	result = <-leaguesChan
	if result.err != nil {
		logger.Println("leagues error", result.err, open)
		return nil, errors.New("Unknown issue, try again later")
	}
	sleagues := result.value.([]apiclient.LeaguePosition)
	var soloInfo, flexInfo, highestRank apiclient.LeaguePosition
	for _, v := range sleagues {
		if v.QueueType == "RANKED_FLEX_SR" || v.QueueType == "RANKED_FLEX_TT" {
			if riotRanks[v.Tier] > riotRanks[flexInfo.Tier] {
				flexInfo = v
			}
		} else if v.QueueType == "RANKED_SOLO_5x5" {
			soloInfo = v
		}
	}
	if riotRanks[flexInfo.Tier] > riotRanks[soloInfo.Tier] {
		highestRank = flexInfo
	} else {
		highestRank = soloInfo
	}
	result = <-matchesChan
	if result.err != nil {
		logger.Println("matches error", result.err, open)
		return nil, errors.New("Unknown issue, try again later")
	}
	smatches := result.value.(*apiclient.Matchlist)
	imagesChan := make(chan image.Image, 4)
	defer close(imagesChan)
	go func() {
		imagesChan <- loadImage(fmt.Sprintf("league/profileicon/%v.png", sinfo.ProfileIconID))
		imagesChan <- loadImage(fmt.Sprintf("league/insignia/%sinsignia.png", strings.ToLower(string(highestRank.Tier))))
		imagesChan <- loadImage(fmt.Sprintf("league/rank/%s_%s.png", soloInfo.Tier, soloInfo.Rank))
		imagesChan <- loadImage(fmt.Sprintf("league/rank/%s_%s.png", flexInfo.Tier, flexInfo.Rank))
		if len(schamps) > 0 {
			imagesChan <- loadImage(fmt.Sprintf("league/champion/%d.png", schamps[0].ChampionID))
		}
	}()
	roleMatches := make(map[lane.Lane]int)
	champMatches := make(map[champion.Champion]int)
	var mainRoles [2]lane.Lane
	var mainChamps []struct {
		champ champion.Champion
		n     int
	}
	roles := make(map[string]int)
	lanes := make(map[lane.Lane]int)
	for _, v := range smatches.Matches {
		roles[v.Role]++
		lanes[v.Lane]++
		switch {
		case v.Role == "DUO_SUPPORT":
			roleMatches["Support"]++
		case v.Role == "DUO_CARRY":
			roleMatches["ADC"]++
		case v.Lane == "MID", v.Lane == "MIDDLE":
			roleMatches["Mid"]++
		case v.Lane != "NONE" && v.Lane != "BOT" && v.Lane != "BOTTOM":
			roleMatches[v.Lane]++
		}
		champMatches[v.Champion]++
	}
	if _, ok := roleMatches["NONE"]; ok {
		delete(roleMatches, "NONE")
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
	sort.Slice(mainChamps, func(i, j int) bool { return mainChamps[i].n > mainChamps[j].n })
	for i, v := range mainRoles {
		if v == "" {
			mainRoles[i] = "N/A"
		} else if mainRoles[i] != "ADC" {
			mainRoles[i] = lane.Lane(titlefy(string(mainRoles[i])))
		}
	}
	var champs []leagueMostChamps
	champs = <-opggchampsChan
	// Create images and freetype context
	fontFile, err := ioutil.ReadFile("league/FrizQuadrataTT.TTF")
	if err != nil {
		logger.Println("error opening font:", err)
		return nil, nil
	}
	f, err := freetype.ParseFont(fontFile)
	if err != nil {
		logger.Println("error parsing font:", err)
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
	logger.Printf("Playercard: %v %v\n", region, *playername)
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
	imageInfo.image = loadImage(fmt.Sprintf("league/rank_border/%sborder.png", strings.ToLower(string(highestRank.Tier))))
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
			logger.Println("Error writing text:", k, err)
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
		for i := range images {
			images[i].image = resize.Resize(75, 0, loadImage(fmt.Sprintf("league/champion/%d.png", mainChamps[i].champ)), resize.Lanczos3)
		}
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
	_ = riotDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("quotes"))
		if b == nil {
			logger.Println("Error opening quote bucket")
			return errors.New("Error opening bucket???")
		}
		v := b.Get([]byte(fmt.Sprint("%v%v", sinfo.AccountID, region)))
		if v != nil {
			riotAddQuote(string(v), cardBack.text, c)
		}
		return nil
	})
	for _, v := range cardBack.images {
		draw.Draw(back, v.area, v.image, image.ZP, draw.Over)
	}
	c.SetDst(back)
	for k, v := range cardBack.text {
		c.SetFontSize(float64(v.fontSize))
		if _, err := c.DrawString(v.text, pointToFixed(v.point)); err != nil {
			logger.Println("Error writing text:", k, err)
		}
	}
	logger.Println("Playercard created successfully!")
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
		logger.Println("Error getting text width:", err)
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
			logger.Println("error getting image from url:", response.StatusCode, path, err)
			return nil
		}
		image, err := png.Decode(response.Body)
		response.Body.Close()
		if err != nil {
			logger.Println("error decoding image from url:", err)
			return nil
		}
		return image
	}
	file, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Println("error reading file:", path, err)
		return nil
	}
	image, err := png.Decode(bytes.NewReader(file))
	if err != nil {
		logger.Println("error decoding image from file:", err)
		return nil
	}
	return image
}
