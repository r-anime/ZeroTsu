package commands

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/r-anime/ZeroTsu/misc"
)

// Picks one item from a specified number of item.
func pickCommand(s *discordgo.Session, m *discordgo.Message) {

	misc.MapMutex.Lock()
	guildBotLog := misc.GuildMap[m.GuildID].GuildConfig.BotLog.ID
	guildPrefix := misc.GuildMap[m.GuildID].GuildConfig.Prefix
	misc.MapMutex.Unlock()

	commandStrings := strings.SplitN(m.Content, " ", 2)

	if len(commandStrings) == 1 {
		_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Usage: `%vpick [item], [item]...`\n\nItem is anything that does not contain `,`\nUse `|` insead of `,` if you need a comma in the item", guildPrefix))
		if err != nil {
			misc.CommandErrorHandler(s, m, err, guildBotLog)
			return
		}
		return
	}

	// Splits each item individually
	items := strings.Split(commandStrings[1], "|")
	if len(items) == 1 {
		items = strings.Split(commandStrings[1], ",")
	}

	// Trims trailing and leading whitespace from each item. Also removes items that are empty
	for i := len(items)-1; i >= 0; i-- {
		items[i] = strings.TrimSpace(items[i])
		if items[i] == "" {
			items = append(items[:i], items[i+1:]...)
		}
	}

	// Check if after the split the item is still one
	if len(items) == 1 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: Not enough items. Please add at least one more item.")
		if err != nil {
			misc.CommandErrorHandler(s, m, err, guildBotLog)
			return
		}
		return
	}

	// Picks a random item
	randomItemNum := rand.Intn(len(items))
	_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**Picked:** %v", items[randomItemNum]))
	if err != nil {
		misc.CommandErrorHandler(s, m, err, guildBotLog)
		return
	}
}

func init() {
	add(&command{
		execute:  pickCommand,
		trigger:  "pick",
		aliases:  []string{"pic", "pik", "p"},
		desc:     "Picks one thing from a specified number of things.",
		category: "normal",
	})
}