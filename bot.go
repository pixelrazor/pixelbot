package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pixelrazor/pixelbot/repository"
	"github.com/yuhanfang/riot/constants/region"

	"github.com/bwmarrin/discordgo"
)

var (
	logger *log.Logger
	// TODO: get this programatically?
	// myID = "133651422154588161"
	dmchannel = "460949603496230914" // test bot
	//dmchannel = "404261686057631748" // real bot

	repo repository.Implementation
)

func main() {
	logger = log.New(os.Stdout, "", log.LstdFlags) // TODO: logrus ?
	// Load the API key
	riotKey, ok := os.LookupEnv("RIOT_API")
	if !ok {
		logger.Fatalln("Missing RIOT_API in env")
	}
	osuKey, ok := os.LookupEnv("OSU_API")
	if !ok {
		logger.Fatalln("Missing OSU_API in env")
	}
	discordKey, ok := os.LookupEnv("DISCORD_API")
	if !ok {
		logger.Fatalln("Missing DISCORD_API in env")
	}

	discord, err := discordgo.New("Bot " + discordKey)
	if err != nil {
		fmt.Println("Error making discordbot object:", err)
		return
	}

	db, err := bolt.Open("pixelbot.db", 0666, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		logger.Fatalln("Error opening pixelbot.db:", err)
	}
	defer db.Close()
	repo = repository.NewBolt(db)
	// Initialize the osu API stuff
	if err := initOsu(osuKey); err != nil {
		fmt.Println("Error during initOsu():", err)
		return
	}
	// Initialize the riot API stuff
	if err := riotInit(riotKey); err != nil {
		fmt.Println("Error during riotInit():", err)
		return
	}
	// Event handlers
	discord.AddHandler(messageCreate)
	discord.AddHandler(messageReactAdd)
	discord.AddHandler(serverJoin)
	discord.AddHandler(serverLeave)
	discord.AddHandler(onReady)
	if err := discord.Open(); err != nil {
		fmt.Println("Error opening discord", err)
		return
	}
	defer discord.Close()
	defer db.Close()
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	logger.Println("Quiting due to signal", <-sc)
}
func onReady(s *discordgo.Session, r *discordgo.Ready) {
	// TODO: fix me
	s.UpdateGameStatus(0, "/help or @Pixel Bot help")
}

func serverJoin(s *discordgo.Session, m *discordgo.GuildCreate) {
	s.ChannelMessageSend(dmchannel, "I joined "+m.Name+fmt.Sprintf(" (%v members) %v %v", m.MemberCount, m.JoinedAt, m.Unavailable))
}
func serverLeave(s *discordgo.Session, m *discordgo.GuildDelete) {
	s.ChannelMessageSend(dmchannel, "I was removed from "+m.Name+fmt.Sprintf("%v", m.Unavailable))
}
func incrementCommandsRun() {
	repo.IncrementCommandCount()
}

func messageReactAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if s.State.User.ID == m.MessageReaction.UserID {
		return
	}
	mesg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		logger.Println("Error getting message in messageReactAdd:", err)
		return
	}
	if mesg.Author.ID != s.State.User.ID {
		return
	}
	if len(mesg.Embeds) > 0 && mesg.Embeds[0].Title == "League In-game" {
		players := strings.Split(riotInGameFile.FindString(mesg.Embeds[0].Image.URL), "_")
		for k, v := range emojis {
			if v == m.MessageReaction.Emoji.Name {
				if k < len(players) {
					summoner, err := riotClient.GetBySummonerID(ctx, region.Region(players[0]), players[k])
					if err != nil {
						logger.Println("messageReactAdd summoner by id:", err)
						return
					}
					waitMesg, err := s.ChannelMessageSend(m.ChannelID, "Working on it...")
					if err != nil {
						logger.Println("messageReactAdd wait message send:", err)
						return
					}
					go incrementCommandsRun()
					playercard, err := riotPlayerCard(&summoner.Name, region.Region(players[0]))
					if err != nil {
						s.ChannelMessageEdit(m.ChannelID, waitMesg.ID, err.Error())
						return
					}
					buffer := new(bytes.Buffer)
					png.Encode(buffer, playercard)
					mesg := discordgo.MessageSend{
						Embed: &discordgo.MessageEmbed{
							Title: "__**" + summoner.Name + "**__",
							Color: embedColor,
							Image: &discordgo.MessageEmbedImage{
								URL: "attachment://playercard.png",
							},
						},
						Files: []*discordgo.File{
							&discordgo.File{
								Name:   "playercard.png",
								Reader: buffer,
							},
						},
					}
					s.ChannelMessageSendComplex(m.ChannelID, &mesg)
					s.ChannelMessageDelete(m.ChannelID, waitMesg.ID)
				}
			}
		}
	}
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
		return
	}
	for _, v := range m.Mentions {
		if v.ID == s.State.User.ID {
			pass1 := strings.Replace(m.Content, v.Mention(), "", -1)
			variant := strings.Replace(v.Mention(), "@", "@!", 1)
			pass2 := strings.Replace(pass1, variant, "", -1)
			parse(strings.Fields(pass2), s, m)
			return
		}
	}
	if m.ChannelID == dmchannel {
		consoleCmd(strings.Fields(m.Content), s)
	}
}
