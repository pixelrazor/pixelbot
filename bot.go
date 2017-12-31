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
	file, err := os.Open("riotapi.key") // For read access.
	if err != nil {
		fmt.Println("Error opening Riot API key file:", err)
	}
	if scanner := bufio.NewScanner(file); scanner.Scan() {
		riotKey = scanner.Text()
		fmt.Println("Riot api key:", riotKey)
	}
	file.Close()
	discord, err := discordgo.New("Bot Mzk0MjM5OTA0MjMxNzE4OTEy.DSBo3g.34D4NitI3_ABo8D2zd9nWsG0amI")
	if err != nil {
		fmt.Println("woops:", err)
		return
	}
	discord.AddHandler(messageCreate)
	if err = discord.Open(); err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	if err = riotInit("7.24.2"); err != nil {
		fmt.Println("Error during riotInit():", err)
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()
}
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID {
		return
	}
	message := strings.Fields(m.Content)
	for _, v := range m.Mentions {
		if v.ID == s.State.User.ID {
			parse(message[1:], s, m)
		}

	}
}
