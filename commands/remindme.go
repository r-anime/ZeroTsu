package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/r-anime/ZeroTsu/config"
	"github.com/r-anime/ZeroTsu/misc"
)

func remindMeCommand(s *discordgo.Session, m *discordgo.Message) {

	var (
		remindMeObject misc.RemindMe
		userID string
		flag bool
		dummySlice misc.RemindMeSlice
	)

	// Checks if message contains filtered words, which would not be allowed as a remind
	badWordExists, _ := isFiltered(s, m)
	if badWordExists {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: Usage of server filtered words in the remind command is not allowed.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	commandStrings := strings.SplitN(m.Content, " ", 3)

	if len(commandStrings) < 3 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "remindme [time] [message]` \n\n"+
			"Time is in #w#d#h#m format, such as 2w1d12h30m for 2 weeks, 1 day, 12 hours, 30 minutes.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	// Figures out the date to show the message
	Date, perma, err := misc.ResolveTimeFromString(commandStrings[1])
	if err != nil {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: Invalid time given.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}
	if perma {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: Cannot use that time. Please use another.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	// Saves the userID in a separate variable
	s.RWMutex.Lock()
	userID = m.Author.ID
	s.RWMutex.Unlock()

	// Saves the remindMe data to an object of type remindMe
	s.RWMutex.Lock()
	remindMeObject.CommandChannel = m.ChannelID
	s.RWMutex.Unlock()
	misc.MapMutex.Lock()
	_, ok := misc.RemindMeMap[userID]
	if ok {
		remindMeObject.RemindID = len(misc.RemindMeMap[userID].RemindMeSlice) + 1
		flag = true
	} else {
		remindMeObject.RemindID = 1
	}
	misc.MapMutex.Unlock()
	remindMeObject.Date = Date
	remindMeObject.Message = commandStrings[2]

	// Adds the above object to the remindMe map where all of the remindMes are kept and writes them to disk
	misc.MapMutex.Lock()
	if !flag {
		misc.RemindMeMap[userID] = &dummySlice
	}
	misc.RemindMeMap[userID].RemindMeSlice = append(misc.RemindMeMap[userID].RemindMeSlice, remindMeObject)
	_, err = misc.RemindMeWrite(misc.RemindMeMap)
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

	_, err = s.ChannelMessageSend(m.ChannelID, "Success! You will be reminded of the message on _" + Date.Format("2006-01-02 15:04") + "_.")
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
		if err != nil {
			return
		}
		return
	}
}

func viewRemindMe(s *discordgo.Session, m *discordgo.Message) {
	var (
		userID 		string
		remindMes 	[]string
		message		string
	)

	s.RWMutex.Lock()
	userID = m.Author.ID
	s.RWMutex.Unlock()

	// Checks if the user has any reminds
	misc.MapMutex.Lock()
	_, ok := misc.RemindMeMap[userID]
	if !ok {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: No saved reminds for you found.")
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
	misc.MapMutex.Unlock()

	commandStrings := strings.Split(m.Content, " ")

	if len(commandStrings) != 1 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "reminds")
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
	for _, remind := range 	misc.RemindMeMap[userID].RemindMeSlice {
		formattedMessage := fmt.Sprintf("`%v` - _%v_ - ID: %v", remind.Message, remind.Date.Format("2006-01-02 15:04"), remind.RemindID)
		remindMes = append(remindMes, formattedMessage)
	}
	misc.MapMutex.Unlock()

	// Splits the message objects into multiple messages if it's too big
	remindMes, message = splitRemindsMessages(remindMes, message)

	// Limits the size it can display so it isn't abused
	if len(remindMes) > 4 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: The message size of all of the reminds is too big to display." +
			" Please wait them out or never use this command again.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	for _, remind := range remindMes {
		_, err := s.ChannelMessageSend(m.ChannelID, remind)
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
	}
}

func removeRemindMe(s *discordgo.Session, m *discordgo.Message) {
	var (
		userID 		string
		remindID	int
		flag		bool
	)

	s.RWMutex.Lock()
	userID = m.Author.ID
	s.RWMutex.Unlock()

	// Checks if the user has any reminds
	misc.MapMutex.Lock()
	_, ok := misc.RemindMeMap[userID]
	if !ok {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: No saved reminds found for you to delete.")
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
	misc.MapMutex.Unlock()

	commandStrings := strings.Split(m.Content, " ")

	if len(commandStrings) != 2 {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `" + config.BotPrefix + "removeremind [ID]")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	remindID, err := strconv.Atoi(commandStrings[1])
	if err != nil {
		_, err := s.ChannelMessageSend(m.ChannelID, "Error: Please input only a number as the second parameter.")
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}

	// Deletes the remind from the map and writes to disk
	misc.MapMutex.Lock()
	for index, remind := range misc.RemindMeMap[userID].RemindMeSlice {
		if remind.RemindID == remindID {

			// Deletes either the entire value or just the remind from the slice
			if len(misc.RemindMeMap[userID].RemindMeSlice) == 1 {
				delete(misc.RemindMeMap, userID)
			} else {
				misc.RemindMeMap[userID].RemindMeSlice = append(misc.RemindMeMap[userID].RemindMeSlice[:index], misc.RemindMeMap[userID].RemindMeSlice[index+1:]...)
			}

			flag = true

			_, err := misc.RemindMeWrite(misc.RemindMeMap)
			if err != nil {
				_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
				if err != nil {
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

	// Prints success or error based on whether it deleted anything above
	if flag {
		_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Sucesss: Deleted remind with ID %v.", remindID))
		if err != nil {
			_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
			if err != nil {
				return
			}
			return
		}
		return
	}
	_, err = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: No such remind with that ID found."))
	if err != nil {
		_, err = s.ChannelMessageSend(config.BotLogID, err.Error() + "\n" + misc.ErrorLocation(err))
		if err != nil {
			return
		}
		return
	}
}

// Splits the view reminds messages into blocks
func splitRemindsMessages (msgs []string, message string) ([]string, string) {
	const maxMsgLength = 1900
	if len(message) > maxMsgLength {
		msgs = append(msgs, message)
		message = ""
	}
	return msgs, message
}

func init() {
	add(&command{
		execute:  remindMeCommand,
		trigger:  "remindme",
		aliases:  []string{"remind", "setremind"},
		desc:     "Reminds you of the message after the command after a period of time. Either messages you or pings you if it cannot.",
		elevated: false,
		category: "normal",
	})
	add(&command{
		execute:  viewRemindMe,
		trigger:  "viewreminds",
		aliases:  []string{"viewremindmes", "viewremindme", "viewremind"},
		desc:     "Shows you what reminds you have currently set.",
		elevated: false,
		category: "normal",
	})
	add(&command{
		execute:  removeRemindMe,
		trigger:  "removeremind",
		aliases:  []string{"removeremindme", "deleteremind", "deleteremindme"},
		desc:     "Removes a previously set remindme.",
		elevated: false,
		category: "normal",
	})
}