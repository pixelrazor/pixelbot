package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/boltdb/bolt"

	"github.com/bwmarrin/discordgo"
)

func console(discord *discordgo.Session, quit chan struct{}) {
	input := bufio.NewScanner(os.Stdin)
	if input == nil {
		fmt.Println("ERROR: pixel bot console will be unavailable")
		return
	}
	fmt.Print("pixelbot > ")
	for input.Scan() {
		cmd := strings.Fields(input.Text())
		quit := consoleCmd(cmd, discord, quit, false)
		if quit {
			return
		}
		fmt.Print("pixelbot > ")
	}
}

func consoleCmd(cmd []string, discord *discordgo.Session, quit chan struct{}, dm bool) bool {
	if len(cmd) == 0 {
		fmt.Println("Stop giving me empty lines")
		return false
	}
	switch strings.ToLower(cmd[0]) {
	case "servers":
		message := "|-----------------------+----------|\n"
		message += fmt.Sprintf("| %-20v  |  %7v |\n", "Server", "Members")
		message += "|-----------------------+----------|\n"
		i := 0
		for _, v := range discord.State.Guilds {
			message += fmt.Sprintf("| %-30v  |  %7v |\n", v.Name, v.MemberCount)
			i += v.MemberCount
		}
		message += "|-----------------------+----------|\n"
		message += fmt.Sprintf("| Total Users           | %8v |\n", i)
		message += "|-----------------------+----------|"
		consolePrint(discord, dm, message)
	case "riotkeys":
		consolePrint(discord, dm, "Discord id, code")
		for k, v := range riotVerified {
			consolePrint(discord, dm, fmt.Sprint(k.dID,", ", v))
		}
	case "message":
		if len(cmd) >= 3 {
			discord.ChannelMessageSend(cmd[1], recombineArgs(cmd[2:]))
		} else {
			consolePrint(discord, dm, "message <channel id> <message>")
		}
	case "quotes":
		riotDB.View(func(t *bolt.Tx) error {
			b := t.Bucket([]byte("quotes"))
			if b == nil {
				return errors.New("Error getting quotes bucket")
			}
			msg := ""
			b.ForEach(func(k, v []byte) error {
				msg += "Quote: " + string(v) + "\n"
				return nil
			})
			consolePrint(discord, dm, msg)
			return nil
		})
	case "quit":
		if quit != nil {
			fmt.Println("Quiting...")
			close(quit)
		}
		return true
	default:
		consolePrint(discord, dm, "Command not recognized")
	}
	return false
}
func consolePrint(discord *discordgo.Session, dm bool, mesg string) {
	if dm {
		discord.ChannelMessageSend(dmchannel, "```\n"+mesg+"```")
	} else {
		fmt.Println(mesg)
	}
}
