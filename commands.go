package main

import (
	"bytes"
	"image/png"
	"strings"

	"github.com/yuhanfang/riot/constants/region"

	"github.com/bwmarrin/discordgo"
)

// Parse commadns from the user
func parse(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" Stop pinging me for no reason, punk")
		return
	}
	switch strings.ToLower(args[0]) {
	case "help":
		var msg discordgo.MessageEmbed
		msg.Title = "__**Pixel Bot Commands**__"
		msg.Description = "All commands can be called by doing '/command arguments' or by pinging me anywhere in it (ex: 'help @pixelbot' or '@pixelbot help'"
		msg.Fields = make([]*discordgo.MessageEmbedField, 2)
		msg.Footer = new(discordgo.MessageEmbedFooter)
		msg.Footer.Text = m.Author.Username
		if m.Author.Avatar == "" {
			msg.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			msg.Footer.IconURL = m.Author.AvatarURL("32")
		}
		msg.Fields[0] = new(discordgo.MessageEmbedField)
		msg.Fields[1] = new(discordgo.MessageEmbedField)
		msg.Fields[0].Name = "help"
		msg.Fields[0].Value = "View this"
		msg.Fields[1].Name = "league help"
		msg.Fields[1].Value = "View the league commands"
		msg.Color = 0xb10fc6
		s.ChannelMessageSendEmbed(m.ChannelID, &msg)
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	case "league":
		if len(args) > 1 {
			leagueCommand(args[1:], s, m, "NA1")
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

func leagueCommand(args []string, s *discordgo.Session, m *discordgo.MessageCreate, region region.Region) {
	var playerName string
	if len(args) < 2 && strings.ToLower(args[0]) != "help" {
		s.ChannelMessageSend(m.ChannelID, "Not enough arguments, please see the 'league help' command")
		return
	} else if strings.ToLower(args[0]) != "help" {
		playerName = recombineArgs(args[1:])
	}
	switch strings.ToLower(args[0]) {
	case "player":
		playercard, err := riotPlayerCard(&playerName, region)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		var buffer bytes.Buffer
		png.Encode(&buffer, playercard)
		cardFile := discordgo.File{
			Name:   "playercard.png",
			Reader: &buffer,
		}
		var embed discordgo.MessageEmbed
		embed.Title = "__**" + playerName + "**__"
		embed.Image = new(discordgo.MessageEmbedImage)
		embed.Image.URL = "attachment://playercard.png"
		embed.Footer = new(discordgo.MessageEmbedFooter)
		embed.Footer.Text = m.Author.Username
		if m.Author.Avatar == "" {
			embed.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			embed.Footer.IconURL = m.Author.AvatarURL("32")
		}
		embed.Color = 0xb10fc6
		mesg := discordgo.MessageSend{
			Embed: &embed,
			Files: []*discordgo.File{&cardFile},
		}
		s.ChannelMessageSendComplex(m.ChannelID, &mesg)
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	case "match":
		s.ChannelMessageSend(m.ChannelID, "WIP, try again later please")
	case "help":
		var msg discordgo.MessageEmbed
		msg.Title = "__**League Commands**__"
		msg.Description = "All commands use NA servers by default"
		msg.Fields = make([]*discordgo.MessageEmbedField, 3)
		msg.Footer = new(discordgo.MessageEmbedFooter)
		msg.Footer.Text = m.Author.Username
		if m.Author.Avatar == "" {
			msg.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			msg.Footer.IconURL = m.Author.AvatarURL("32")
		}
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
		s.ChannelMessageDelete(m.ChannelID, m.ID)
	case "na", "br", "eune", "euw", "jp", "kr", "lan", "las", "oce", "tr", "ru", "pbe":
		leagueCommand(args[1:], s, m, riotRegions[strings.ToLower(args[0])])
	default:
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'league help' command.")
	}
}
