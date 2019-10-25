package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang/freetype"
	"github.com/nfnt/resize"
)

type osuUser struct {
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	Count300      string `json:"count300"`
	Count100      string `json:"count100"`
	Count50       string `json:"count50"`
	Playcount     string `json:"playcount"`
	RankedScore   string `json:"ranked_score"`
	TotalScore    string `json:"total_score"`
	PpRank        string `json:"pp_rank"`
	Level         string `json:"level"`
	PpRaw         string `json:"pp_raw"`
	Accuracy      string `json:"accuracy"`
	CountRankSs   string `json:"count_rank_ss"`
	CountRankSSH  string `json:"count_rank_ssh"`
	CountRankS    string `json:"count_rank_s"`
	CountRankSh   string `json:"count_rank_sh"`
	CountRankA    string `json:"count_rank_a"`
	Country       string `json:"country"`
	PpCountryRank string `json:"pp_country_rank"`
}
type osuTopMap struct {
	BeatmapID   string `json:"beatmap_id"`
	Score       string `json:"score"`
	Maxcombo    string `json:"maxcombo"`
	Count50     int64  `json:"count50,string"`
	Count100    int64  `json:"count100,string"`
	Count300    int64  `json:"count300,string"`
	Countmiss   int64  `json:"countmiss,string"`
	Countkatu   string `json:"countkatu"`
	Countgeki   string `json:"countgeki"`
	Perfect     string `json:"perfect"`
	EnabledMods uint64 `json:"enabled_mods,string"`
	UserID      string `json:"user_id"`
	Date        string `json:"date"`
	Rank        string `json:"rank"`
	Pp          string `json:"pp"`
}
type osuMap struct {
	BeatmapsetID     string `json:"beatmapset_id"`
	BeatmapID        string `json:"beatmap_id"`
	Approved         string `json:"approved"`
	TotalLength      string `json:"total_length"`
	HitLength        string `json:"hit_length"`
	Version          string `json:"version"`
	FileMd5          string `json:"file_md5"`
	DiffSize         string `json:"diff_size"`
	DiffOverall      string `json:"diff_overall"`
	DiffApproach     string `json:"diff_approach"`
	DiffDrain        string `json:"diff_drain"`
	Mode             string `json:"mode"`
	ApprovedDate     string `json:"approved_date"`
	LastUpdate       string `json:"last_update"`
	Artist           string `json:"artist"`
	Title            string `json:"title"`
	Creator          string `json:"creator"`
	CreatorID        string `json:"creator_id"`
	Bpm              string `json:"bpm"`
	Source           string `json:"source"`
	Tags             string `json:"tags"`
	GenreID          string `json:"genre_id"`
	LanguageID       string `json:"language_id"`
	FavouriteCount   string `json:"favourite_count"`
	Playcount        string `json:"playcount"`
	Passcount        string `json:"passcount"`
	MaxCombo         string `json:"max_combo"`
	Difficultyrating string `json:"difficultyrating"`
}

var (
	osukey  string
	osuMods = []string{
		"NF",
		"Easy",
		"TouchDevice",
		"HD",
		"HR",
		"SD",
		"DT",
		"RX",
		"HT",
		"NC",
		"FL",
		"Autoplay",
		"SO",
		"AP",
		"PF",
		"4K",
		"5K",
		"6K",
		"7K",
		"8K",
		"FadeIn",
		"Random",
		"Cinema",
		"TP",
		"9K",
		"KeyCoop",
		"1K",
		"3K",
		"2K",
		"ScoreV2",
		"LastMod",
	}
)

func osuPlayercard(name string) (*image.RGBA, error) {
	user := make([]osuUser, 0)
	resp, err := http.Get("https://osu.ppy.sh/api/get_user?k=94353e099b9c422c6f02fc3faf5e989e72573cb1&type=string&u=" + name)
	if err != nil || resp.StatusCode >= 400 {
		return nil, errors.New("Error: user '" + name + "' not found")
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&user)
	resp.Body.Close()
	if err != nil {
		logger.Println("Osu playercard decode error:", err)
		return nil, errors.New("Error: An unknown issue occured")
	}
	if len(user) != 1 {
		return nil, errors.New("Error: player not found")
	}
	resp, err = http.Get("https://a.ppy.sh/" + user[0].UserID)
	if err != nil {
		logger.Println("Osu playercard avatar error:", err)
		return nil, errors.New("Error: An unknown issue occured")
	}
	avatar, _, err := image.Decode(resp.Body)
	resp.Body.Close()
	if err != nil {
		logger.Println("Osu playercard avatar error:", err)
		return nil, errors.New("Error: An unknown issue occured")
	}
	tops, err := osuGetTop(name)
	if err != nil {
		logger.Println("Osu playercard GetTop error:", err)
		return nil, errors.New("Error: An unknown issue occured")
	}
	topMaps := make([]string, 0)
	mods := make([]string, 0)
	for _, v := range tops {
		m, _ := osuGetMap(v.BeatmapID)
		stars, _ := strconv.ParseFloat(m.Difficultyrating, 64)
		topMaps = append(topMaps, fmt.Sprintf("%v [%v] %.2f*", m.Title, m.Version, stars))
		pp, _ := strconv.ParseFloat(v.Pp, 64)
		acc := float64(v.Count300*300+v.Count100*100+v.Count50*50) / float64((v.Count300+v.Count100+v.Count50+v.Countmiss)*300) * 100
		mod := fmt.Sprintf("%.2f%% - %.2fpp", acc, pp)
		if v.EnabledMods != 0 {
			modlist := "+"
			for i := uint(0); v.EnabledMods != 0; i++ {
				if (1<<i)&v.EnabledMods != 0 {
					modlist += osuMods[i] + ","
				}
				v.EnabledMods &= ^(1 << i)
			}
			mod = modlist[:len(modlist)-1] + " - " + mod
		}
		mods = append(mods, mod)
	}
	temp, err := os.Open("osu/template.png")
	if err != nil {
		logger.Println("Osu playercard template error:", err)
		return nil, errors.New("Error: An unknown issue occured")
	}
	defer temp.Close()
	base, err := png.Decode(temp)
	if err != nil {
		logger.Println("Osu playercard template decode error:", err)
		return nil, errors.New("Error: An unknown issue occured")
	}
	avatar = resize.Resize(64, 0, squareImage(avatar), resize.Lanczos3)
	out := image.NewRGBA(base.Bounds())
	draw.Draw(out, base.Bounds(), base, image.ZP, draw.Over)
	draw.Draw(out, image.Rect(16, 16, 80, 80), avatar, image.ZP, draw.Over)
	fontFile, err := ioutil.ReadFile("osu/Aller_Rg.ttf")
	if err != nil {
		panic(err)
	}
	font, err := freetype.ParseFont(fontFile)
	if err != nil {
		panic(err)
	}
	flt, _ := strconv.ParseFloat(user[0].Accuracy, 64)
	acc := fmt.Sprintf("%.02f%%", flt)
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(font)
	c.SetClip(out.Bounds())
	c.SetSrc(image.Black)
	c.SetDst(out)
	c.SetFontSize(32)
	c.DrawString(user[0].Username, freetype.Pt(112, 60))
	c.SetFontSize(24)
	if index := strings.Index(user[0].Level, "."); index != -1 {
		c.DrawString(commafy(user[0].Level[:index]), freetype.Pt(182, 143))
	} else {
		c.DrawString(commafy(user[0].Level), freetype.Pt(182, 143))
	}
	c.DrawString(acc, freetype.Pt(182, 175))
	c.DrawString(commafy(user[0].Playcount), freetype.Pt(182, 209))
	c.DrawString(commafy(user[0].TotalScore), freetype.Pt(182, 247))
	c.DrawString(commafy(user[0].RankedScore), freetype.Pt(182, 281))
	c.DrawString(commafy(user[0].PpRank), freetype.Pt(182, 315))
	c.DrawString(commafy(user[0].CountRankSSH), freetype.Pt(486, 101))
	c.DrawString(commafy(user[0].CountRankSs), freetype.Pt(486, 145))
	c.DrawString(commafy(user[0].CountRankSh), freetype.Pt(486, 189))
	c.DrawString(commafy(user[0].CountRankS), freetype.Pt(486, 233))
	c.DrawString(commafy(user[0].CountRankA), freetype.Pt(486, 277))
	for i, v := range topMaps {
		size := 18
		for textWidth(c, v, size) > 430 {
			size--
		}
		c.DrawString(v, freetype.Pt(783-textWidth(c, v, size)/2, 128+i*72))
		size = 18
		for textWidth(c, mods[i], size) > 430 {
			size--
		}
		c.DrawString(mods[i], freetype.Pt(783-textWidth(c, mods[i], size)/2, 148+i*72))
	}
	return out, nil
}
func osuGetMap(id string) (*osuMap, error) {
	res, err := http.Get("https://osu.ppy.sh/api/get_beatmaps?k=94353e099b9c422c6f02fc3faf5e989e72573cb1&b=" + id)
	if err != nil || res.StatusCode >= 400 {
		return nil, err
	}
	defer res.Body.Close()
	maps := make([]osuMap, 0)
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&maps); err != nil || len(maps) == 0 {
		return nil, err
	}
	return &maps[0], err
}
func osuGetTop(id string) ([]osuTopMap, error) {
	res, err := http.Get("https://osu.ppy.sh/api/get_user_best?k=94353e099b9c422c6f02fc3faf5e989e72573cb1&type=string&limit=3&u=" + id)
	if err != nil || res.StatusCode >= 400 {
		return nil, err
	}
	defer res.Body.Close()
	maps := make([]osuTopMap, 0)
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&maps); err != nil || len(maps) == 0 {
		return nil, err
	}
	return maps, err
}
func squareImage(i image.Image) *image.RGBA {
	width, height, short := i.Bounds().Dx(), i.Bounds().Dy(), 0
	if width > height {
		short = height
	} else {
		short = width
	}
	square := image.NewRGBA(image.Rect(0, 0, short, short))
	draw.Draw(square, square.Bounds(), i, image.Pt(width/2+width%2-short/2-short%2, height/2+height%2-short/2-short%2), draw.Src)
	return square
}
