package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type summonerInfo struct {
	IconId    int    `json:"profileIconId"`
	Name      string `json:"name"`
	Level     int    `json:"summonerLevel"`
	AccountId int    `json:"accountId"`
	Id        int    `json:"id"`
}

type summonerLeagues struct {
	QueueType string `json:"queueType"`
	Rank      string `json:"rank"`
	Tier      string `json:"tier"`
}
type leaguesResult []summonerLeagues

var riotKey string
var httpClient = &http.Client{Timeout: 10 * time.Second}

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

func getPlayerInfo(name string) (summonerInfo, leaguesResult) {
	var sinfo summonerInfo
	sleagues := new(leaguesResult)
	adjustedName := strings.Replace(name, " ", "%20", -1)
	data := getURL("https://na1.api.riotgames.com/lol/summoner/v3/summoners/by-name/" + adjustedName + "?api_key=" + riotKey)
	err := json.Unmarshal(data, &sinfo)
	if err != nil {
		fmt.Println("Error getting summoner id:", err)
		return sinfo, *sleagues
	}
	data = getURL(fmt.Sprintf("https://na1.api.riotgames.com/lol/league/v3/positions/by-summoner/%v?api_key=%s", sinfo.Id, riotKey))
	err = json.Unmarshal(data, &sleagues)
	if err != nil {
		fmt.Println("Error getting summoner leagues:", err, string(data))
		return sinfo, *sleagues
	}
	return sinfo, *sleagues
}
