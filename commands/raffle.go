package commands

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/r-anime/ZeroTsu/config"
	"github.com/r-anime/ZeroTsu/misc"
)

// Assigns a user to participate in a raffle
func raffleParticipateCommand(s *discordgo.Session, m *discordgo.Message) {
	var (
		raffleExists bool
	)

	commandStrings := strings.SplitN(m.Content, " ", 2)

	if len(commandStrings) != 2 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "jraffle [raffle name]`")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	// Checks if such a raffle exists and adds the user ID to it if so
	misc.MapMutex.Lock()
	for index, raffle := range misc.RafflesSlice {
		if raffle.Name == strings.ToLower(commandStrings[1]) {
			raffleExists = true

			// Checks if the user already joined that raffle
			for _, ID := range raffle.ParticipantIDs {
				if ID == m.Author.ID {
					_, err := s.ChannelMessageSend(m.ChannelID, "You've already joined that raffle!")
					if err != nil {
						_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
						if err != nil {
							misc.MapMutex.Unlock()
							return
						}
						misc.MapMutex.Unlock()
						return
					}
					misc.MapMutex.Unlock()
					return
				}
			}

			// Adds user ID to the raffle list
			misc.RafflesSlice[index].ParticipantIDs = append(misc.RafflesSlice[index].ParticipantIDs, m.Author.ID)
			err := misc.RafflesWrite(misc.RafflesSlice)
			if err != nil {
				misc.CommandErrorHandler(s, m, err)
			}
			break
		}
	}
	misc.MapMutex.Unlock()
	if !raffleExists {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: No such raffle exists.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	_, err := s.ChannelMessageSend(m.ChannelID, "Success! You have entered raffle `" + commandStrings[1] + "`")
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
		if err != nil {
			return
		}
		return
	}
}

// Enters a user in a raffle if they react
func RaffleReactJoin(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Saves program from panic and continues running normally without executing the command if it happens
	defer func() {
		if rec := recover(); rec != nil {
			_, err := s.ChannelMessageSend(config.BotLogID, rec.(string))
			if err != nil {
				return
			}
		}
	}()

	// Checks if it's the slot machine emoji or the bot itself
	if r.Emoji.APIName() != "🎰" {
		return
	}
	if r.UserID == config.BotID {
		return
	}

	// Checks if that message has a raffle react set for it
	misc.MapMutex.Lock()
	for i, raffle := range misc.RafflesSlice {
		if raffle.ReactMessageID == r.MessageID {
			misc.RafflesSlice[i].ParticipantIDs = append(misc.RafflesSlice[i].ParticipantIDs, r.UserID)
			err := misc.RafflesWrite(misc.RafflesSlice)
			if err != nil {
				_, err := s.ChannelMessageSend(config.BotLogID, err.Error() +"\n" + misc.ErrorLocation(err))
				if err != nil {
					misc.MapMutex.Unlock()
					return
				}
				misc.MapMutex.Unlock()
				return
			}
			misc.MapMutex.Unlock()
			return
		}
	}
	misc.MapMutex.Unlock()
}

// Removes a user from a raffle if they unreact
func RaffleReactLeave(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	// Saves program from panic and continues running normally without executing the command if it happens
	defer func() {
		if rec := recover(); rec != nil {
			_, err := s.ChannelMessageSend(config.BotLogID, rec.(string))
			if err != nil {
				return
			}
		}
	}()

	// Checks if it's the slot machine emoji or the bot
	if r.Emoji.APIName() != "🎰" {
		return
	}
	if r.UserID == s.State.SessionID {
		return
	}

	// Checks if that message has a raffle react set for it and removes it
	misc.MapMutex.Lock()
	for index, raffle := range misc.RafflesSlice {
		if raffle.ReactMessageID == r.MessageID {
			for i := range misc.RafflesSlice[index].ParticipantIDs {
				misc.RafflesSlice[index].ParticipantIDs = append(misc.RafflesSlice[index].ParticipantIDs[:i], misc.RafflesSlice[index].ParticipantIDs[i+1:]...)
			}
			err := misc.RafflesWrite(misc.RafflesSlice)
			if err != nil {
				_, err := s.ChannelMessageSend(config.BotLogID, err.Error() +"\n" + misc.ErrorLocation(err))
				if err != nil {
					misc.MapMutex.Unlock()
					return
				}
				misc.MapMutex.Unlock()
				return
			}
			misc.MapMutex.Unlock()
			return
		}
	}
	misc.MapMutex.Unlock()
}

// Removes a user from a raffle
func raffleLeaveCommand(s *discordgo.Session, m *discordgo.Message) {
	var (
		raffleExists bool
		userInRaffle bool
	)

	commandStrings := strings.SplitN(m.Content, " ", 2)

	if len(commandStrings) != 2 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "lraffle [raffle name]`")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	// Checks if such a raffle exists and removes the user ID from it if so
	misc.MapMutex.Lock()
	for _, raffle := range misc.RafflesSlice {
		if raffle.Name == commandStrings[1] {
			raffleExists = true

			// Checks if the user already joined that raffle and removes him if so
			for i, ID := range raffle.ParticipantIDs {
				if ID == m.Author.ID {
					userInRaffle = true
					misc.RafflesSlice = append(misc.RafflesSlice[:i], misc.RafflesSlice[i+1:]...)
					break
				}
			}
			if !userInRaffle {
				_, err := s.ChannelMessageSend(m.ChannelID, "You're not in that raffle!")
				if err != nil {
					_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
					if err != nil {
						misc.MapMutex.Unlock()
						return
					}
					misc.MapMutex.Unlock()
					return
				}
				misc.MapMutex.Unlock()
				return
			}
			break
		}
	}
	misc.MapMutex.Unlock()
	if !raffleExists {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: No such raffle exists.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	_, err := s.ChannelMessageSend(m.ChannelID, "Success! You have left raffle `" + commandStrings[1] + "`")
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
		if err != nil {
			return
		}
		return
	}
}

// Creates a raffle if it doesn't exist
func craffleCommand(s *discordgo.Session, m *discordgo.Message) {
	var temp misc.Raffle

	commandStrings := strings.SplitN(m.Content, " ", 3)

	if len(commandStrings) != 3 ||
		(commandStrings[1] != "true" &&
			commandStrings[1] != "false") {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "craffle [react bool] [raffle name] `\n\n" +
			"Type `true` or `false` in `[react bool]` parameter to indicate whether you want users to be able to react to join the raffle. (default react emoji is slot machine.)")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	temp.Name = commandStrings[2]
	temp.ParticipantIDs = nil

	// Checks if that raffle already exists in the raffles slice
	misc.MapMutex.Lock()
	for _, sliceRaffle := range misc.RafflesSlice {
		if sliceRaffle.Name == temp.Name {
			_, err := s.ChannelMessageSend(m.ChannelID, "Error: Such a raffle already exists.")
			if err != nil {
				_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
				if err != nil {
					misc.MapMutex.Unlock()
					return
				}
				misc.MapMutex.Unlock()
				return
			}
			misc.MapMutex.Unlock()
			return
		}
	}
	misc.MapMutex.Unlock()

	if commandStrings[1] == "true" {
		message, err := s.ChannelMessageSend(m.ChannelID, "Raffle `" + temp.Name + "` is now active. ")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		err = s.MessageReactionAdd(message.ChannelID, message.ID, "🎰")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		temp.ReactMessageID = message.ID

		// Adds the raffle object to the raffle slice
		misc.MapMutex.Lock()
		misc.RafflesSlice = append(misc.RafflesSlice, temp)

		// Writes the raffle object to storage
		err = misc.RafflesWrite(misc.RafflesSlice)
		if err != nil {
			misc.MapMutex.Unlock()
			misc.CommandErrorHandler(s, m, err)
			return
		}
		misc.MapMutex.Unlock()
		return
	}

	_, err := s.ChannelMessageSend(m.ChannelID, "Raffle `" + temp.Name + "` is now active. Please use `" + config.BotPrefix + "jraffle " + temp.Name + "` to join the raffle.")
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
		if err != nil {
			return
		}
		return
	}
}

// Picks a random winner from those participating in the raffle
func raffleWinnerCommand(s *discordgo.Session, m *discordgo.Message) {
	var (
		winnerIndex 	int
		winnerID		string
		winnerMention 	string
	)
	commandStrings := strings.SplitN(m.Content, " ", 2)

	if len(commandStrings) != 2 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "rafflewinner [raffle name]`")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	misc.MapMutex.Lock()
	for raffleIndex, raffle := range misc.RafflesSlice {
		if raffle.Name == strings.ToLower(commandStrings[1]) {
			participantLen := len(misc.RafflesSlice[raffleIndex].ParticipantIDs)
			if participantLen == 0 {
				winnerID = "none"
				break
			}
			winnerIndex = rand.Intn(participantLen)
			winnerID = misc.RafflesSlice[raffleIndex].ParticipantIDs[winnerIndex]
			break
		}
	}
	misc.MapMutex.Unlock()

	if winnerID == "" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: No such raffle exists.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}
	if winnerID == "none" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: There is nobody to pick from to win in that raffle.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	// Parses mention if user is in the server or not
	winnerMention = fmt.Sprintf("<@%v>", winnerID)
	_, err := s.GuildMember(config.ServerID, winnerID)
	if err != nil {
		winnerMention = misc.MentionParser(s, winnerMention)
	}

	_, err = s.ChannelMessageSend(m.ChannelID, "**" + commandStrings[1] + "** winner is " + winnerMention + "! Congratulations!")
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
		if err != nil {
			return
		}
		return
	}
}

// Removes a raffle
func removeRaffleCommand(s *discordgo.Session, m *discordgo.Message) {
	commandStrings := strings.SplitN(m.Content, " ", 2)

	if len(commandStrings) != 2 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "removeraffle [raffle name]`")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	err := misc.RaffleRemove(commandStrings[1])
	if err != nil {
		_, err := s.ChannelMessageSend(m.ChannelID, err.Error())
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	_, err = s.ChannelMessageSend(m.ChannelID, "Success! Removed raffle `" + commandStrings[1] + "`.")
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
		if err != nil {
			return
		}
		return
	}
}

// Shows existing raffles
func viewRafflesCommand(s *discordgo.Session, m *discordgo.Message) {
	var message string

	commandStrings := strings.Split(m.Content, " ")

	if len(commandStrings) != 1 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "vraffle`")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	misc.MapMutex.Lock()
	if len(misc.RafflesSlice) == 0 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: There are no raffles.")
		if err != nil {
			_, err := s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				misc.MapMutex.Unlock()
				return
			}
			misc.MapMutex.Unlock()
			return
		}
		misc.MapMutex.Unlock()
		return
	}

	// Iterates through all the raffles if they exist and adds them to the message string
	for _, raffle := range misc.RafflesSlice {
		if message == "" {
			message = "Raffles:\n\n`" + raffle.Name + "`"
		} else {
			message += "\n `" + raffle.Name + "`"
		}
	}
	misc.MapMutex.Unlock()

	_, err := s.ChannelMessageSend(m.ChannelID, message)
	if err != nil {
		_, _ = s.ChannelMessageSend(config.BotLogID, err.Error())
	}
}

func init() {
	add(&command{
		execute: raffleParticipateCommand,
		aliases: []string{"joinraffle", "enterraffle"},
		trigger: "jraffle",
		desc:    "Allows you to participate in a raffle.",
		category:"raffles",
	})
	add(&command{
		execute: raffleLeaveCommand,
		aliases: []string{"leaveraffle"},
		trigger: "lraffle",
		desc:    "Removes you from a raffle.",
		category:"raffles",
	})
	add(&command{
		execute: craffleCommand,
		aliases: []string{"createraffle"},
		trigger: "craffle",
		desc:    "Create a raffle.",
		elevated: true,
		category:"raffles",
	})
	add(&command{
		execute: raffleWinnerCommand,
		aliases: []string{"pickrafflewin", "pickrafflewinner", "rafflewin", "winraffle", "raffwin"},
		trigger: "rafflewinner",
		desc:    "Picks a random winner from those participating in a raffle.",
		elevated: true,
		category:"raffles",
	})
	add(&command{
		execute: removeRaffleCommand,
		aliases: []string{"deleteraffle"},
		trigger: "removeraffle",
		desc:    "Removes a previously set raffle.",
		elevated: true,
		category:"raffles",
	})
	add(&command{
		execute: viewRafflesCommand,
		aliases: []string{"vraffles", "vraffle", "viewraffle"},
		trigger: "viewraffles",
		desc:    "Shows existing raffles.",
		elevated: true,
		category:"raffles",
	})
}