package main

import (
	"errors"
	"fmt"
	"github.com/golang/freetype"
	"github.com/nfnt/resize"
	"github.com/pixelrazor/osu"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"
)

var osuClient *osu.Client

func initOsu(key string) error {
	osuClient = osu.NewClient(key)
	return nil // would be cool if i had a way to verify the validity of the key
}
func osuPlayercard(name string) (*image.RGBA, error) {
	user, err := osuClient.User(name, osu.UsernameType.Name)
	if err != nil {
		logger.Println("osu api user error:", err)
		return nil, err
	}
	avatar := loadImageNoErr("https://a.ppy.sh/" + user.UserID) // fucking dumb, i should make it user.ID
	if avatar == nil {
		logger.Println("Osu playercard avatar error: null image")
		return nil, errors.New("Error: An unknown issue occured")
	}
	tops, err := osuClient.UserBest(name, osu.UsernameType.Name, osu.UserBestLimit(3))
	if err != nil {
		logger.Println("Osu playercard GetTop error:", err)
		return nil, errors.New("Error: An unknown issue occured")
	}
	topMaps := make([]string, 0)
	mods := make([]string, 0)
	for _, v := range tops {
		m, err := osuClient.Beatmaps(osu.BeatmapsWithID(v.BeatmapID))
		if err != nil {
			logger.Println("Osu playercard get beatmap error:", err)
			return nil, errors.New("Error: An unknown issue occured")
		}
		if len(m) == 0 {
			logger.Println("Osu playercard get beatmap error: empty list")
			return nil, errors.New("Error: An unknown issue occured")
		}
		topMaps = append(topMaps, fmt.Sprintf("%v [%v] %.2f*", m[0].Title, m[0].Version, m[0].Difficultyrating))
		acc := float64(v.Count300*300+v.Count100*100+v.Count50*50) / float64((v.Count300+v.Count100+v.Count50+v.Countmiss)*300) * 100
		mod := fmt.Sprintf("%.2f%% - %.2fpp", acc, v.Pp)
		if len(v.EnabledMods.List()) != 0 {
			modlist := "+"
			for _, mod := range v.EnabledMods.List() {
				modlist += mod.String() + ","
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
	acc := fmt.Sprintf("%.02f%%", user.Accuracy)
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(font)
	c.SetClip(out.Bounds())
	c.SetSrc(image.Black)
	c.SetDst(out)
	c.SetFontSize(32)
	c.DrawString(user.Username, freetype.Pt(112, 60))
	c.SetFontSize(24)
	c.DrawString(commafy(fmt.Sprint(int64(user.Level))), freetype.Pt(182, 143))
	c.DrawString(acc, freetype.Pt(182, 175))
	c.DrawString(commafy(fmt.Sprint(user.Playcount)), freetype.Pt(182, 209))
	c.DrawString(commafy(fmt.Sprint(user.TotalScore)), freetype.Pt(182, 247))
	c.DrawString(commafy(fmt.Sprint(user.RankedScore)), freetype.Pt(182, 281))
	c.DrawString(commafy(fmt.Sprint(user.PpRank)), freetype.Pt(182, 315))
	c.DrawString(commafy(fmt.Sprint(user.CountRankSSH)), freetype.Pt(486, 101))
	c.DrawString(commafy(fmt.Sprint(user.CountRankSs)), freetype.Pt(486, 145))
	c.DrawString(commafy(fmt.Sprint(user.CountRankSh)), freetype.Pt(486, 189))
	c.DrawString(commafy(fmt.Sprint(user.CountRankS)), freetype.Pt(486, 233))
	c.DrawString(commafy(fmt.Sprint(user.CountRankA)), freetype.Pt(486, 277))
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
