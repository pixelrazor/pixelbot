package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// Load the API key
	file, err := os.Open("riotapi.key")
	if err != nil {
		fmt.Println("Error opening Riot API key file:", err)
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
	}
	var discordKey string
	if scanner := bufio.NewScanner(file); scanner.Scan() {
		discordKey = scanner.Text()
		fmt.Println("Discord api key:", discordKey)
	}
	file.Close()
	discord, err := discordgo.New("Bot "+discordKey)
	if err != nil {
		fmt.Println("woops:", err)
		return
	}
	// Event handler for when a message is posted on any channel
	discord.AddHandler(messageCreate)
	if err = discord.Open(); err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	// Initialize the riot API stuff (currently just load the json file. I'll do checks to make sure i'm using the most up to date version later)
	if err = riotInit("7.24.2"); err != nil {
		fmt.Println("Error during riotInit():", err)
	}
	// I need to change the to allow for administration commanmds from the prompt
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages made by the bot
	if m.Author.ID == s.State.User.ID {
		return
	}
	message := strings.Fields(m.Content)
	if message[0][0] == '/' {
		message[0] = message[0][1:]
		parse(message, s, m)
	}
	for _, v := range m.Mentions {
		if v.ID == s.State.User.ID {
			parse(message[1:], s, m)
		}

	}
}
