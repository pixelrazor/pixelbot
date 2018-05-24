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
		if len(cmd) == 0 {
			fmt.Println("Stop giving me empty lines")
			continue
		}
		switch strings.ToLower(cmd[0]) {
		case "servers":
			fmt.Println("|-----------------------+----------|")
			fmt.Printf("| %-20v  |  %7v |\n", "Server", "Members")
			fmt.Println("|-----------------------+----------|")
			i := 0
			for _, v := range discord.State.Guilds {
				fmt.Printf("| %-20v  |  %7v |\n", v.Name, v.MemberCount)
				i += v.MemberCount
			}
			fmt.Println("|-----------------------+----------|")
			fmt.Printf("| Total Users           | %8v |\n", i)
			fmt.Println("|-----------------------+----------|")
		case "riotkeys":
			fmt.Println("Discord id, code")
			for k, v := range riotVerified {
				fmt.Println(k.dID, v)
			}
		case "help":
			fmt.Println("servers\nriotkeys\nquit")
		case "quit":
			fmt.Println("Quiting...")
			close(quit)
			return
		default:
			fmt.Println("Command not recognized")
		}
		fmt.Print("pixelbot > ")
	}
}
