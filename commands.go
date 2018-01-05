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
	switch strings.ToLower(args[0]) {
	case "help":
		s.ChannelMessageSend(m.ChannelID, "Commands:\nleague\nhelp")
	case "league":
		if len(args) > 1 {
			leagueCommand(args[1:], s, m, "na1")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Not enough arguments, try the 'league help' command.")
		}

	default:
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'help' command.")
	}
	//s.ChannelMessageDelete(m.ChannelID, m.ID)
}

// Takes an array or strings (the arguments) and combines them into one string separated by spaces
func recombineArgs(args []string) string {
	var result string
	for _, v := range args {
		result += v + " "
	}
	return strings.TrimSpace(result)
}

func leagueCommand(args []string, s *discordgo.Session, m *discordgo.MessageCreate, region string) {
	var playerName string
	if len(args) < 2 && strings.ToLower(args[0]) != "help" {
		s.ChannelMessageSend(m.ChannelID, "Not enough arguments, please see the 'league help' command")
		return
	} else if strings.ToLower(args[0]) != "help" {
		playerName = recombineArgs(args[1:])
	}
	switch strings.ToLower(args[0]) {
	case "player":
		playercard := riotPlayerCard(playerName, region)
		var buffer bytes.Buffer
		png.Encode(&buffer, playercard)
		/*reader, writer := io.Pipe()
		go func() {
			png.Encode(writer, playercard)
		}()
		file, err := os.Open("out.png")*/
		_, err := s.ChannelFileSend(m.ChannelID, "playercard.png", &buffer)
		if err != nil {
			fmt.Println("error uploading playercard:", err)
		}
	case "match":
		s.ChannelMessageSend(m.ChannelID, "WIP, try again later please")
	case "help":
		var msg discordgo.MessageEmbed
		msg.Title = "__**League Commands**__"
		msg.Description = "All commands use NA servers by default"
		msg.Fields = make([]*discordgo.MessageEmbedField, 3)
		msg.Footer = new(discordgo.MessageEmbedFooter)
		msg.Footer.Text = m.Author.Username
		msg.Footer.IconURL = m.Author.AvatarURL("32")
		msg.Fields[0] = new(discordgo.MessageEmbedField)
		msg.Fields[1] = new(discordgo.MessageEmbedField)
		msg.Fields[2] = new(discordgo.MessageEmbedField)
		msg.Fields[0].Name = "league player <player name>"
		msg.Fields[0].Value = "View the stats of the specified player"
		msg.Fields[1].Name = "league match <player name>"
		msg.Fields[1].Value = "View the current match data of an in-game player"
		msg.Fields[2].Name = "league <region> <command>"
		msg.Fields[2].Value = "Run the previous commands for other regions.\nRegions: __na__, __br__, __eune__, __euw__, __jp__, __kr__, __lan__, __las__, __oce__, __tr__, __ru__"
		msg.Color = 0xb10fc6
		s.ChannelMessageSendEmbed(m.ChannelID, &msg)
	case "na", "br", "eune", "euw", "jp", "kr", "lan", "las", "oce", "tr", "ru", "pbe":
		leagueCommand(args[1:], s, m, riotRegions[strings.ToLower(args[0])])
	default:
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'league help' command.")
	}
}
