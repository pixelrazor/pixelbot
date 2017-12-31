package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

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
		riotPlayerCard(playerName)
		/*reader, writer := io.Pipe()
		go func() {
			png.Encode(writer, playercard)
		}()*/
		file, err := os.Open("out.png")
		_, err = s.ChannelFileSend(m.ChannelID, "out.png", file)
		if err != nil {
			fmt.Println("error uploading playercard:", err)
		}
		/*sinfo, sleagues := getPlayerInfo(playerName)
		fmt.Printf("%+v\n", sinfo)
		var msg discordgo.MessageEmbed
		msg.Title = "__**" + playerName + "**__"
		msg.Fields = make([]*discordgo.MessageEmbedField, len(sleagues))
		msg.Color = 0xb10fc6
		msg.Thumbnail = new(discordgo.MessageEmbedThumbnail)
		msg.Thumbnail.URL = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/7.24.2/img/profileicon/%v.png", sinfo.IconId)
		msg.Thumbnail.Height = 64
		msg.Thumbnail.Width = 64
		for i, v := range sleagues {
			field := new(discordgo.MessageEmbedField)
			switch v.QueueType {
			case "RANKED_FLEX_SR":
				field.Name = "Flex 5v5"
			case "RANKED_FLEX_TT":
				field.Name = "Flex 3v3"
			case "RANKED_SOLO_5x5":
				field.Name = "Solo 5v5"
			case "RANKED_TEAM_3x3":
				field.Name = "Team 3v3"
			case "RANKED_TEAM_5x5":
				field.Name = "Team 5v5"
			default:
				field.Name = "Other"
			}
			field.Value = strings.Title(strings.ToLower(v.Tier)) + " " + v.Rank
			msg.Fields[i] = field
		}
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, &msg)
		if err != nil {
			fmt.Println("Error:", err)
		}*/
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
