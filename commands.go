package main

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yuhanfang/riot/constants/region"

	"github.com/bwmarrin/discordgo"
)

var embedColor = 0xb10fc6
var emojis = map[int]string{
	1:  "1⃣",
	2:  "2⃣",
	3:  "3⃣",
	4:  "4⃣",
	5:  "5⃣",
	6:  "6⃣",
	7:  "7⃣",
	8:  "8⃣",
	9:  "9⃣",
	10: "🔟",
}
var uptime = time.Now()

// Parse commadns from the user
func parse(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	go incrementCommandsRun()
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" Stop pinging me for no reason, punk")
		return
	}
	switch strings.ToLower(args[0]) {
	case "help":
		msg := &discordgo.MessageEmbed{
			Title:       "__**Pixel Bot Commands**__",
			Description: "All commands can be called by prepending a '/' or by pinging me anywhere in it (ex: 'help @pixelbot' or '@pixelbot help')",
			Color:       embedColor,
			Footer: &discordgo.MessageEmbedFooter{
				Text: m.Author.Username,
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "about",
					Value: "View information about Pixel Bot and the developer",
				},
				{
					Name:  "feedback <message>",
					Value: "Have any questions, comments, or suggestions? Use this to have Pixel Bot inform me of what you have to say!",
				},
				{
					Name:  "help",
					Value: "View this",
				},
				{
					Name:  "league help",
					Value: "View the league commands",
				},
				{
					Name:  "stats",
					Value: "View some stats about Pixel Bot",
				},
				{
					Name:  "uptime",
					Value: "View how long Pixel Bot has been online for",
				},
			},
		}
		if m.Author.Avatar == "" {
			msg.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			msg.Footer.IconURL = m.Author.AvatarURL("32")
		}
		s.ChannelMessageSendEmbed(m.ChannelID, msg)
	case "league":
		if len(args) > 1 {
			leagueCommand(args[1:], s, m, "NA1")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Not enough arguments, try the 'league help' command.")
			return
		}
	case "uptime":
		t := time.Since(uptime).Round(time.Millisecond)
		if t < 24*time.Hour {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Pixel Bot has been online for %v", t))
		} else {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Pixel Bot has been online for %dd%v", t/(time.Hour*24), t%(time.Hour*24)))
		}

	case "stats":
		users := 0
		for _, v := range s.State.Guilds {
			if v.MemberCount == 0 {
				if guild, err := s.Guild(v.ID); err == nil {
					users += guild.MemberCount
				}
			} else {
				users += v.MemberCount
			}

		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Pixel Bot is currently on %v servers with a reach of %v people! %v commands have been run so far.", commafy(strconv.Itoa(len(s.State.Guilds))), commafy(strconv.Itoa(users)), commafy(strconv.FormatUint(commandsRun, 10))))
	case "about":
		file, err := os.Open("Avatar.png")
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "An unknown error occured")
			return
		}
		defer file.Close()
		mesg := &discordgo.MessageSend{
			Embed: &discordgo.MessageEmbed{
				Title:       "__**About Pixel Bot**__",
				Description: "Pixel Bot is an open source project created by Austin Pohlmann (Pixel Razor)",
				Color:       embedColor,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: "attachment://Avatar.png",
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: m.Author.Username,
				},
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Source Code",
						Value: "All source code is in the GitHub repository, found [here](https://github.com/pixelrazor/pixelbot).",
					},
					{
						Name:  "Support Me",
						Value: "If you like Pixel Bot or what I do, please consider supporting me by [buying me a coffee](https://www.buymeacoffee.com/iZ1Dhem).",
					},
					{
						Name:  "Suggestions",
						Value: "If you have any questions/comments/suggestions/concerns, or if you find that the bot is offline for some reason, please use the '/feedback' command or contact me at pixelrazor@gmail.com",
					},
				},
			},
			Files: []*discordgo.File{
				&discordgo.File{
					Name:   "Avatar.png",
					Reader: file,
				},
			},
		}
		if m.Author.Avatar == "" {
			mesg.Embed.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			mesg.Embed.Footer.IconURL = m.Author.AvatarURL("32")
		}
		s.ChannelMessageSendComplex(m.ChannelID, mesg)
	case "feedback":
		s.ChannelMessageSend(dmchannel, m.ChannelID+" "+m.Author.ID+": "+recombineArgs(args[1:]))
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
		logger.Printf("Playercard: %v %v\n", region, playerName)
		playercard, err := riotPlayerCard(&playerName, region)
		if err != nil {
			s.ChannelMessageEdit(m.ChannelID, waitMesg.ID, err.Error())
			return
		}
		buffer := new(bytes.Buffer)
		png.Encode(buffer, playercard)
		mesg := &discordgo.MessageSend{
			Embed: &discordgo.MessageEmbed{
				Title:       "__**" + playerName + "**__",
				Description: "op.gg link ^",
				URL:         opggLink(region) + "summoner/userName=" + strings.Replace(playerName, " ", "+", -1),
				Image: &discordgo.MessageEmbedImage{
					URL: "attachment://playercard.png",
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: m.Author.Username,
				},
				Color: embedColor,
			},
			Files: []*discordgo.File{
				{
					Name:   "playercard.png",
					Reader: buffer,
				},
			},
		}
		if m.Author.Avatar == "" {
			mesg.Embed.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			mesg.Embed.Footer.IconURL = m.Author.AvatarURL("32")
		}
		s.ChannelMessageSendComplex(m.ChannelID, mesg)
		s.ChannelMessageDelete(m.ChannelID, waitMesg.ID)
	case "match":
		card, name, numPlayers, err := riotMakeInGame(playerName, region)
		if err != nil {
			logger.Println("league match error:", err)
			s.ChannelMessageSend(m.ChannelID, "Summoner not found/not in game")
			return
		}
		buffer := new(bytes.Buffer)
		png.Encode(buffer, card)
		mesg := &discordgo.MessageSend{
			Embed: &discordgo.MessageEmbed{
				Title:       "League In-game",
				Description: "Click the numbered reactions to pull up the playercard of the corresponding player",
				Image: &discordgo.MessageEmbedImage{
					URL: "attachment://" + name,
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: m.Author.Username,
				},
				Color: embedColor,
			},
			Files: []*discordgo.File{
				&discordgo.File{
					Name:   name,
					Reader: buffer,
				},
			},
		}
		if m.Author.Avatar == "" {
			mesg.Embed.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			mesg.Embed.Footer.IconURL = m.Author.AvatarURL("32")
		}
		message, _ := s.ChannelMessageSendComplex(m.ChannelID, mesg)
		for i := 1; i <= numPlayers; i++ {
			s.MessageReactionAdd(m.ChannelID, message.ID, emojis[i])
		}
	case "code":
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
		finalMesg := discordgo.MessageSend{
			Embed: &discordgo.MessageEmbed{
				Title:       "Temporarily verify a league account",
				Description: "Please enter the following code into your league client and save (see picture for details): " + code,
				Color:       embedColor,
				Image: &discordgo.MessageEmbedImage{
					URL: "attachment://verify.png",
				},
			},
			Files: []*discordgo.File{
				&discordgo.File{
					Name:   "verify.png",
					Reader: file,
				},
			},
		}
		s.ChannelMessageSendComplex(uch.ID, &finalMesg)
	case "verify":
		err := riotCheckVerify(playerName, m.Author.ID, region)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("You have succesfully verified for %v (%v)", playerName, region))
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
		player, quote := strings.TrimSpace(preParse[:delimiter]), strings.TrimSpace(preParse[delimiter+1:])
		err := riotSetQuote(m.Author.ID, player, quote, region)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		}
		s.ChannelMessageSend(m.ChannelID, "Your summoner quote has been set to "+quote)
	case "help":
		msg := &discordgo.MessageEmbed{
			Title:       "__**League Commands**__",
			Description: "All commands can be run without specifying region (which will default to NA)\nRegions: na, br, eune, euw, jp, kr, lan, las, oce, tr, ru",
			Color:       embedColor,
			Footer: &discordgo.MessageEmbedFooter{
				Text: m.Author.Username,
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "league <region> code <player name>",
					Value: "Get a verification code for the specified account. The code expires after 24 hours or if a new one is generated. Use the verify command after following the appropriate steps",
				},
				{
					Name:  "league <region> player <player name>",
					Value: "View the stats of the specified player",
				},
				{
					Name:  "league <region> match <player name>",
					Value: "View the current match data of an in-game player",
				},
				{
					Name:  "league <region> verify <player name>",
					Value: "Check an account's verification code against the generated one",
				},
				{
					Name:  "league <region> setquote <player name> : <player quote>",
					Value: "Set the quote for the specified player (must be verified, max length 96 characters)",
				},
			},
		}
		if m.Author.Avatar == "" {
			msg.Footer.IconURL = "https://discordapp.com/assets/dd4dbc0016779df1378e7812eabaa04d.png"
		} else {
			msg.Footer.IconURL = m.Author.AvatarURL("32")
		}
		s.ChannelMessageSendEmbed(m.ChannelID, msg)
	case "na", "br", "eune", "euw", "jp", "kr", "lan", "las", "oce", "tr", "ru", "pbe":
		leagueCommand(args[1:], s, m, riotRegions[strings.ToLower(args[0])])
	default:
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'league help' command.")
	}
}
