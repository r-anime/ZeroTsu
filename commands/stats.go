package commands

import (
	"fmt"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/r-anime/ZeroTsu/config"
	"github.com/r-anime/ZeroTsu/misc"
)

const dateFormat = "2006-01-02"

func OnMessageChannel(s *discordgo.Session, m *discordgo.MessageCreate) {

	var channelStatsVar misc.Channel
	t := time.Now()

	// Saves program from panic and continues running normally without executing the command if it happens
	defer func() {
		if rec := recover(); rec != nil {
			_, err := s.ChannelMessageSend(config.BotLogID, rec.(string))
			if err != nil {
				return
			}
		}
	}()

	// Checks if it's within the config server and whether it's the bot
	ch, err := s.State.Channel(m.ChannelID)
	if err != nil {
		ch, err = s.Channel(m.ChannelID)
		if err != nil {
			return
		}
	}
	if ch.GuildID != config.ServerID {
		return
	}
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Pull channel info
	channel, err := s.State.Channel(m.ChannelID)
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error())
		if err != nil {
			return
		}
		return
	}

	// Sets channel params if it didn't exist before in database
	misc.MapMutex.Lock()
	if misc.ChannelStats[m.ChannelID] == nil {
		channelStatsVar.ChannelID = channel.ID
		channelStatsVar.Name = channel.Name
		channelStatsVar.RoleCount = make(map[string]int)
		channelStatsVar.RoleCount[channel.Name] = misc.GetRoleUserAmount(*s, channel.Name)

		// Removes role stat for channels without associated roles. Else turns bool to true
		if channelStatsVar.RoleCount[channel.Name] == 0 {
			channelStatsVar.RoleCount = nil
		} else {
			channelStatsVar.Optin = true
		}

		channelStatsVar.Messages = make(map[string]int)
		channelStatsVar.Exists = true
		misc.ChannelStats[m.ChannelID] = &channelStatsVar
	}
	if misc.ChannelStats[m.ChannelID].ChannelID == "" {
		misc.ChannelStats[m.ChannelID].ChannelID = channel.Name
	}

	misc.ChannelStats[m.ChannelID].Messages[t.Format(dateFormat)]++
	misc.MapMutex.Unlock()
}

// Prints all channel stats
func showStats(s *discordgo.Session, m *discordgo.Message) {

	var (
		msgs               []string
		normalChannelTotal int
		optinChannelTotal  int
	)
	t := time.Now()

	// Saves program from panic and continues running normally without executing the command if it happens
	defer func() {
		if rec := recover(); rec != nil {
			_, err := s.ChannelMessageSend(config.BotLogID, rec.(string))
			if err != nil {
				return
			}
		}
	}()

	// Sorts channel by their message use
	misc.MapMutex.Lock()
	channels := make([]*misc.Channel, len(misc.ChannelStats))
	for i := 0; i < len(misc.ChannelStats); i++ {
		for _, channel := range misc.ChannelStats {
			channels[i] = channel
			i++
		}
	}
	misc.MapMutex.Unlock()
	sort.Sort(byFrequencyChannel(channels))

	// Calculates normal channels and optin channels message totals
	misc.MapMutex.Lock()
	for chas := range misc.ChannelStats {
		if !misc.ChannelStats[chas].Optin {
			for date := range misc.ChannelStats[chas].Messages {
				normalChannelTotal += misc.ChannelStats[chas].Messages[date]
			}
		} else {
			for date := range misc.ChannelStats[chas].Messages {
				optinChannelTotal += misc.ChannelStats[chas].Messages[date]
			}
		}
	}
	misc.MapMutex.Unlock()

	// Pull guild info
	guild, err := s.State.Guild(config.ServerID)
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error())
		if err != nil {
			return
		}
		return
	}

	// Adds the channels and their stats to message and formats it
	message := "```CSS\nName:                            ([Daily Messages] | [Total Messages]) \n\n"
	for _, channel := range channels {

		// Fixes channels without ID param. Also fixes role size
		if channel.ChannelID == "" {
			misc.MapMutex.Lock()
			for id := range misc.ChannelStats {
				if misc.ChannelStats[id].ChannelID == "" {
					misc.ChannelStats[id].ChannelID = id
				}
			}
			misc.MapMutex.Unlock()
		}
		// Checks if channel exists and sets optin status
		ok := isChannelUsable(channel, guild)
		if !ok {
			continue
		}
		// Formats  and splits message
		if !channel.Optin {
			misc.MapMutex.Lock()
			message += lineSpaceFormatChannel(channel.ChannelID, false, *s)
			misc.MapMutex.Unlock()
			message += "\n"
		}
		msgs, message = splitStatMessages(msgs, message)
	}

	message += fmt.Sprintf("\nNormal Total: %d\n\n------", normalChannelTotal)
	message += "\n\nOpt-in Name:                     ([Daily Messages] | [Total Messages] | [Role Members]) \n\n"

	for _, channel := range channels {
		if channel.Optin {

			// Checks if channel exists and sets optin status
			ok := isChannelUsable(channel, guild)
			if !ok {
				continue
			}
			// Formats  and splits message
			misc.MapMutex.Lock()
			message += lineSpaceFormatChannel(channel.ChannelID, true, *s)
			misc.MapMutex.Unlock()
			msgs, message = splitStatMessages(msgs, message)
		}
	}

	message += fmt.Sprintf("\nOpt-in Total: %d\n\n------\n", optinChannelTotal)
	message += fmt.Sprintf("\nGrand Total Messages: %d\n\n", optinChannelTotal+normalChannelTotal)
	misc.MapMutex.Lock()
	message += fmt.Sprintf("\nDaily User Change: %d\n\n", misc.UserStats[t.Format(dateFormat)])
	misc.MapMutex.Unlock()

	// Final message split for last block + formatting
	msgs, message = splitStatMessages(msgs, message)
	if message != "" {
		msgs = append(msgs, message)
	}
	msgs[0] += "```"
	for i := 1; i < len(msgs); i++ {
		msgs[i] = "```CSS\n" + msgs[i] + "\n```"
	}

	for j := 0; j < len(msgs); j++ {
		_, err := s.ChannelMessageSend(m.ChannelID, msgs[j])
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error())
			if err != nil {
				return
			}
			return
		}
	}
}

// Sort functions for emoji use by message use
type byFrequencyChannel []*misc.Channel

func (e byFrequencyChannel) Len() int {
	return len(e)
}
func (e byFrequencyChannel) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
func (e byFrequencyChannel) Less(i, j int) bool {
	var (
		jTotalMessages int
		iTotalMessages int
	)
	for date := range e[j].Messages {
		jTotalMessages += e[j].Messages[date]
	}
	for date := range e[i].Messages {
		iTotalMessages += e[i].Messages[date]
	}
	return jTotalMessages < iTotalMessages
}

// Formats the line space length for the above to keep level spacing
func lineSpaceFormatChannel(id string, optin bool, s discordgo.Session) string {

	var totalMessages int
	t := time.Now()

	for date := range misc.ChannelStats[id].Messages {
		totalMessages += misc.ChannelStats[id].Messages[date]
	}
	line := fmt.Sprintf("%v", misc.ChannelStats[id].Name)
	spacesRequired := 33 - len(misc.ChannelStats[id].Name)
	for i := 0; i < spacesRequired; i++ {
		line += " "
	}
	line += fmt.Sprintf("([%d])", misc.ChannelStats[id].Messages[t.Format(dateFormat)])
	spacesRequired = 51 - len(line)
	for i := 0; i < spacesRequired; i++ {
		line += " "
	}
	line += fmt.Sprintf("| [%d])", totalMessages)
	spacesRequired = 70 - len(line)
	for i := 0; i < spacesRequired; i++ {
		line += " "
	}
	if optin {
		line += fmt.Sprintf("| [%d])\n", misc.ChannelStats[id].RoleCount[t.Format(dateFormat)])
	}

	return line
}

// Adds 1 to User Change on member join
func OnMemberJoin(s *discordgo.Session, u *discordgo.GuildMemberAdd) {
	t := time.Now()
	misc.MapMutex.Lock()
	misc.UserStats[t.Format(dateFormat)]++
	misc.MapMutex.Unlock()
}

// Removes 1 from User Change on member removal
func OnMemberRemoval(s *discordgo.Session, u *discordgo.GuildMemberRemove) {
	t := time.Now()
	misc.MapMutex.Lock()
	misc.UserStats[t.Format(dateFormat)]--
	misc.MapMutex.Unlock()
}

// Checks if specific channel stat should be printed
func isChannelUsable(channel *misc.Channel, guild *discordgo.Guild) bool {
	// Checks if channel exists and if it's optin
	for guildIndex := range guild.Channels {
		for roleIndex := range guild.Roles {
			if guild.Roles[roleIndex].Position < misc.OptinUnderPosition &&
				guild.Roles[roleIndex].Position > misc.OptinAbovePosition &&
				guild.Channels[guildIndex].Name == guild.Roles[roleIndex].Name {
				channel.Optin = true
				break
			}
		}
		if guild.Channels[guildIndex].Name == channel.Name {
			channel.Exists = true
			break
		} else {
			channel.Exists = false
		}
	}
	misc.MapMutex.Lock()
	misc.ChannelStats[channel.ChannelID] = channel
	misc.MapMutex.Unlock()

	if channel.Exists {
		return true
	}
	return false
}

// Splits the stat messages into blocks
func splitStatMessages (msgs []string, message string) ([]string, string) {
	const maxMsgLength = 1900
	if len(message) > maxMsgLength {
		msgs = append(msgs, message)
		message = ""
	}
	return msgs, message
}

// Adds channel stats command to the commandHandler
func init() {
	add(&command{
		execute:   showStats,
		trigger:  "channels",
		aliases:  []string{"channelstats", "stats"},
		desc:     "Prints channel stats.",
		elevated: true,
	})
}