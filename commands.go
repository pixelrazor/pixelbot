package main

import (
	"bytes"
	"fmt"
	"image/png"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yuhanfang/riot/constants/region"

	"github.com/bwmarrin/discordgo"
)

var (
	embedColor = 0xb10fc6
	emojis     = map[int]string{
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
	startTime   = time.Now()
	cmdHandlers = make(map[string]cmdHandler)
)

type cmdHandler func([]string, *discordgo.Session, *discordgo.MessageCreate)

// Parse commadns from the user
func parse(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	go incrementCommandsRun()
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, m.Author.Mention()+" Stop pinging me for no reason, punk")
		return
	}
	if h, ok := cmdHandlers[args[0]]; ok {
		h(args, s, m)
	} else {
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'help' command.")
	}
}

func idToDate(s int64) time.Time {
	const discordEpoch int64 = 1420070400000
	return time.Unix(((s>>22)+discordEpoch)/1000, 0)
}
func helpcmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	msg := &discordgo.MessageEmbed{
		Title:       "__**Pixel Bot Commands**__",
		Description: "All commands can be called by prepending a '/' or by pinging me anywhere in it (ex: 'help @pixelbot' or '@pixelbot help')",
		Color:       embedColor,
		Footer: &discordgo.MessageEmbedFooter{
			IconURL: m.Author.AvatarURL("32"),
			Text:    m.Author.Username,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "about",
				Value: "View information about Pixel Bot and the developer",
			},
			{
				Name:  "ask",
				Value: "Ask pixel bot a question! (Totally not a magic 8 ball)",
			},
			{
				Name:  "cinfo <channels>",
				Value: "View information about channels. If no channels are specified, it will run on the channel the command was called in",
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
				Name:  "osu <username>",
				Value: "View the osu! stats card for the specified user",
			},
			{
				Name:  "sinfo",
				Value: "View information about this server",
			},
			{
				Name:  "stats",
				Value: "View some stats about Pixel Bot",
			},
			{
				Name:  "uinfo <users>",
				Value: "View information about users. If no users are specified, it will run on the person calling the command",
			},
			{
				Name:  "uptime",
				Value: "View how long Pixel Bot has been online for",
			},
		},
	}
	s.ChannelMessageSendEmbed(m.ChannelID, msg)
}
func leaguecmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(args) > 1 {
		leagueCommand(args[1:], s, m, "NA1")
	} else {
		s.ChannelMessageSend(m.ChannelID, "Not enough arguments, try the 'league help' command.")
		return
	}
}
func uptimecmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	t := time.Since(startTime).Round(time.Millisecond)
	if t < 24*time.Hour {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Pixel Bot has been online for %v", t))
	} else {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Pixel Bot has been online for %dd%v", t/(time.Hour*24), t%(time.Hour*24)))
	}
}
func uinfocmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Mentions) == 0 {
		singleuinfo(m.Author, m.ChannelID, m.GuildID, s)
	}
	for _, v := range m.Mentions {
		singleuinfo(v, m.ChannelID, m.GuildID, s)
	}
}
func singleuinfo(u *discordgo.User, channel, guild string, s *discordgo.Session) {
	id, err := strconv.ParseInt(u.ID, 10, 64)
	if err != nil {
		logger.Println("uinfo ParseInt:", err)
		return
	}
	created := idToDate(id)
	member, err := s.GuildMember(guild, u.ID)
	if err != nil {
		logger.Println("uinfo GuildMember:", err)
		return
	}
	join, err := discordgo.Timestamp(member.JoinedAt).Parse()
	if err != nil {
		logger.Println("uinfo JoinedAt.Parse:", err)
		return
	}
	roleMap := make(map[string]string)
	gRoles, err := s.GuildRoles(guild)
	if err != nil {
		logger.Println("uinfo GuildRoles:", err)
		return
	}
	for _, v := range gRoles {
		roleMap[v.ID] = v.Name
	}
	roles := ""
	for _, v := range member.Roles {
		roles += roleMap[v] + "\n"
	}
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%v#%v", u.Username, u.Discriminator),
		Color: embedColor,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: u.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{"ID", u.ID, true},
			{"Joined server", join.Format("January 2, 2006"), true},
			{"Joined Discord", created.Format("January 2, 2006"), true},
		},
	}
	if roles != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{fmt.Sprintf("Roles (%v)", len(member.Roles)), roles, true})
	}
	if member.Nick != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{"Nickname", member.Nick, true})
	}
	_, err = s.ChannelMessageSendEmbed(channel, embed)
}
func askcmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	responses := []string{"It is certain.",
		"It is decidedly so.",
		"Without a doubt.",
		"Yes - definitely.",
		"You may rely on it.",

		"As I see it, yes.",
		"Most likely.",
		"Outlook good.",
		"Yes.",
		"Signs point to yes.",

		"Reply hazy, try again",
		"Ask again later.",
		"Better not tell you now.",
		"Cannot predict now.",
		"Concentrate and ask again.",

		"Don't count on it.",
		"My reply is no.",
		"My sources say no.",
		"Outlook not so good.",
		"Very doubtful.",

		"Mom go boom.",
		"Big oof.",
		"The intent is to provide players with a sense of pride and accomplishment for unlocking different heroes." +
			"\n\nAs for cost, we selected initial values based upon data from the Open Beta and other adjustments made to milestone rewards before launch. Among other things, we’re looking at average per-player credit earn rates on a daily basis, and we’ll be making constant adjustments to ensure that players have challenges that are compelling, rewarding, and of course attainable via gameplay." +
			"\n\nWe appreciate the candid feedback, and the passion the community has put forth around the current topics here on Reddit, our forums and across numerous social media outlets." +
			"\n\nOur team will continue to make changes and monitor community feedback and update everyone as soon and as often as we can.",
	}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	s.ChannelMessageSend(m.ChannelID, responses[random.Intn(len(responses))])
}
func cinfocmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(args) == 0 {
		singlecinfo(m.ChannelID, s, m)
	}
	for _, v := range args[1:] {
		if match, err := regexp.MatchString("<#[0-9]*>", v); err == nil && match {
			singlecinfo(v[2:len(v)-1], s, m)
		}
	}
}
func singlecinfo(cID string, s *discordgo.Session, m *discordgo.MessageCreate) {
	channel, err := s.Channel(cID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error: could not find that channel")
		return
	}
	id, err := strconv.ParseInt(cID, 10, 64)
	if err != nil {
		logger.Println("cinfo ParseInt:", err)
		return
	}
	embed := &discordgo.MessageEmbed{
		Title:       channel.Name,
		Description: channel.Topic + " ",
		Color:       embedColor,
		Fields: []*discordgo.MessageEmbedField{
			{"ID", cID, true},
			{"Created at", idToDate(id).Format("January 2, 2006"), true},
			{"Users", fmt.Sprint(len(channel.Recipients)), true},
		},
	}
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}
func sinfocmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	iconImage, err := s.GuildIcon(m.GuildID)
	if err != nil {
		logger.Println("sinfo GuildIcon:", err)
		return
	}
	var buff bytes.Buffer
	err = png.Encode(&buff, iconImage)
	if err != nil {
		logger.Println("sinfo png.Encode:", err)
		return
	}
	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		logger.Println("sinfo Guild:", err)
		return
	}
	owner, err := s.User(guild.OwnerID)
	if err != nil {
		logger.Println("sinfo User:", err)
		return
	}
	voice, text := 0, 0
	for _, v := range guild.Channels {
		switch v.Type {
		case discordgo.ChannelTypeGuildText:
			text++
		case discordgo.ChannelTypeGuildVoice:
			voice++
		}
	}
	id, err := strconv.ParseInt(guild.ID, 10, 64)
	if err != nil {
		logger.Println("sinfo ParseInt:", err)
		return
	}
	created := idToDate(id)
	mesg := &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title: guild.Name,
			Color: embedColor,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "attachment://thumb.png",
			},
			Fields: []*discordgo.MessageEmbedField{
				{"ID", guild.ID, true},
				{"Owner", fmt.Sprintf("%v#%v", owner.Username, owner.Discriminator), true},
				{"Members", fmt.Sprint(guild.MemberCount), true},
				{"Text channels", fmt.Sprint(text), true},
				{"Voice channels", fmt.Sprint(voice), true},
				{"Created at", created.Format("January 2, 2006"), true},
				{"Region", guild.Region, true},
				{"Roles", fmt.Sprint(len(guild.Roles)), true},
				{"Custom Emojis", fmt.Sprint(len(guild.Emojis)), true},
			},
		},
		Files: []*discordgo.File{
			&discordgo.File{
				Name:   "thumb.png",
				Reader: &buff,
			},
		},
	}
	_, err = s.ChannelMessageSendComplex(m.ChannelID, mesg)
	if err != nil {
		logger.Println("sinfo message send error:", err)
	}
}
func statscmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
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
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Pixel Bot is currently on %v servers with a reach of %v people! %v commands have been run so far.",
		commafy(strconv.Itoa(len(s.State.Guilds))),
		commafy(strconv.Itoa(users)),
		commafy(strconv.FormatUint(commandsRun, 10))))
}
func aboutcmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
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
				IconURL: m.Author.AvatarURL("32"),
				Text:    m.Author.Username,
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
	s.ChannelMessageSendComplex(m.ChannelID, mesg)
}
func feedbackcmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(dmchannel, m.ChannelID+" "+m.Author.ID+": "+recombineArgs(args[1:]))
}

// Takes an array or strings (the arguments) and combines them into one string separated by spaces
func recombineArgs(args []string) string {
	var result string
	for _, v := range args {
		result += v + " "
	}
	return strings.TrimSpace(result)
}
func osucmd(args []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Error: Please specify a user")
		return
	}
	waitMesg, err := s.ChannelMessageSend(m.ChannelID, "Working on it...")
	if err != nil {
		return
	}
	name := strings.Replace(recombineArgs(args[1:]), " ", "%20", -1)
	card, err := osuPlayercard(name)
	if err != nil {
		s.ChannelMessageEdit(m.ChannelID, waitMesg.ID, err.Error())
		return
	}
	buffer := new(bytes.Buffer)
	png.Encode(buffer, card)
	mesg := &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title: "__**" + name + "**__",
			URL:   "https://osu.ppy.sh/users/" + name,
			Image: &discordgo.MessageEmbedImage{
				URL: "attachment://card.png",
			},
			Footer: &discordgo.MessageEmbedFooter{
				IconURL: m.Author.AvatarURL("32"),
				Text:    m.Author.Username,
			},
			Color: embedColor,
		},
		Files: []*discordgo.File{
			{
				Name:   "card.png",
				Reader: buffer,
			},
		},
	}
	s.ChannelMessageSendComplex(m.ChannelID, mesg)
	s.ChannelMessageDelete(m.ChannelID, waitMesg.ID)
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
		logger.Printf("League Player: %v %v\n", region, playerName)
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
					IconURL: m.Author.AvatarURL("32"),
					Text:    m.Author.Username,
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
		s.ChannelMessageSendComplex(m.ChannelID, mesg)
		s.ChannelMessageDelete(m.ChannelID, waitMesg.ID)
	case "match":
		card, name, numPlayers, err := riotMakeInGame(playerName, region)
		if err != nil {
			logger.Println("league match error:", err)
			s.ChannelMessageSend(m.ChannelID, "Summoner not found/not in game")
			return
		}
		logger.Printf("League Match: %v %v\n", region, playerName)
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
					IconURL: m.Author.AvatarURL("32"),
					Text:    m.Author.Username,
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
		logger.Printf("League Code: %v %v\n", region, playerName)
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

		logger.Printf("League Verify: %v %v\n", region, playerName)
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

		logger.Printf("League Quote: %v %v %v\n", region, playerName, quote)
		s.ChannelMessageSend(m.ChannelID, "Your summoner quote has been set to "+quote)
	case "help":
		msg := &discordgo.MessageEmbed{
			Title:       "__**League Commands**__",
			Description: "All commands can be run without specifying region (which will default to NA)\nRegions: na, br, eune, euw, jp, kr, lan, las, oce, tr, ru",
			Color:       embedColor,
			Footer: &discordgo.MessageEmbedFooter{
				IconURL: m.Author.AvatarURL("32"),
				Text:    m.Author.Username,
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
		s.ChannelMessageSendEmbed(m.ChannelID, msg)
	case "na", "br", "eune", "euw", "jp", "kr", "lan", "las", "oce", "tr", "ru", "pbe":
		leagueCommand(args[1:], s, m, riotRegions[strings.ToLower(args[0])])
	default:
		s.ChannelMessageSend(m.ChannelID, "Command not found, try the 'league help' command.")
	}
}
