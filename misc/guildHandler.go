package misc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/r-anime/ZeroTsu/config"
)

var (
	GuildMap = make(map[string]*guildInfo)
	dbPath = "database/guilds"
	fileNames = [...]string{"bannedUsers.json", "filters.json", "messReqs.json", "spoilerRoles.json", "rssThreads.json",
		"rssThreadCheck.json", "raffles.json", "waifus.json", "waifuTrades.json", "memberInfo.json", "emojiStats.json",
		"channelStats.json", "userChangeStats.json", "verifiedStats.json", "remindMes.json", "voteInfo.json", "tempCha.json",
		"reactJoin.json", "guildSettings.json"}
)

type guildInfo struct {
	GuildID 			string
	GuildConfig			GuildSettings

	BannedUsers			[]BannedUsers
	Filters 			[]Filter
	MessageRequirements []MessRequirement
	SpoilerRoles		[]discordgo.Role
	RssThreads			[]RssThread
	RssThreadChecks		[]RssThreadCheck
	Raffles 			[]Raffle
	Waifus				[]Waifu
	WaifuTrades			[]WaifuTrade

	MemberInfoMap		map[string]*UserInfo
	SpoilerMap			map[string]*discordgo.Role
	EmojiStats 			map[string]*Emoji
	ChannelStats 		map[string]*Channel
	UserChangeStats 	map[string]int
	VerifiedStats 		map[string]int
	RemindMes			map[string]*RemindMeSlice
	VoteInfoMap			map[string]*VoteInfo
	TempChaMap 			map[string]*TempChaInfo
	ReactJoinMap		map[string]*ReactJoin
	EmojiRoleMap		map[string][]string
}

// Guild settings for misc things
type GuildSettings struct {
	Prefix              string     `json:"Prefix"`
	BotLog              Cha        `json:"BotLogID"`
	CommandRoles        []Role     `json:"CommandRoles"`
	OptInUnder          OptinRole  `json:"OptInUnder"`
	OptInAbove          OptinRole  `json:"OptInAbove"`
	VoiceChas           []VoiceCha `json:"VoiceChas"`
	VoteModule          bool       `json:"VoteModule"`
	VoteChannelCategory Cha        `json:"VoteChannelCategory"`
	WaifuModule         bool       `json:"WaifuModule"`
	ReactsModule        bool       `json:"ReactsModule"`
	FileFilter          bool       `json:"FileFilter"`
	DailyStats			bool	   `json:"DailyStats"`
}

type Role struct {
	Name	string	`json:"Name"`
	ID		string	`json:"ID"`
}

type VoiceCha struct {
	Name	string	`json:"Name"`
	ID		string	`json:"ID"`
	Roles	[]Role	`json:"Roles"`
}

type Cha struct {
	Name	string	`json:"Name"`
	ID		string	`json:"ID"`
}

type OptinRole struct {
	Name		string	`json:"Name"`
	ID			string	`json:"ID"`
	Position	int		`json:"Position"`
}

// VoteInfo is the in memory storage of each vote channel's info
type VoteInfo struct {
	Date         time.Time          `json:"Date"`
	Channel      string             `json:"Channel"`
	ChannelType  string             `json:"ChannelType"`
	Category     string             `json:"Category,omitempty"`
	Description  string             `json:"Description,omitempty"`
	VotesReq     int                `json:"VotesReq"`
	MessageReact *discordgo.Message `json:"MessageReact"`
	User		 *discordgo.User	`json:"User"`
}

type TempChaInfo struct {
	CreationDate	time.Time		`json:"CreationDate"`
	RoleName		string			`json:"RoleName"`
	Elevated		bool			`json:"Elevated"`
}

type ReactJoin struct {
	RoleEmojiMap []map[string][]string `json:"roleEmoji"`
}

type Filter struct {
	Filter 	string	`json:"Filter"`
}

type MessRequirement struct {
	Phrase 		string	`json:"Phrase"`
	Type 		string	`json:"Type"`
	Channel		string	`json:"Channel"`
	LastUserID	string
}

type RssThread struct {
	Thread  string `json:"Thread"`
	Channel string `json:"Channel"`
	Author  string `json:"Author"`
}

type RssThreadCheck struct {
	Thread string    `json:"Thread"`
	Date   time.Time `json:"Date"`
	ChannelID string `json:"ChannelID"`
}

type Emoji struct {
	ID          	   string `json:"id"`
	Name               string `json:"name"`
	MessageUsage       int    `json:"messageUsage"`
	UniqueMessageUsage int    `json:"uniqueMessages"`
	Reactions          int    `json:"reactions"`
}

type Channel struct {
	ChannelID 	  string
	Name 		  string
	Messages  	  map[string]int
	RoleCount 	  map[string]int `json:",omitempty"`
	Optin     	  bool
	Exists    	  bool
}

type RemindMeSlice struct {
	RemindMeSlice []RemindMe
}

type RemindMe struct {
	Message			string
	Date			time.Time
	CommandChannel	string
	RemindID		int
}

type Raffle struct {
	Name			string		`json:"Name"`
	ParticipantIDs	[]string	`json:"ParticipantIDs"`
	ReactMessageID	string		`json:"ReactMessageID"`
}

type Waifu struct {
	Name			string		`json:"Name"`
}

type WaifuTrade struct {
	TradeID			string		`json:"TradeID"`
	InitiatorID		string		`json:"InitiatorID"`
	AccepteeID		string		`json:"AccepteeID"`
}

// Loads all guilds in the database/guilds folder
func LoadGuilds() {

	// Creates missing "database" and "guilds" folder if they don't exist
	if _, err := os.Stat("database"); os.IsNotExist(err) {
		os.Mkdir("database", 0777)
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		os.Mkdir(dbPath, 0777)
		return
	}

	folders, err := ioutil.ReadDir(dbPath)
	if err != nil {
		log.Panicln(err)
	}

	for _, f := range folders {
		if !f.IsDir() {
			continue
		}
		folderName := f.Name()
		files, err := IOReadDir(fmt.Sprintf("database/guilds/%s", folderName))
		if err != nil {
			log.Panicln(err)
		}
		MapMutex.Lock()

		GuildMap[folderName] = &guildInfo{
			GuildID:             folderName,
			GuildConfig:		 GuildSettings{Prefix: ".", VoteModule: false, WaifuModule: false, ReactsModule: true, FileFilter: false},
			BannedUsers:         nil,
			Filters:             nil,
			MessageRequirements: nil,
			SpoilerRoles:        nil,
			RssThreads:          nil,
			RssThreadChecks:     nil,
			Raffles:             nil,
			Waifus:              nil,
			WaifuTrades:         nil,
			MemberInfoMap:       make(map[string]*UserInfo),
			SpoilerMap:          make(map[string]*discordgo.Role),
			EmojiStats:          make(map[string]*Emoji),
			ChannelStats:        make(map[string]*Channel),
			UserChangeStats:     make(map[string]int),
			VerifiedStats:       make(map[string]int),
			RemindMes:           make(map[string]*RemindMeSlice),
			VoteInfoMap:		 make(map[string]*VoteInfo),
			TempChaMap:			 make(map[string]*TempChaInfo),
			ReactJoinMap:   	 make(map[string]*ReactJoin),
			EmojiRoleMap: 		 make(map[string][]string),
		}
		for _, file := range files {
			LoadGuildFile(folderName, file)
		}
		MapMutex.Unlock()
	}
}

func LoadGuildFile(guildID string, file string) {
	// Reads all the info from the file and puts them in infoByte as bytes
	infoByte, err := ioutil.ReadFile(fmt.Sprintf("%s/%s/%s", dbPath, guildID, file))
	if err != nil {
		log.Println(err)
		return
	}

	// Takes the data and puts it into the appropriate field
	switch file {
	case "bannedUsers.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].BannedUsers)
	case "filters.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].Filters)
	case "messReqs.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].MessageRequirements)
	case "spoilerRoles.json":
		err = json.Unmarshal(infoByte, &GuildMap[guildID].SpoilerRoles)
		if err != nil {
			return
		}
		// Fills spoilerMap with roles from the spoilerRoles.json file if latter is not empty
		for i := 0; i < len(GuildMap[guildID].SpoilerRoles); i++ {
			GuildMap[guildID].SpoilerMap[GuildMap[guildID].SpoilerRoles[i].ID] = &GuildMap[guildID].SpoilerRoles[i]
		}
	case "rssThreads.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].RssThreads)
	case "rssThreadCheck.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].RssThreadChecks)
	case "raffles.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].Raffles)
	case "waifus.json":
		if GuildMap[guildID].GuildConfig.WaifuModule {
			_ = json.Unmarshal(infoByte, &GuildMap[guildID].Waifus)
		}
	case "waifuTrades.json":
		if GuildMap[guildID].GuildConfig.WaifuModule {
			_ = json.Unmarshal(infoByte, &GuildMap[guildID].WaifuTrades)
		}
	case "memberInfo.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].MemberInfoMap)
	case "emojiStats.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].EmojiStats)
	case "channelStats.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].ChannelStats)
	case "userChangeStats.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].UserChangeStats)
	case "verifiedStats.json":
		if config.Website != "" {
			_ = json.Unmarshal(infoByte, &GuildMap[guildID].VerifiedStats)
		}
	case "remindMes.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].RemindMes)
	case "voteInfo.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].VoteInfoMap)
	case "tempCha.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].TempChaMap)
	case "reactJoin.json":
		if GuildMap[guildID].GuildConfig.ReactsModule {
			_ = json.Unmarshal(infoByte, &GuildMap[guildID].ReactJoinMap)
		}
	case "guildSettings.json":
		_ = json.Unmarshal(infoByte, &GuildMap[guildID].GuildConfig)
	}
}

// Writes to memberInfo.json
func WriteMemberInfo(info map[string]*UserInfo, guildID string) {

	// Turns info slice into byte ready to be pushed to file
	MarshaledStruct, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/memberInfo.json", guildID), MarshaledStruct, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// Writes emoji stats to emojiStats.json
func EmojiStatsWrite(emojiStats map[string]*Emoji, guildID string) (bool, error) {

	// Turns that map into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(emojiStats, "", "    ")
	if err != nil {
		return false, err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/emojiStats.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return false, err
	}

	return false, err
}

// Writes channel stats to channelStats.json
func ChannelStatsWrite(channelStats map[string]*Channel, guildID string) (bool, error) {

	// Turns that map into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(channelStats, "", "    ")
	if err != nil {
		return false, err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/channelStats.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return false, err
	}

	return false, err
}

// Writes User Change stats to userChangeStats.json
func UserChangeStatsWrite(userStats map[string]int, guildID string) (bool, error) {

	// Turns that map into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(userStats, "", "    ")
	if err != nil {
		return false, err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/userChangeStats.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return false, err
	}

	return false, err
}

// Writes Verified stats to verifiedStats.json
func VerifiedStatsWrite(verifiedStats map[string]int, guildID string) error {

	// Turns that map into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(verifiedStats, "", "    ")
	if err != nil {
		return err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/verifiedStats.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}
// Writes RemindMe notes to remindMes.json
func RemindMeWrite(remindMe map[string]*RemindMeSlice, guildID string) (bool, error) {

	// Turns that slice into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(remindMe, "", "    ")
	if err != nil {
		return false, err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/remindMes.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return false, err
	}

	return false, err
}

// Writes vote info to voteInfo.json
func VoteInfoWrite(info map[string]*VoteInfo, guildID string) {

	// Turns info slice into byte ready to be pushed to file
	MarshaledStruct, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		return
	}

	//Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/voteInfo.json", guildID), MarshaledStruct, 0644)
	if err != nil {
		return
	}
}

// Writes temp cha info to tempCha.json
func TempChaWrite(info map[string]*TempChaInfo, guildID string) {

	// Turns info map into byte ready to be pushed to file
	MarshaledStruct, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		return
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/tempCha.json", guildID), MarshaledStruct, 0644)
	if err != nil {
		return
	}
}

// Writes react channel join info to ReactJoin.json
func ReactJoinWrite(info map[string]*ReactJoin, guildID string) {

	// Turns info slice into byte ready to be pushed to file
	marshaledStruct, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		return
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/reactJoin.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return
	}
}

// Writes Raffles to raffles.json
func RafflesWrite(raffle []Raffle, guildID string) error {

	// Turns that slice into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(raffle, "", "    ")
	if err != nil {
		return err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/raffles.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Writes Waifus to waifus.json
func WaifusWrite(waifu []Waifu, guildID string) error {

	// Turns that slice into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(waifu, "", "    ")
	if err != nil {
		return err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/waifus.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Writes WaifuTrades to waifutrades.json
func WaifuTradesWrite(trade []WaifuTrade, guildID string) error {

	// Turns that slice into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(trade, "", "    ")
	if err != nil {
		return err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/waifuTrades.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Writes to bannedUsers.json from bannedUsersSlice
func BannedUsersWrite(bannedUsers []BannedUsers, guildID string) {
	// Turns that slice into bytes to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(bannedUsers, "", "    ")
	if err != nil {
		return
	}
	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/bannedUsers.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return
	}
	return
}

// Removes raffle with name string "raffle" from raffles.json
func RaffleRemove(raffle string, guildID string) error {

	var raffleExists = false

	raffle = strings.ToLower(raffle)

	// Checks if that raffle already exists in the raffles slice
	MapMutex.Lock()
	for i, sliceRaffle := range GuildMap[guildID].Raffles {
		if strings.ToLower(sliceRaffle.Name) == raffle {
			raffleExists = true
			GuildMap[guildID].Raffles = append(GuildMap[guildID].Raffles[:i], GuildMap[guildID].Raffles[i+1:]...)
			break
		}
	}
	if !raffleExists {
		MapMutex.Unlock()
		return fmt.Errorf("Error: No such raffle exists")
	}

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.Marshal(GuildMap[guildID].Raffles)
	if err != nil {
		MapMutex.Unlock()
		return err
	}
	MapMutex.Unlock()

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/raffles.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Adds string "phrase" to filters.json and memory
func FiltersWrite(phrase string, guildID string) error {

	var filterStruct = 	Filter{phrase}

	// Appends the new filtered phrase to a slice of all of the old ones if it doesn't exist
	MapMutex.Lock()
	for _, filter := range GuildMap[guildID].Filters {
		if filter.Filter == phrase {
			MapMutex.Unlock()
			return fmt.Errorf(fmt.Sprintf("Error: `%v` is already on the filter list.", phrase))
		}
	}

	// Adds the phrase to the filter list
	GuildMap[guildID].Filters = append(GuildMap[guildID].Filters, filterStruct)

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(GuildMap[guildID].Filters, "", "    ")
	if err != nil {
		MapMutex.Unlock()
		return err
	}
	MapMutex.Unlock()

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/filters.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Removes string "phrase" from filters.json and memory
func FiltersRemove(phrase string, guildID string) error {

	var filterExists	bool

	// Deletes the filtered phrase if it finds it exists
	MapMutex.Lock()
	for i, filter := range GuildMap[guildID].Filters {
		if filter.Filter == phrase {
			GuildMap[guildID].Filters = append(GuildMap[guildID].Filters[:i], GuildMap[guildID].Filters[i+1:]...)
			filterExists = true
			break
		}
	}

	// Exits func if the filter is not on the list
	if !filterExists {
		MapMutex.Unlock()
		return fmt.Errorf(fmt.Sprintf("Error: `%v` is not in the filter list.", phrase))
	}

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.Marshal(GuildMap[guildID].Filters)
	if err != nil {
		MapMutex.Unlock()
		return err
	}
	MapMutex.Unlock()

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/filters.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Adds string "phrase" to messReqs.json and memory
func MessRequirementWrite(phrase string, channel string, filterType string, guildID string) error {

	var MessRequirementStruct = MessRequirement{phrase,filterType, channel, ""}

	// Appends the new phrase to a slice of all of the old ones if it doesn't exist
	MapMutex.Lock()
	for _, requirement := range GuildMap[guildID].MessageRequirements {
		if requirement.Phrase == phrase {
			MapMutex.Unlock()
			return fmt.Errorf(fmt.Sprintf("Error: `%v` is already on the message requirement list.", phrase))
		}
	}

	// Adds the phrase to the message requirement list
	GuildMap[guildID].MessageRequirements = append(GuildMap[guildID].MessageRequirements, MessRequirementStruct)

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(GuildMap[guildID].MessageRequirements, "", "    ")
	if err != nil {
		MapMutex.Unlock()
		return err
	}
	MapMutex.Unlock()

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/messReqs.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Removes string "phrase" from messReqs.json and memory
func MessRequirementRemove(phrase string, channelID string, guildID string) error {

	var phraseExists	bool

	// Deletes the filtered phrase if it finds it exists
	MapMutex.Lock()
	for i, requirement:= range GuildMap[guildID].MessageRequirements {
		if requirement.Phrase == phrase {
			if channelID != "" {
				if requirement.Channel != channelID {
					continue
				}
			}
			GuildMap[guildID].MessageRequirements = append(GuildMap[guildID].MessageRequirements[:i], GuildMap[guildID].MessageRequirements[i+1:]...)
			phraseExists = true
			break
		}
	}

	// Exits func if the filter is not on the list
	if !phraseExists {
		MapMutex.Unlock()
		return fmt.Errorf(fmt.Sprintf("Error: `%v` is not in the message requirement list.", phrase))
	}

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.Marshal(GuildMap[guildID].MessageRequirements)
	if err != nil {
		MapMutex.Unlock()
		return err
	}
	MapMutex.Unlock()

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/messReqs.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Writes spoilerRoles map to spoilerRoles.json
func SpoilerRolesWrite(SpoilerMapWrite map[string]*discordgo.Role, guildID string) {

	var roleExists  bool

	// Appends the new spoiler role to a slice of all of the old ones if it doesn't exist
	if len(GuildMap[guildID].SpoilerRoles) == 0 {
		for k := range SpoilerMapWrite {
			GuildMap[guildID].SpoilerRoles = append(GuildMap[guildID].SpoilerRoles, *SpoilerMapWrite[k])
		}
	} else {
		for k := range SpoilerMapWrite {
			for i := 0; i < len(GuildMap[guildID].SpoilerRoles); i++ {
				if GuildMap[guildID].SpoilerRoles[i].ID == SpoilerMapWrite[k].ID {
					roleExists = true
					break

				} else {
					roleExists = false
				}
			}

			if !roleExists {
				GuildMap[guildID].SpoilerRoles = append(GuildMap[guildID].SpoilerRoles, *SpoilerMapWrite[k])
			}
		}
	}

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(GuildMap[guildID].SpoilerRoles, "", "    ")
	if err != nil {
		return
	}

	// Writes to file
	_ = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/spoilerRoles.json", guildID), marshaledStruct, 0644)
}

// Deletes a role from spoilerRoles map to spoilerRoles.json
func SpoilerRolesDelete(roleID string, guildID string) {

	if len(GuildMap[guildID].SpoilerRoles) == 0 {
		return
	}
	for i := 0; i < len(GuildMap[guildID].SpoilerRoles); i++ {
		if GuildMap[guildID].SpoilerRoles[i].ID == roleID {
			GuildMap[guildID].SpoilerRoles = append(GuildMap[guildID].SpoilerRoles[:i], GuildMap[guildID].SpoilerRoles[i+1:]...)
		}
	}

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(GuildMap[guildID].SpoilerRoles, "", "    ")
	if err != nil {
		return
	}

	// Writes to file
	_ = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/spoilerRoles.json", guildID), marshaledStruct, 0644)
}

// Writes string "thread" to rssThreadsCheck.json
func RssThreadsWrite(thread string, channel string, author string, guildID string) (bool, error) {

	thread = strings.ToLower(thread)

	var (
		threadStruct = 	RssThread{thread, channel, author}
		err				error
	)

	// Appends the new thread to a slice of all of the old ones if it doesn't exist
	for i := 0; i < len(GuildMap[guildID].RssThreads); i++ {
		if GuildMap[guildID].RssThreads[i].Thread == threadStruct.Thread && GuildMap[guildID].RssThreads[i].Channel == threadStruct.Channel {
			return true, err
		}
	}

	GuildMap[guildID].RssThreads = append(GuildMap[guildID].RssThreads, threadStruct)

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(GuildMap[guildID].RssThreads, "", "    ")
	if err != nil {
		return false, err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/rssThreads.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return false, err
	}

	return false, err
}

// Removes string "thread" from rssThreads.json
func RssThreadsRemove(thread string, author string, guildID string) (bool, error) {

	thread = strings.ToLower(thread)

	var (
		threadExists = false
		err          error
	)

	// Deletes the thread if it finds it exists
	for i, readThread := range GuildMap[guildID].RssThreads {
		if readThread.Thread == thread {
			threadExists = true
			if author == "" {
				GuildMap[guildID].RssThreads = GuildMap[guildID].RssThreads[:i+copy(GuildMap[guildID].RssThreads[i:], GuildMap[guildID].RssThreads[i+1:])]
				break
			} else {
				if readThread.Author == author {
					GuildMap[guildID].RssThreads = GuildMap[guildID].RssThreads[:i+copy(GuildMap[guildID].RssThreads[i:], GuildMap[guildID].RssThreads[i+1:])]
					break
				} else {
					threadExists = false
				}
			}
		}
	}

	if !threadExists {
		return false, err
	}

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.Marshal(GuildMap[guildID].RssThreads)
	if err != nil {
		return true, err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/rssThreads.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return true, err
	}

	return true, err
}

// Writes string "thread" to rssThreadCheck.json. Returns bool depending on success or not
func RssThreadsTimerWrite(thread string, date time.Time, channelID string, guildID string) bool {

	thread = strings.ToLower(thread)

	var threadCheckStruct= RssThreadCheck{thread, date, channelID}

	// Appends the new thread to a slice of all of the old ones if it doesn't exist
	for p := 0; p < len(GuildMap[guildID].RssThreadChecks); p++ {
		if GuildMap[guildID].RssThreadChecks[p].Thread == threadCheckStruct.Thread &&
			GuildMap[guildID].RssThreadChecks[p].ChannelID == threadCheckStruct.ChannelID {
			return false
		}
	}

	GuildMap[guildID].RssThreadChecks = append(GuildMap[guildID].RssThreadChecks, threadCheckStruct)

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.MarshalIndent(GuildMap[guildID].RssThreadChecks, "", "    ")
	if err != nil {
		return false
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/rssThreadCheck.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return false
	}

	return true
}

// Removes string "thread" to rssThreadCheck.json
func RssThreadsTimerRemove(thread string, date time.Time, channelID string, guildID string) error {

	thread = strings.ToLower(thread)

	var (
		threadExists= false
		threadCheckStruct= RssThreadCheck{thread, date, channelID}
	)

	// Deletes the thread if it finds it exists
	for i := 0; i < len(GuildMap[guildID].RssThreadChecks); i++ {
		if strings.ToLower(GuildMap[guildID].RssThreadChecks[i].Thread) == threadCheckStruct.Thread &&
			GuildMap[guildID].RssThreadChecks[i].ChannelID == threadCheckStruct.ChannelID {
			threadExists = true
			GuildMap[guildID].RssThreadChecks = append(GuildMap[guildID].RssThreadChecks[:i], GuildMap[guildID].RssThreadChecks[i+1:]...)
			break
		}
	}
	if !threadExists {
		return fmt.Errorf("Thread doesn't exist")
	}

	// Turns that struct slice into bytes again to be ready to written to file
	marshaledStruct, err := json.Marshal(GuildMap[guildID].RssThreadChecks)
	if err != nil {
		return err
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/rssThreadCheck.json", guildID), marshaledStruct, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Writes guild settings to guildSettings.json
func GuildSettingsWrite(info GuildSettings, guildID string) {

	// Turns info map into byte ready to be pushed to file
	MarshaledStruct, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		return
	}

	// Writes to file
	err = ioutil.WriteFile(fmt.Sprintf(dbPath + "/%v/guildSettings.json", guildID), MarshaledStruct, 0644)
	if err != nil {
		return
	}
}

// Reads and returns the names of every file in that directory
func IOReadDir(root string) ([]string, error) {
	var files []string
	fileInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}
	return files, nil
}

// Initializes a guild's DB files
func initDB(guildID string) {
	path := fmt.Sprintf("%v/%v", dbPath, guildID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}
	for _, name := range fileNames {
		file, err := os.OpenFile(fmt.Sprintf("%v/%v/%v",dbPath, guildID , name), os.O_RDONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Println(err)
			continue
		}
		err = file.Close()
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

// Writes/Refreshes all DBs
func writeAll(guildID string) {
	LoadGuilds()
	MapMutex.Lock()
	WriteMemberInfo(GuildMap[guildID].MemberInfoMap, guildID)
	_, _ = EmojiStatsWrite(GuildMap[guildID].EmojiStats, guildID)
	_, _ = ChannelStatsWrite(GuildMap[guildID].ChannelStats, guildID)
	_, _ = UserChangeStatsWrite(GuildMap[guildID].UserChangeStats, guildID)
	_ = VerifiedStatsWrite(GuildMap[guildID].VerifiedStats, guildID)
	_, _ = RemindMeWrite(GuildMap[guildID].RemindMes, guildID)
	VoteInfoWrite(GuildMap[guildID].VoteInfoMap, guildID)
	TempChaWrite(GuildMap[guildID].TempChaMap, guildID)
	ReactJoinWrite(GuildMap[guildID].ReactJoinMap, guildID)
	_ = RafflesWrite(GuildMap[guildID].Raffles, guildID)
	_ = WaifusWrite(GuildMap[guildID].Waifus, guildID)
	_ = WaifuTradesWrite(GuildMap[guildID].WaifuTrades, guildID)
	BannedUsersWrite(GuildMap[guildID].BannedUsers, guildID)
	GuildSettingsWrite(GuildMap[guildID].GuildConfig, guildID)
	MapMutex.Unlock()
}