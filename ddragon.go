package main

import (
	"fmt"
	"github.com/pixelrazor/ddrago"
	"image"
	"image/png"
	"os"
	"sync"
)

var (
	championFolderLock sync.RWMutex
)

func getChampionSquare(id int64) image.Image {
	filename := fmt.Sprintf("league/champion/%v.png", id)
	img, err := loadImage(filename)
	if err != nil {
		championFolderLock.Lock()
		defer championFolderLock.Unlock()
		// TODO: return a placeholder image on errors
		versions, _ := ddrago.Versions()
		champ, _ := ddrago.Champions(ddrago.English_UnitedStates, versions[0])
		for _, c := range champ {
			if c.Key == fmt.Sprint(id){
				square, err := c.FetchSquareImg()
				if err != nil {
					fmt.Println("shit",err)
				}
				f, _ := os.Create(filename)
				defer f.Close()
				png.Encode(f, square)
				return square
			}
		}
	}else {
		return img
	}
	return nil
}
