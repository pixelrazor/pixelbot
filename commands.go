package main

import (
	"bytes"
	"image/png"
	"os"
	"strings"

	"github.com/yuhanfang/riot/constants/region"

	"github.com/bwmarrin/discordgo"
)

var embedColor = 0xb10fc6

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
		msg.Description = "All commands can be called by doing '/command arguments' or by pinging me anywhere in it (ex: 'help @pixelbot' or '@pixelbot help')"
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
		msg.Color = embedColor
		s.ChannelMessageSendEmbed(m.ChannelID, &msg)
	case "league":
		if len(args) > 1 {
			leagueCommand(args[1:], s, m, "NA1")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Not enough arguments, try the 'league help' command.")
			return
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
		waitMesg, err := s.ChannelMessageSend(m.ChannelID, "Working on it...")
		if err != nil {
			return
		}
		playercard, err := riotPlayerCard(&playerName, region)
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
		embed.Color = embedColor
		mesg := discordgo.MessageSend{
			Embed: &embed,
			Files: []*discordgo.File{&cardFile},
		}
		s.ChannelMessageSendComplex(m.ChannelID, &mesg)
		s.ChannelMessageDelete(m.ChannelID, waitMesg.ID)
	case "match":
		s.ChannelMessageSend(m.ChannelID, "WIP, try again later please")
	case "verify":
		code, err := riotVerify(playerName, m.Author.ID, region)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error finding summoner "+playerName)
			return
		}
		uch, err := s.UserChannelCreate(m.Author.ID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "An unknown error occured")
			return
		}
		file, err := os.Open("league/verify.png")
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "An unknown error occured")
			return
		}
		defer file.Close()
		verifile := discordgo.File{
			Name:   "verify.png",
			Reader: file,
		}
		var embed discordgo.MessageEmbed
		embed.Title = "Temporarily verify a league account"
		embed.Description = "Please enter the following code into your league client and save (see picture for details): " + code
		embed.Image = new(discordgo.MessageEmbedImage)
		embed.Image.URL = "attachment://verify.png"
		embed.Color = embedColor
		finalMesg := discordgo.MessageSend{
			Embed: &embed,
			Files: []*discordgo.File{&verifile},
		}
		s.ChannelMessageSendComplex(uch.ID, &finalMesg)
	case "setquote":
		preParse := recombineArgs(args[1:])
		delimiter := -1
		for i, v := range preParse {
			if v == ':' {
				delimiter = i
				break
			}
		}
		if delimiter < 0 {
			s.ChannelMessageSend(m.ChannelID, "Syntax error: Could not find ':'. Please see '/league help' for information on how to use the command")
			return
		}
		player := strings.TrimSpace(preParse[:delimiter])
		quote := strings.TrimSpace(preParse[delimiter+1:])
		err := riotSetQuote(m.Author.ID, player, quote, region)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		s.ChannelMessageSend(m.ChannelID, "Your summoner quote has been set to "+quote)
	case "help":
		var msg discordgo.MessageEmbed
		msg.Title = "__**League Commands**__"
		msg.Description = "All commands use NA servers by default"
		msg.Fields = make([]*discordgo.MessageEmbedField, 5)
		msg.Footer = new(discordgo.MessageEmbedFooter)
		msg.Footer.Text = m.Author.Username
		if m.Author.Avatar == "" {
			msg.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			msg.Footer.IconURL = m.Author.AvatarURL("32")
		}
		for i := range msg.Fields {
			msg.Fields[i] = new(discordgo.MessageEmbedField)
		}
		msg.Fields[0].Name = "league player <player name>"
		msg.Fields[0].Value = "View the stats of the specified player"
		msg.Fields[1].Name = "league match <player name>"
		msg.Fields[1].Value = "View the current match data of an in-game player"
		msg.Fields[2].Name = "league <region> <command>"
		msg.Fields[2].Value = "Run the previous commands for other regions.\nRegions: __na__, __br__, __eune__, __euw__, __jp__, __kr__, __lan__, __las__, __oce__, __tr__, __ru__"
		msg.Fields[3].Name = "league verify <player name>"
		msg.Fields[3].Value = "Get a verification code that lasts for 10 minutes (required to set player quote)"
		msg.Fields[4].Name = "league setquote <player name> : <player quote>"
		msg.Fields[4].Value = "Set the quote for the specified player (must be verified, max length 96 characters)"
		msg.Color = embedColor
		s.ChannelMessageSendEmbed(m.ChannelID, &msg)
	case "na", "br", "eune", "euw", "jp", "kr", "lan", "las", "oce", "tr", "ru", "pbe":
		leagueCommand(args[1:], s, m, riotRegions[strings.ToLower(args[0])])
	default:
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'league help' command.")
	}
}
