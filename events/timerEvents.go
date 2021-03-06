package events

import (
	"fmt"
	"github.com/r-anime/ZeroTsu/common"
	"github.com/r-anime/ZeroTsu/db"
	"github.com/r-anime/ZeroTsu/embeds"
	"github.com/r-anime/ZeroTsu/entities"
	"github.com/r-anime/ZeroTsu/functionality"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"

	"github.com/r-anime/ZeroTsu/config"
)

const feedCheckLifespanHours = 720

var redditFeedBlock Block

type Block struct {
	sync.RWMutex
	Block bool
}

// Write Events
func WriteEvents(s *discordgo.Session, _ *discordgo.Ready) {
	var (
		t                time.Time
		randomPlayingMsg string
	)

	for range time.NewTicker(30 * time.Minute).C {
		t = time.Now()
		rand.Seed(t.UnixNano())

		// Updates playing status
		entities.Mutex.RLock()
		if len(config.PlayingMsg) > 1 {
			randomPlayingMsg = config.PlayingMsg[rand.Intn(len(config.PlayingMsg))]
		}
		entities.Mutex.RUnlock()
		if randomPlayingMsg != "" {
			_ = s.UpdateGameStatus(0, randomPlayingMsg)
		}

		// Sends server count to bot list sites if it's the public ZeroTsu
		GuildIds.RLock()
		guildCountStr := strconv.Itoa(len(GuildIds.Ids))
		GuildIds.RUnlock()
		functionality.SendServers(guildCountStr, s)
	}
}

// Common Timer Events
func CommonEvents(s *discordgo.Session, _ *discordgo.Ready) {
	var (
		guildIds []string
		guildID  string
	)

	for range time.NewTicker(2 * time.Minute).C {
		GuildIds.RLock()
		for guildID := range GuildIds.Ids {
			guildIds = append(guildIds, guildID)
		}
		GuildIds.RUnlock()

		for _, guildID = range guildIds {
			// Handles RemindMes
			remindMeHandler(s, guildID)
		}

		// Handles Reddit Feeds
		feedHandler(s, guildIds)

		guildIds = []string{}
	}
}

// remindMeHandler handles sending remindMe messages when called if it's time.
// Sends either a DM, or, failing that, a ping in the channel the remindMe was set.
func remindMeHandler(s *discordgo.Session, guildID string) {
	var (
		writeFlag bool
		t         = time.Now()
	)

	if entities.SharedInfo.GetRemindMesMap() == nil || len(entities.SharedInfo.GetRemindMesMap()) == 0 {
		return
	}

	for userID, remindMeSlice := range entities.SharedInfo.GetRemindMesMap() {
		if remindMeSlice == nil || remindMeSlice.GetRemindMeSlice() == nil || len(remindMeSlice.GetRemindMeSlice()) == 0 {
			continue
		}

		// Filter in place if needed
		i := 0
		for _, remindMe := range remindMeSlice.GetRemindMeSlice() {
			if remindMe == nil {
				continue
			}

			// Checks if it's time to send message/ping the user
			if t.Sub(remindMe.GetDate()) <= 0 {
				remindMeSlice.GetRemindMeSlice()[i] = remindMe
				i++
				continue
			}

			msgDM := fmt.Sprintf("RemindMe: %s", remindMe.GetMessage())
			msgChannel := fmt.Sprintf("<@%s> Remindme: %s", userID, remindMe.GetMessage())
			cmdChannel := remindMe.GetCommandChannel()

			dm, err := s.UserChannelCreate(userID)
			if err == nil {
				_, err = s.ChannelMessageSend(dm.ID, msgDM)
			}
			if err != nil && guildID != "" {
				// Checks if the user is in the server and then pings him if true
				_, err := s.GuildMember(guildID, userID)
				if err == nil {
					_, _ = s.ChannelMessageSend(cmdChannel, msgChannel)
				}
			}

			writeFlag = true
		}
		remindMeSlice.SetRemindMeSlice(remindMeSlice.GetRemindMeSlice()[:i])
	}

	if !writeFlag {
		return
	}

	err := entities.RemindMeWrite(entities.SharedInfo.GetRemindMesMap())
	if err != nil && guildID != "" {
		guildSettings := db.GetGuildSettings(guildID)
		common.LogError(s, guildSettings.BotLog, err)
		return
	}
}

// Fetches reddit feeds and returns the feeds that need to posted for all guilds
func feedHandler(s *discordgo.Session, guildIds []string) {
	redditFeedBlock.Lock()
	if redditFeedBlock.Block {
		redditFeedBlock.Unlock()
		return
	}
	redditFeedBlock.Block = true
	redditFeedBlock.Unlock()

	// Store current time
	t := time.Now()

	for _, guildID := range guildIds {
		var (
			guildFeeds      = db.GetGuildFeeds(guildID)
			guildFeedChecks = db.GetGuildFeedChecks(guildID)
			fp              = gofeed.NewParser()
			removedCheck    bool
		)
		fp.Client = &http.Client{Transport: &common.UserAgentTransport{RoundTripper: http.DefaultTransport}, Timeout: time.Second * 6}

		// Removes a check if more than its allowed lifespan hours have passed
		for _, feedCheck := range guildFeedChecks {
			dateRemoval := feedCheck.GetDate().Add(feedCheckLifespanHours)
			if t.Sub(dateRemoval) > 0 {
				continue
			}

			db.SetGuildFeedCheck(guildID, feedCheck, true)
			removedCheck = true
		}
		if removedCheck {
			guildFeedChecks = db.GetGuildFeedChecks(guildID)
		}

		for _, feed := range guildFeeds {
			var pinnedItems = make(map[*gofeed.Item]bool)

			// Parse the feed
			feedParser, err := fp.ParseURL(fmt.Sprintf("https://www.reddit.com/r/%s/%s/.rss", feed.GetSubreddit(), feed.GetPostType()))
			if err != nil {
				if _, ok := err.(*gofeed.HTTPError); ok {
					if err.(*gofeed.HTTPError).StatusCode == 429 {
						continue
					}
				}
				continue
			}

			// Iterates through each feed parser item to see if it finds something that should be posted
			for _, item := range feedParser.Items {
				var (
					skip   bool
					exists bool
				)

				for _, feedCheck := range guildFeedChecks {
					if feedCheck.GetGUID() == item.GUID &&
						feedCheck.GetFeed().GetChannelID() == feed.GetChannelID() {
						skip = true
						break
					}
				}
				if skip {
					continue
				}

				// Check if author is same and skip if not true
				if feed.GetAuthor() != "" && item.Author != nil && strings.ToLower(item.Author.Name) != fmt.Sprintf("/u/%s", feed.GetAuthor()) {
					continue
				}

				// Check if the item title starts with the set feed title
				if feed.GetTitle() != "" && !strings.HasPrefix(strings.ToLower(item.Title), feed.GetTitle()) {
					continue
				}

				// Stops the iteration if the feed doesn't exist anymore
				guildFeeds = db.GetGuildFeeds(guildID)
				for _, guildFeed := range guildFeeds {
					if guildFeed.GetSubreddit() == feed.GetSubreddit() &&
						guildFeed.GetChannelID() == feed.GetChannelID() {
						exists = true
						break
					}
				}
				guildFeeds = nil
				if !exists {
					break
				}
				exists = false

				// Checks if the item has already been posted
				feedChecks := db.GetGuildFeedChecks(guildID)
				for _, feedCheck := range feedChecks {
					if feedCheck.GetGUID() == item.GUID &&
						feedCheck.GetFeed().GetChannelID() == feed.GetChannelID() {
						exists = true
						break
					}
				}
				feedChecks = nil
				if exists {
					continue
				}
				exists = false

				// Wait some milliseconds so it doesn't hit the rate limit easily
				time.Sleep(time.Millisecond * 100)

				// Sends the feed item
				message, err := embeds.Feed(s, &feed, item)
				if err != nil {
					continue
				}

				// Adds that the feed has been posted
				t = time.Now()
				db.AddGuildFeedCheck(guildID, entities.NewFeedCheck(feed, t, item.GUID))

				// Pins/unpins the feed items if necessary
				if !feed.GetPin() {
					continue
				}
				if _, ok := pinnedItems[item]; ok {
					continue
				}

				pins, err := s.ChannelMessagesPinned(message.ChannelID)
				if err != nil {
					continue
				}

				// Unpins if necessary
				for _, pin := range pins {
					// Checks for whether the pin is one that should be unpinned
					if pin.Author.ID != s.State.User.ID {
						continue
					}
					if len(pin.Embeds) == 0 {
						continue
					}
					if pin.Embeds[0].Author == nil {
						continue
					}
					if !strings.HasPrefix(strings.ToLower(pin.Embeds[0].Author.URL), fmt.Sprintf("https://www.reddit.com/r/%s/comments/", feed.GetSubreddit())) {
						continue
					}

					_ = s.ChannelMessageUnpin(pin.ChannelID, pin.ID)
				}
				pins = nil

				// Pins
				_ = s.ChannelMessagePin(message.ChannelID, message.ID)
			}
		}
	}

	redditFeedBlock.Lock()
	redditFeedBlock.Block = false
	redditFeedBlock.Unlock()
}
