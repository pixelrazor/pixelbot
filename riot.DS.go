package main

import (
	"image"
	"regexp"

	"github.com/yuhanfang/riot/constants/champion"
	"github.com/yuhanfang/riot/constants/region"
	"github.com/yuhanfang/riot/constants/tier"
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
type riotVerifyKey struct {
	dID string
	sID int64
}
type ddchamp struct {
	ID  string
	Key int `json:",string"`
}
type riotInGameCH struct {
	card *image.RGBA
	num  int
}
type ddchampions struct {
	Data map[string]ddchamp
}

func (obj *ddchampions) toMap() map[champion.Champion]string {
	newMap := make(map[champion.Champion]string)
	for _, v := range obj.Data {
		newMap[champion.Champion(v.Key)] = v.ID
	}
	return newMap
}

var riotInGameFile = regexp.MustCompile(`[A-Z]+[0-9]?(_[0-9]+)+`)
var riotVerified map[riotVerifyKey]string
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
