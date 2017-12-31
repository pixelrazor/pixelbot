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

	"github.com/golang/freetype"
	"github.com/nfnt/resize"
)

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
}
type champMastery struct {
	Level  int `json:"championLevel"`
	ID     int `json:"championId"`
	Points int `json:"championPoints"`
}
type champion struct {
	Name string `json:"name"`
}
type leaguesResult []summonerLeagues
type masteryResult []champMastery

var riotKey string
var httpClient = &http.Client{Timeout: 10 * time.Second}
var riotChamps map[int]string

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
// I should probably have this unmarshal the json data as well instead of doing it right after a call to this function
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
func riotPlayerInfo(name string) (summonerInfo, leaguesResult) {
	var sinfo summonerInfo
	sleagues := new(leaguesResult)
	adjustedName := strings.Replace(name, " ", "%20", -1)
	data := getURL("https://na1.api.riotgames.com/lol/summoner/v3/summoners/by-name/" + adjustedName + "?api_key=" + riotKey)
	err := json.Unmarshal(data, &sinfo)
	if err != nil {
		fmt.Println("Error getting summoner id:", err)
		return sinfo, *sleagues
	}
	data = getURL(fmt.Sprintf("https://na1.api.riotgames.com/lol/league/v3/positions/by-summoner/%v?api_key=%s", sinfo.ID, riotKey))
	err = json.Unmarshal(data, &sleagues)
	if err != nil {
		fmt.Println("Error getting summoner leagues:", err, string(data))
		return sinfo, *sleagues
	}
	return sinfo, *sleagues
}

// This is a big one, bear with me here. I'll try to clean it and make things more modular later
func riotPlayerCard(playername string) *image.RGBA {
	sinfo, sleagues := riotPlayerInfo(playername)
	schamps := *new(masteryResult)
	data := getURL(fmt.Sprintf("https://na1.api.riotgames.com/lol/champion-mastery/v3/champion-masteries/by-summoner/%v?api_key=%s", sinfo.ID, riotKey))
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
	fmt.Println("Name:", playername)
	fmt.Println("Soloq:", soloInfo)
	fmt.Println("Flexq:", flexInfo)
	fmt.Println("Champ:", schamps[0])
	var urls [2]string
	var files [7]string
	var images [9]image.Image
	urls[0] = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/champion/%s.png", riotChamps[schamps[0].ID])
	urls[1] = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/profileicon/%v.png", sinfo.IconID)
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
	soloInfo.Tier = strings.ToTitle(strings.ToLower(soloInfo.Tier))
	flexInfo.Tier = strings.ToTitle(strings.ToLower(flexInfo.Tier))
	// Fill up the images array with the images. I need to label what each index is somewhere
	for i, v := range urls {
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
	// Draw image. There has to be some way I can tidy this up
	rgba := image.NewRGBA(image.Rect(0, 0, 320, 570))
	center := image.Pt(rgba.Bounds().Dx(), rgba.Bounds().Dy())
	images[1] = resize.Resize(100, 0, images[1], resize.Lanczos3)
	images[3] = resize.Resize(100, 0, images[3], resize.Lanczos3)
	images[4] = resize.Resize(100, 0, images[4], resize.Lanczos3)
	images[0] = resize.Resize(93, 0, images[0], resize.Lanczos3)
	images[6] = resize.Resize(256, 0, images[6], resize.Lanczos3)
	draw.Draw(rgba, rgba.Bounds(), images[2], image.Pt(images[2].Bounds().Dx()/2-center.X/2, images[2].Bounds().Dy()/2-center.Y/2), draw.Src)
	draw.Draw(rgba, image.Rect(center.X/2-116, 230, center.X/2-16, 330), images[3], image.ZP, draw.Over)
	draw.Draw(rgba, image.Rect(center.X/2+16, 230, center.X/2+116, 330), images[4], image.ZP, draw.Over)
	draw.Draw(rgba, rgba.Bounds(), images[5], image.ZP, draw.Over)
	draw.Draw(rgba, image.Rect(center.X/2-images[1].Bounds().Dx()/2, 32, center.X/2+images[1].Bounds().Dx()/2, images[1].Bounds().Dy()+32), images[1], image.ZP, draw.Src)
	draw.DrawMask(rgba, image.Rect(center.X/2-images[0].Bounds().Dx()/2, 386, center.X/2+images[0].Bounds().Dx()/2, images[0].Bounds().Dy()+386),
		images[0], image.ZP, images[8], image.ZP, draw.Over)
	draw.Draw(rgba, image.Rect(center.X/2-images[7].Bounds().Dx()/2, 380, center.X/2+images[7].Bounds().Dx()/2, images[7].Bounds().Dy()+380),
		images[7], image.ZP, draw.Over)
	draw.Draw(rgba, image.Rect(center.X/2-images[6].Bounds().Dx()/2, 190, center.X/2-images[6].Bounds().Dx()/2+images[6].Bounds().Dx(),
		images[6].Bounds().Dy()+190), images[6], image.ZP, draw.Over)
	// Draw text. Probably needs to be cleaned more than anything else
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(f)
	c.SetFontSize(32)
	c.SetClip(rgba.Bounds())
	c.SetSrc(image.White)
	pt := freetype.Pt(0, 50)
	temp := image.NewRGBA(image.Rect(0, 0, 300, 100))
	c.SetDst(temp)
	endpoint, err := c.DrawString(playername, pt)
	if err != nil {
		fmt.Println("error1:", err)
	}
	for fontSize := 30.0; int(endpoint.X>>6) > 256; fontSize -= 2 {
		c.SetFontSize(fontSize)
		endpoint, err = c.DrawString(playername, pt)
		if err != nil {
			fmt.Println("error1:", err)
		}
	}
	c.SetDst(rgba)
	pt = freetype.Pt(rgba.Bounds().Dx()/2-int(endpoint.X>>6)/2, 180)
	if _, err := c.DrawString(playername, pt); err != nil {
		fmt.Println("error1:", err)
	}

	c.SetFontSize(20)

	c.SetDst(temp)
	pt = freetype.Pt(0, 50)
	endpoint, err = c.DrawString("Solo", pt)
	if err != nil {
		fmt.Println("error2:", err)
	}
	c.SetDst(rgba)
	pt = freetype.Pt(rgba.Bounds().Dx()/2-16-50-int(endpoint.X>>6)/2, 345)
	if _, err := c.DrawString("Solo", pt); err != nil {
		fmt.Println("error2:", err)
	}

	c.SetDst(temp)
	pt = freetype.Pt(0, 50)
	endpoint, err = c.DrawString("Flex", pt)
	if err != nil {
		fmt.Println("error3:", err)
	}
	c.SetDst(rgba)
	pt = freetype.Pt(rgba.Bounds().Dx()/2+16+50-int(endpoint.X>>6)/2, 345)
	if _, err := c.DrawString("Flex", pt); err != nil {
		fmt.Println("error3:", err)
	}

	c.SetFontSize(16)
	c.SetDst(temp)
	pt = freetype.Pt(0, 50)
	endpoint, err = c.DrawString(fmt.Sprintf("%s %s", soloInfo.Tier, soloInfo.Rank), pt)
	if err != nil {
		fmt.Println("error4:", err)
	}
	c.SetDst(rgba)
	pt = freetype.Pt(rgba.Bounds().Dx()/2-16-50-int(endpoint.X>>6)/2, 345+20)
	if _, err := c.DrawString(fmt.Sprintf("%s %s", soloInfo.Tier, soloInfo.Rank), pt); err != nil {
		fmt.Println("error4:", err)
	}
	c.SetDst(temp)
	pt = freetype.Pt(0, 50)
	endpoint, err = c.DrawString(fmt.Sprintf("%s %s", flexInfo.Tier, flexInfo.Rank), pt)
	if err != nil {
		fmt.Println("error5:", err)
	}
	c.SetDst(rgba)
	pt = freetype.Pt(rgba.Bounds().Dx()/2+16+50-int(endpoint.X>>6)/2, 345+20)
	if _, err := c.DrawString(fmt.Sprintf("%s %s", flexInfo.Tier, flexInfo.Rank), pt); err != nil {
		fmt.Println("error5:", err)
	}
	c.SetDst(temp)
	pt = freetype.Pt(0, 50)
	endpoint, err = c.DrawString(commafy(strconv.FormatInt(int64(schamps[0].Points), 10)), pt)
	if err != nil {
		fmt.Println("error6:", err)
	}
	c.SetDst(rgba)
	pt = freetype.Pt(rgba.Bounds().Dx()/2-int(endpoint.X>>6)/2, 540)
	if _, err := c.DrawString(commafy(strconv.FormatInt(int64(schamps[0].Points), 10)), pt); err != nil {
		fmt.Println("error6:", err)
	}
	fmt.Println("Playercard created successfully!")
	return rgba
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
