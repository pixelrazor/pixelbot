package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/golang/freetype"
)

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

// Converts a discord snowflake to a Time object
func idToDate(s int64) time.Time {
	const discordEpoch int64 = 1420070400000
	return time.Unix(((s>>22)+discordEpoch)/1000, 0)
}

// Load an image from a url or file
// todo: return error as well
func loadImage(path string) image.Image {
	var reader io.Reader
	if strings.Contains(path, "http") {
		response, err := http.Get(path)
		if err != nil || response.StatusCode != 200 {
			logger.Println("error getting image from url:", response.StatusCode, path, err)
			return nil
		}
		defer response.Body.Close()
		reader = response.Body
	} else {
		file, err := ioutil.ReadFile(path)
		if err != nil {
			logger.Println("error reading file:", path, err)
			return nil
		}
		reader = bytes.NewReader(file)
	}
	img, format, err := image.Decode(reader)
	if err != nil {
		logger.Println("error decoding image from file:", format, err)
		return nil
	}
	return img
}

// Calling both functions from the strings package to get all caps into titles is annoying
func titlefy(text string) string {
	return strings.Title(strings.ToLower(text))
}

// Take a number and add commas every three digits, from the right
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
