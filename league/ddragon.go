package ddragon

import (
	"encoding/json"
	"fmt"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {
	var patches []string
	res, err := http.Get("https://ddragon.leagueoflegends.com/api/versions.json")
	if err != nil {
		log.Panic(err)
	}
	asd, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(asd, &patches)
	res.Body.Close()
	fmt.Println("Getting champs")
	GetChamps(patches[0])
	fmt.Println("Getting summoner spells")
	GetSumms(patches[0])
	fmt.Println("Getting icons")
	GetIcons(patches[0])
	fmt.Println("Getting runes")
	GetRunes(patches[0])
}

// GetChamps downloads all square champion images to ./league/champion/
func GetChamps(patch string) {
	res, err := http.Get("http://ddragon.leagueoflegends.com/cdn/" + patch + "/data/en_US/champion.json")
	if err != nil {
		log.Panic(err)
	}
	rawchamps := new(ddWithID)
	asd, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(asd, rawchamps)
	res.Body.Close()
	champs := rawchamps.toMap()
	os.Mkdir("league/champion", os.ModeDir)
	for k, v := range champs {
		if _, err := os.Stat(fmt.Sprintf("league/champion/%v.png", k)); os.IsExist(err) {
			continue
		}
		res, err := http.Get("http://ddragon.leagueoflegends.com/cdn/" + patch + "/img/champion/" + v + ".png")
		if err != nil {
			fmt.Println("Couldn't download champion", v, err)
			continue
		}
		image, err := png.Decode(res.Body)
		if err != nil {
			fmt.Println("Error decoding png", v, err)
			res.Body.Close()
			continue
		}
		res.Body.Close()
		out, _ := os.Create(fmt.Sprintf("league/champion/%v.png", k))
		png.Encode(out, image)
		out.Close()
	}
}

// GetSumms downloads all summoner spell images to ./league/summoners/
func GetSumms(patch string) {
	res, err := http.Get("http://ddragon.leagueoflegends.com/cdn/" + patch + "/data/en_US/summoner.json")
	if err != nil {
		log.Panic(err)
	}
	asd, _ := ioutil.ReadAll(res.Body)
	rawsumms := new(ddWithID)
	json.Unmarshal(asd, rawsumms)
	res.Body.Close()
	summs := rawsumms.toMap()
	os.Mkdir("league/summoners", os.ModeDir)
	for k, v := range summs {
		if _, err := os.Stat(fmt.Sprintf("league/summoners/%v.png", k)); os.IsExist(err) {
			continue
		}
		res, err := http.Get("http://ddragon.leagueoflegends.com/cdn/" + patch + "/img/spell/" + v + ".png")
		if err != nil {
			fmt.Println("Couldn't download summoner", v, err)
			continue
		}
		image, err := png.Decode(res.Body)
		if err != nil {
			fmt.Println("decode", k, v, err)
			res.Body.Close()
			continue
		}
		res.Body.Close()
		out, err := os.Create(fmt.Sprintf("league/summoners/%v.png", k))
		png.Encode(out, image)
		out.Close()
	}
}

// GetIcons downloads all profile icons to ./league/profileicon/
func GetIcons(patch string) {
	res, err := http.Get("http://ddragon.leagueoflegends.com/cdn/" + patch + "/data/en_US/profileicon.json")
	if err != nil {
		log.Panic(err)
	}
	asd, _ := ioutil.ReadAll(res.Body)
	icons := new(ddicons)
	json.Unmarshal(asd, icons)
	res.Body.Close()
	os.Mkdir("league/profileicon", os.ModeDir)
	for k := range icons.Data {
		if _, err := os.Stat(fmt.Sprintf("league/profileicon/%v.png", k)); os.IsExist(err) {
			continue
		}
		res, err := http.Get("http://ddragon.leagueoflegends.com/cdn/" + patch + "/img/profileicon/" + k + ".png")
		if err != nil {
			fmt.Println("Couldn't download icon", k, err)
			continue
		}
		image, err := png.Decode(res.Body)
		if err != nil {
			fmt.Println("decode icon", k, err)
			res.Body.Close()
			continue
		}
		res.Body.Close()
		out, _ := os.Create(fmt.Sprintf("league/profileicon/%v.png", k))
		png.Encode(out, image)
		out.Close()
	}
}

// GetRunes downloads all rune images to ./league/runes/
func GetRunes(patch string) {
	res, err := http.Get("http://ddragon.leagueoflegends.com/cdn/" + patch + "/data/en_US/runesReforged.json")
	if err != nil {
		fmt.Println("error with get")
		return
	}
	asd, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	runes := new(ddRunes)
	json.Unmarshal(asd, runes)
	os.Mkdir("league/runes", os.ModeDir)
	for _, i := range *runes {
		if _, err := os.Stat(fmt.Sprintf("league/runes/%v.png", i.ID)); os.IsNotExist(err) {
			res, _ := http.Get("https://ddragon.leagueoflegends.com/cdn/img/" + i.Icon)
			image, _ := png.Decode(res.Body)
			res.Body.Close()
			out, _ := os.Create(fmt.Sprintf("league/runes/%v.png", i.ID))
			png.Encode(out, image)
			out.Close()
		}
		for _, j := range i.Slots {
			for _, k := range j.Runes {
				if _, err := os.Stat(fmt.Sprintf("league/runes/%v.png", k.ID)); os.IsNotExist(err) {
					res, _ := http.Get("https://ddragon.leagueoflegends.com/cdn/img/" + k.Icon)
					image, _ := png.Decode(res.Body)
					res.Body.Close()
					out, _ := os.Create(fmt.Sprintf("league/runes/%v.png", k.ID))
					png.Encode(out, image)
					out.Close()
				}
			}
		}
	}
}

type ddRunes []struct {
	ID    int
	Icon  string
	Slots []struct {
		Runes []struct {
			ID   int
			Icon string
		}
	}
}
type ddicons struct {
	Data map[string]struct{}
}
type ddWithID struct {
	Data map[string]struct {
		ID  string
		Key int `json:",string"`
	}
}

func (obj *ddWithID) toMap() map[int]string {
	newMap := make(map[int]string)
	for _, v := range obj.Data {
		newMap[v.Key] = v.ID
	}
	return newMap
}
