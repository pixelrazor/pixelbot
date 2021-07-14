package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func consoleCmd(cmd []string, discord *discordgo.Session) bool {
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
		discord.ChannelMessageSend(dmchannel, "```\n"+message+"```")
	case "riotkeys":
		discord.ChannelMessageSend(dmchannel, "```\n"+"Discord id, code"+"```")
		for k, v := range riotVerified {
			discord.ChannelMessageSend(dmchannel, "```\n"+fmt.Sprint(k.dID, ", ", v)+"```")
		}
	case "message":
		if len(cmd) >= 3 {
			discord.ChannelMessageSend(cmd[1], recombineArgs(cmd[2:]))
		} else {
			discord.ChannelMessageSend(dmchannel, "```\n"+"message <channel id> <message>"+"```")
		}
	default:
		discord.ChannelMessageSend(dmchannel, "```\n"+"Command not recognized"+"```")
	}
	return false
}
