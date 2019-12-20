package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	ddragon "github.com/pixelrazor/pixelbot/league"

	"github.com/boltdb/bolt"

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

var riotPatch string
var httpClient = &http.Client{Timeout: 10 * time.Second}
var ctx = context.Background()
var limiter = ratelimit.NewLimiter()
var riotClient apiclient.Client
var riotChamps map[champion.Champion]string

// Initialize some data for use later
func riotInit(key string) error {
	riotVerified = make(map[riotVerifyKey]string)
	riotClient = apiclient.New(key, httpClient, limiter)
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
		ddragon.GetChamps(patches[0])
		ddragon.GetIcons(patches[0])
		ddragon.GetRunes(patches[0])
		ddragon.GetSumms(patches[0])
		for true {
			var patches []string
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
			}
			riotPatch = patches[0]
			res.Body.Close()
			<-time.After(time.Hour)
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
	key := riotVerifyKey{discordID, summoner.ID, region}
	code := randString(13)
	riotVerified[key] = code
	go func() {
		<-time.After(24 * time.Hour)
		_, ok := riotVerified[key]
		if ok {
			delete(riotVerified, key)
		}
	}()
	return code, nil
}

func riotCheckVerify(playername, discordID string, region region.Region) error {
	summoner, err := riotClient.GetBySummonerName(ctx, region, playername)
	if err != nil {
		return errors.New("Error: could not locate that account")
	}
	code, ok := riotVerified[riotVerifyKey{discordID, summoner.ID, region}]
	if !ok {
		return errors.New("Error: You are not currently verified for this account")
	}
	pcode, err := riotClient.GetThirdPartyCodeByID(ctx, region, summoner.ID)
	if err != nil {
		fmt.Println(err)
		return errors.New("Unknown error occured")
	}
	if pcode != code {
		return errors.New("Error: your verification code does not match the summoner's")
	}
	err = db.Update(func(tx *bolt.Tx) error {
		verify := tx.Bucket([]byte(riotBucket)).Bucket([]byte(riotVerifyBucket))
		err := verify.Put([]byte(fmt.Sprintf("%v%v", discordID, summoner.PUUID)), []byte("1"))
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func riotSetQuote(discordID, pname, quote string, region region.Region) error {
	if len(quote) > 96 {
		return errors.New("Error: Quote too long")
	}
	summoner, err := riotClient.GetBySummonerName(ctx, region, pname)
	if err != nil {
		return errors.New("Error: Could not find summoner " + pname)
	}
	err = db.View(func(tx *bolt.Tx) error {
		verify := tx.Bucket([]byte(riotBucket)).Bucket([]byte(riotVerifyBucket))
		if result := verify.Get([]byte(fmt.Sprintf("%v%v", discordID, summoner.PUUID))); result != nil {
			return nil
		}
		return errors.New("Error: You are not verified for this account")
	})
	if err != nil {
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		quotes := tx.Bucket([]byte(riotBucket)).Bucket([]byte(riotQuotesBucket))
		err := quotes.Put([]byte(fmt.Sprintf("%v%v", summoner.PUUID, region)), []byte(quote))
		if err != nil {
			return err
		}
		return nil
	})
	return err
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
	names := make([]string, len(info.Participants))
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
	sinfo, err := riotClient.GetBySummonerName(ctx, region, *playername)
	if err != nil {
		return nil, errors.New("Couldn't find summoner '" + *playername + "'")
	}
	*playername = sinfo.Name
	schamps, err := riotClient.GetAllChampionMasteries(ctx, region, sinfo.ID)
	if err != nil {
		logger.Println("mastery error", err)
		return nil, errors.New("Unknown issue, try again later")
	}
	if len(schamps) == 0 {
		return nil, errors.New("Account is too old/unused")
	}
	sleagues, err := riotClient.GetAllLeaguePositionsForSummoner(ctx, region, sinfo.ID)
	if err != nil {
		logger.Println("leagues error", err)
		return nil, errors.New("Unknown issue, try again later")
	}
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
	smatches, err := riotClient.GetMatchlist(ctx, region, sinfo.AccountID, nil)
	if err != nil {
		logger.Println("matches error", err)
		return nil, errors.New("Unknown issue, try again later")
	}
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
	champs := opggRankedChamps(sinfo.Name, region)
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
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(f)
	c.SetClip(front.Bounds())
	c.SetSrc(image.White)
	c.SetDst(front)
	// Load the templates and fill in the missing info.
	oldRanks := opggPastRanks(c, sinfo.Name, region)
	cardFront := summonerCardFront()
	cardBack := summonerCardBack()
	imageInfo := cardFront.images["background"]
	imageInfo.image = loadImage("league/bg.png")
	cardFront.images["background"] = imageInfo
	draw.Draw(front, cardFront.images["background"].area, cardFront.images["background"].image, cardFront.images["background"].point, draw.Over)
	draw.Draw(back, cardFront.images["background"].area, cardFront.images["background"].image, cardFront.images["background"].point, draw.Over)
	delete(cardFront.images, "background")
	imageInfo = cardFront.images["profileIcon"]
	imageInfo.image = resize.Resize(100, 0, loadImage(fmt.Sprintf("league/profileicon/%v.png", sinfo.ProfileIconID)), resize.Lanczos3)
	cardFront.images["profileIcon"] = imageInfo
	imageInfo = cardFront.images["insignia"]
	imageInfo.image = resize.Resize(256, 0, loadImage(fmt.Sprintf("league/insignia/%sinsignia.png", strings.ToLower(string(highestRank.Tier)))), resize.Lanczos3)
	cardFront.images["insignia"] = imageInfo
	imageInfo = cardFront.images["solo"]
	imageInfo.image = resize.Resize(100, 0, loadImage(fmt.Sprintf("league/rank/%s_%s.png", soloInfo.Tier, soloInfo.Rank)), resize.Lanczos3)
	cardFront.images["solo"] = imageInfo
	imageInfo = cardFront.images["flex"]
	imageInfo.image = resize.Resize(100, 0, loadImage(fmt.Sprintf("league/rank/%s_%s.png", flexInfo.Tier, flexInfo.Rank)), resize.Lanczos3)
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
		imageInfo.image = resize.Resize(93, 0, loadImage(fmt.Sprintf("league/champion/%d.png", schamps[0].ChampionID)), resize.Lanczos3)
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
			images[i].image = resize.Resize(uint(images[i].area.Dy()), 0, loadImage(champs[i].imageURL), resize.Lanczos3)
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
	_ = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(riotBucket)).Bucket([]byte(riotQuotesBucket))
		v := b.Get([]byte(fmt.Sprintf("%v%v", sinfo.PUUID, region)))
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
	/*
		imageInfo = cardFront.images["border"]
		imageInfo.image = loadImage(fmt.Sprintf("league/rank_border/%sborder.png", strings.ToLower(string(highestRank.Tier))))
		cardFront.images["border"] = imageInfo
		cardBack.images["border"] = imageInfo*/
	both := image.NewRGBA(image.Rect(0, 0, 2*340, 589))

	draw.Draw(both, front.Bounds().Add(image.Pt(10, 17)), front, image.ZP, draw.Src)
	draw.Draw(both, front.Bounds().Add(image.Pt(350, 17)), back, image.ZP, draw.Src)
	border := loadImage(fmt.Sprintf("league/rank_border/%sborder.png", strings.ToLower(string(highestRank.Tier))))
	draw.Draw(both, border.Bounds(), border, image.ZP, draw.Over)
	draw.Draw(both, border.Bounds().Add(image.Pt(340, 0)), border, image.ZP, draw.Over)
	return both, nil
}

// Wrapper for converting image.Point to fixed.Point26_6 to make things a tad cleaner
func pointToFixed(point image.Point) fixed.Point26_6 {
	return freetype.Pt(point.X, point.Y)
}
