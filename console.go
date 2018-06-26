package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
		consolePrint(discord, dm, "|-----------------------+----------|")
		consolePrint(discord, dm, fmt.Sprintf("| %-20v  |  %7v |", "Server", "Members"))
		consolePrint(discord, dm, "|-----------------------+----------|")
		i := 0
		for _, v := range discord.State.Guilds {
			consolePrint(discord, dm, fmt.Sprintf("| %-20v  |  %7v |", v.Name, v.MemberCount))
			i += v.MemberCount
		}
		consolePrint(discord, dm, "|-----------------------+----------|")
		consolePrint(discord, dm, fmt.Sprintf("| Total Users           | %8v |", i))
		consolePrint(discord, dm, "|-----------------------+----------|")
	case "riotkeys":
		consolePrint(discord, dm, "Discord id, code")
		for k, v := range riotVerified {
			consolePrint(discord, dm, fmt.Sprint(k.dID, v))
		}
	case "message":
		if len(cmd) >= 3 {
			discord.ChannelMessageSend(cmd[1], recombineArgs(cmd[2:]))
		} else {
			consolePrint(discord, dm, "message <channel id> <message>")
		}
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
		discord.ChannelMessageSend(dmchannel, mesg)
	} else {
		fmt.Println(mesg)
	}
}
