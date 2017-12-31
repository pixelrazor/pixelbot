package main

import (
	"bytes"
	"fmt"
	"image/png"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Parse commadns from the user
func parse(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	switch args[0] {
	case "help":
		s.ChannelMessageSend(m.ChannelID, "Commands:\nleague\nhelp")
	case "league":
		if len(args) > 1 {
			leagueCommand(args[1:], s, m)
		} else {
			s.ChannelMessageSend(m.ChannelID, "Not enough arguments, try the 'league help' command.")
		}

	default:
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'help' command.")
	}
}

// Takes an array or strings (the arguments) and combines them into one string separated by spaces
func recombineArgs(args []string) string {
	var result string
	for _, v := range args {
		result += v + " "
	}
	return strings.TrimSpace(result)
}

func leagueCommand(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	var playerName string
	if len(args) < 2 && args[0] != "help" {
		s.ChannelMessageSend(m.ChannelID, "Not enough arguments, please see the 'league help' command")
		return
	} else if args[0] != "help" {
		playerName = recombineArgs(args[1:])
	}
	playerName = playerName
	switch args[0] {
	case "player":
		// I need to get the image data from this function and give that to s.ChannelFileSend() instead of using a file
		playercard := riotPlayerCard(playerName)
		buffer := bytes.NewBuffer(nil)
		if buffer == nil {
			fmt.Println("Error creating buffer")
		}
		png.Encode(buffer, playercard)
		/*reader, writer := io.Pipe()
		go func() {
			png.Encode(writer, playercard)
		}()
		file, err := os.Open("out.png")*/
		_, err := s.ChannelFileSend(m.ChannelID, "playercard.png", buffer)
		if err != nil {
			fmt.Println("error uploading playercard:", err)
		}
	case "match":
		s.ChannelMessageSend(m.ChannelID, "Shit isn't working yet")
	case "help":
		var msg discordgo.MessageEmbed
		msg.Title = "__**League Commands**__"
		msg.Fields = make([]*discordgo.MessageEmbedField, 2)
		msg.Fields[0] = new(discordgo.MessageEmbedField)
		msg.Fields[1] = new(discordgo.MessageEmbedField)
		msg.Fields[0].Name = "league player <player name>"
		msg.Fields[0].Value = "View the stats of the specified player"
		msg.Fields[1].Name = "league match <player name>"
		msg.Fields[1].Value = "View the current match data of an in-game player"
		msg.Color = 0xb10fc6
		s.ChannelMessageSendEmbed(m.ChannelID, &msg)
	default:
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'league help' command.")
	}
}
