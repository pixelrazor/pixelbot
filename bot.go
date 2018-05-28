package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image/png"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/yuhanfang/riot/constants/region"

	"github.com/bwmarrin/discordgo"
)

var logger *log.Logger

func main() {
	os.Mkdir("logs", os.ModeDir)
	f, _ := os.Create("logs/" + time.Now().Format("2006-01-02_15-04-05") + ".log")
	defer f.Close()
	logger = log.New(f, "", log.LstdFlags)
	// Load the API key
	file, err := os.Open("riotapi.key")
	if err != nil {
		fmt.Println("Error opening Riot API key file:", err)
		return
	}
	if scanner := bufio.NewScanner(file); scanner.Scan() {
		riotKey = scanner.Text()
		fmt.Println("Riot api key:", riotKey)
	}
	file.Close()
	// Create the bot
	file, err = os.Open("discordapi.key")
	if err != nil {
		fmt.Println("Error opening Discord API key file:", err)
		return
	}
	var discordKey string
	if scanner := bufio.NewScanner(file); scanner.Scan() {
		discordKey = scanner.Text()
		fmt.Println("Discord api key:", discordKey)
	}
	file.Close()
	discord, err := discordgo.New("Bot " + discordKey)
	if err != nil {
		fmt.Println("Error making discordbot object:", err)
		return
	}
	// Event handler for when a message is posted on any channel
	discord.AddHandler(messageCreate)
	discord.AddHandler(messageReactAdd)
	if err = discord.Open(); err != nil {
		fmt.Println("Error opening discord", err)
		return
	}
	// Initialize the riot API stuff
	if err = riotInit(); err != nil {
		fmt.Println("Error during riotInit():", err)
		return
	}
	defer riotDB.Close()
	// I need to change the to allow for administration commands from the prompt
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	consoleQuit := make(chan struct{})
	go console(discord, consoleQuit)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	select {
	case sig := <-sc:
		fmt.Println("Quiting due to signal", sig)
	case <-consoleQuit:
		fmt.Println("Quit from console")
	}
	discord.Close()
}

func messageReactAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if s.State.User.ID == m.MessageReaction.UserID {
		return
	}
	mesg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		logger.Println("Error getting message in messageReactAdd:", err)
	}
	if mesg.Author.ID != s.State.User.ID {
		return
	}
	if len(mesg.Embeds) > 0 && mesg.Embeds[0].Title == "League In-game" {
		players := strings.Split(riotInGameFile.FindString(mesg.Embeds[0].Image.URL), "_")
		for k, v := range emojis {
			if v == m.MessageReaction.Emoji.Name {
				if k < len(players) {
					sid, err := strconv.ParseInt(players[k], 10, 64)
					if err != nil {
						logger.Println("messageReactAdd parseint:", err)
						return
					}
					summoner, err := riotClient.GetBySummonerID(ctx, region.Region(players[0]), sid)
					if err != nil {
						logger.Println("messageReactAdd summoner by id:", err)
						return
					}
					waitMesg, err := s.ChannelMessageSend(m.ChannelID, "Working on it...")
					if err != nil {
						logger.Println("messageReactAdd wait message send:", err)
						return
					}
					playercard, err := riotPlayerCard(&summoner.Name, region.Region(players[0]))
					if err != nil {
						s.ChannelMessageEdit(m.ChannelID, waitMesg.ID, err.Error())
						return
					}
					var buffer bytes.Buffer
					png.Encode(&buffer, playercard)
					cardFile := discordgo.File{
						Name:   "playercard.png",
						Reader: &buffer,
					}
					var embed discordgo.MessageEmbed
					embed.Title = "__**" + summoner.Name + "**__"
					embed.Image = new(discordgo.MessageEmbedImage)
					embed.Image.URL = "attachment://playercard.png"
					embed.Color = embedColor
					mesg := discordgo.MessageSend{
						Embed: &embed,
						Files: []*discordgo.File{&cardFile},
					}
					s.ChannelMessageSendComplex(m.ChannelID, &mesg)
					s.ChannelMessageDelete(m.ChannelID, waitMesg.ID)
				}
			}
		}
	}
}
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages made by the bot
	if m.Author.ID == s.State.User.ID {
		return
	}
	message := strings.Fields(m.Content)
	if len(message) > 0 && len(message[0]) > 2 && message[0][0] == '/' {
		message[0] = message[0][1:]
		parse(message, s, m)
	}
	for _, v := range m.Mentions {
		if v.ID == s.State.User.ID {
			pass1 := strings.Replace(m.Content, v.Mention(), "", -1)
			variant := strings.Replace(v.Mention(), "@", "@!", 1)
			pass2 := strings.Replace(pass1, variant, "", -1)
			parse(strings.Fields(pass2), s, m)
		}
	}
}
