package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/yuhanfang/riot/constants/region"

	"github.com/bwmarrin/discordgo"
)

var (
	logger                    *log.Logger
	servers                   map[string]bool
	commandsRun               uint64
	commandsLock, serversLock sync.Mutex
	//dmchannel                 = "460949603496230914" // test bot
	dmchannel = "404261686057631748" // real bot
)

func main() {
	os.Mkdir("logs", os.ModeDir)
	f, _ := os.Create("logs/" + time.Now().Format("2006-01-02_15-04-05") + ".log")
	defer f.Close()
	logger = log.New(f, "", log.LstdFlags)
	// Load the API key
	file, err := os.Open("riotapi.key")
	if err != nil {
		fmt.Println("Error opening Riot API key file:", err)
		return
	}
	if scanner := bufio.NewScanner(file); scanner.Scan() {
		riotKey = scanner.Text()
		fmt.Println("Riot api key:", riotKey)
	}
	file.Close()
	// Create the bot
	file, err = os.Open("discordapi.key")
	if err != nil {
		fmt.Println("Error opening Discord API key file:", err)
		return
	}
	var discordKey string
	if scanner := bufio.NewScanner(file); scanner.Scan() {
		discordKey = scanner.Text()
		fmt.Println("Discord api key:", discordKey)
	}
	file.Close()
	discord, err := discordgo.New("Bot " + discordKey)
	if err != nil {
		fmt.Println("Error making discordbot object:", err)
		return
	}
	// Event handlers
	discord.AddHandler(messageCreate)
	discord.AddHandler(messageReactAdd)
	discord.AddHandler(serverJoin)
	discord.AddHandler(serverLeave)
	err = botInit()
	if err != nil {
		fmt.Println("Error in botinit:", err)
		return
	}
	if err = discord.Open(); err != nil {
		fmt.Println("Error opening discord", err)
		return
	}
	// Initialize the riot API stuff
	if err = riotInit(); err != nil {
		fmt.Println("Error during riotInit():", err)
		return
	}
	defer riotDB.Close()
	discord.UpdateStatus(0, "/help or @Pixel Bot help")
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	consoleQuit := make(chan struct{})
	go console(discord, consoleQuit)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	select {
	case sig := <-sc:
		fmt.Println("Quiting due to signal", sig)
	case <-consoleQuit:
		fmt.Println("Quit from console")
	}
	discord.Close()
}

func botInit() error {
	{
		f, err := os.Open("servers.txt")
		if err != nil {
			return err
		}
		defer f.Close()
		servers = make(map[string]bool)
		scan := bufio.NewScanner(f)
		for scan.Scan() {
			servers[scan.Text()] = true
		}
	}
	f, err := os.Open("commands.bytes")
	if err != nil {
		return err
	}
	defer f.Close()
	read := bufio.NewScanner(f)
	if read.Scan() {
		number := read.Bytes()
		commandsRun = binary.BigEndian.Uint64(number)
	}
	return nil
}
func serverJoin(s *discordgo.Session, m *discordgo.GuildCreate) {
	if !servers[m.ID] {
		serversLock.Lock()
		defer serversLock.Unlock()
		servers[m.ID] = true
		f, err := os.OpenFile("servers.txt", os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			fmt.Println("strange issue tring to open servers.txt:", err)
		}
		f.WriteString(m.ID + "\n")
		s.ChannelMessageSend(dmchannel, "I joined "+m.Name+fmt.Sprintf(" (%v members)", m.MemberCount))
	}
}
func serverLeave(s *discordgo.Session, m *discordgo.GuildDelete) {
	serversLock.Lock()
	defer serversLock.Unlock()
	delete(servers, m.ID)
	f, err := os.Create("servers.txt")
	if err != nil {
		fmt.Println("error opening servers.txt to remove someone")
		return
	}
	defer f.Close()
	for k, v := range servers {
		if v {
			f.WriteString(k + "\n")
		}
	}
	s.ChannelMessageSend(dmchannel, "I was removed from "+m.Name)
}
func incrementCommandsRun() {
	commandsLock.Lock()
	defer commandsLock.Unlock()
	commandsRun++
	f, err := os.Create("commands.bytes")
	if err != nil {
		return
	}
	cmdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(cmdBytes, commandsRun)
	f.Write(cmdBytes)
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
					sid, err := strconv.ParseInt(players[k], 10, 64)
					if err != nil {
						logger.Println("messageReactAdd parseint:", err)
						return
					}
					summoner, err := riotClient.GetBySummonerID(ctx, region.Region(players[0]), sid)
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
		consoleCmd(strings.Fields(m.Content), s, nil, true)
	}
}
