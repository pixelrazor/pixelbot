package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	if err = discord.Open(); err != nil {
		fmt.Println("Error adding event handler", err)
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
