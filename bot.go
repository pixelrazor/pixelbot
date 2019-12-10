package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/yuhanfang/riot/constants/region"
	"image/png"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	logger    *log.Logger
	dmchannel = "460949603496230914" // test bot
	//dmchannel = "404261686057631748" // real bot
)

func main() { /*
		os.Mkdir("logs", os.ModeDir)
		f, _ := os.Create("logs/" + time.Now().Format("2006-01-02_15-04-05") + ".log")
		defer f.Close()
		logger = log.New(f, "", log.LstdFlags)*/
	logger = log.New(os.Stdout, "", log.LstdFlags)
	// Load the API key
	riotKey := os.Getenv("RIOT_API")
	osuKey := os.Getenv("OSU_API")
	discordKey := os.Getenv("DISCORD_API")
	fmt.Println("Riot api key:", riotKey)
	fmt.Println("Osu api key:", osuKey)
	fmt.Println("Discord api key:", discordKey)
	discord, err := discordgo.New("Bot " + discordKey)
	if err != nil {
		fmt.Println("Error making discordbot object:", err)
		return
	}
	err = initDB()
	if err != nil {
		fmt.Println("Error initializing db:", err)
		return
	}
	// Initialize the osu API stuff
	if err = initOsu(osuKey); err != nil {
		fmt.Println("Error during initOsu():", err)
		return
	}
	// Initialize the riot API stuff
	if err = riotInit(riotKey); err != nil {
		fmt.Println("Error during riotInit():", err)
		return
	}
	// Event handlers
	discord.AddHandler(messageCreate)
	discord.AddHandler(messageReactAdd)
	discord.AddHandler(serverJoin)
	discord.AddHandler(serverLeave)
	discord.AddHandler(onReady)
	err = botInit()
	if err != nil {
		fmt.Println("Error in botinit:", err)
		return
	}
	if err = discord.Open(); err != nil {
		fmt.Println("Error opening discord", err)
		return
	}
	defer db.Close()
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	select {
	case sig := <-sc:
		fmt.Println("Quiting due to signal", sig)
	}
	discord.Close()
}
func onReady(s *discordgo.Session, r *discordgo.Ready) {
	s.UpdateStatus(0, "/help or @Pixel Bot help")
}
func botInit() error {
	cmdHandlers["help"] = helpcmd
	cmdHandlers["about"] = aboutcmd
	cmdHandlers["uptime"] = uptimecmd
	cmdHandlers["league"] = leaguecmd
	cmdHandlers["osu"] = osucmd
	cmdHandlers["stats"] = statscmd
	cmdHandlers["feedback"] = feedbackcmd
	cmdHandlers["uinfo"] = uinfocmd
	cmdHandlers["cinfo"] = cinfocmd
	cmdHandlers["sinfo"] = sinfocmd
	cmdHandlers["ask"] = askcmd
	return nil
}
func serverJoin(s *discordgo.Session, m *discordgo.GuildCreate) {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(generalBucket)).Bucket([]byte(generalServersBucket))
		server := b.Get([]byte(m.ID))
		if len(server) == 0 {
			b.Put([]byte(m.ID),[]byte{1})
			s.ChannelMessageSend(dmchannel, "I joined "+m.Name+fmt.Sprintf(" (%v members)", m.MemberCount))
		}
		return nil
	})
}
func serverLeave(s *discordgo.Session, m *discordgo.GuildDelete) {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(generalBucket)).Bucket([]byte(generalServersBucket))
		b.Delete([]byte(m.ID))
		return nil
	})
	s.ChannelMessageSend(dmchannel, "I was removed from "+m.Name)
}
func incrementCommandsRun() {
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(generalBucket))
		val := binary.BigEndian.Uint64(b.Get([]byte("commands")))
		cmdBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(cmdBytes, val+1)
		b.Put([]byte("commands"), cmdBytes)
		return nil
	})
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
