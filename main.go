package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

var lastCD time.Time
var start = make(chan int)
var quit = make(chan bool)

var seed = rand.NewSource(time.Now().Unix())
var rnd = rand.New(seed)

var (
	dToken       = flag.String("t", "", "discord autentication token")
	rToken       = flag.String("r", "", "rapidapi autentication token")
	cronSpec     = flag.String("c", "0 1 * * *", "cron spec for periodic actions")
	initialChans = flag.String("i", "", "comma separated string ofinitial channels to report to")
	passwd       = flag.String("p", "", "password for the bot")
	operators    = flag.String("o", "", "comma separated string of operators for the bot")
	reportCron   *cron.Cron
	reportCronID cron.EntryID
	discord      *discordgo.Session
	mCreate      *discordgo.MessageCreate
	covChans     map[string]struct{}
	covOps       map[string]struct{}
	sc           chan os.Signal
)

func main() {

	flag.Parse()

	if *dToken == "" {
		fmt.Println("Usage: dist_twit -t <auth_token>")
		return
	}

	var err error
	discord, err = discordgo.New("Bot " + *dToken)

	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	discord.AddHandler(messageCreate)
	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	covChans = make(map[string]struct{})
	for _, c := range strings.Split(*initialChans, ",") {
		covChans[c] = struct{}{}

	}
	covOps = make(map[string]struct{})
	for _, c := range strings.Split(*operators, ",") {
		covOps[c] = struct{}{}

	}

	reportCron = cron.New()
	reportCronID, err = reportCron.AddFunc(*cronSpec, cronReport)
	if err == nil {
		reportCron.Start()
	} else {
		fmt.Println(err)
	}

	fmt.Printf("Cronspec is %s\n", *cronSpec)
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc = make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Stop cron jobs.
	reportCron.Stop()
	// Cleanly close down the Discord session.
	discord.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	msg := strings.Split(m.Content, " ")
	switch msg[0] {
	case "!cd": // coop countdown function
		if len(msg) > 1 {
			var count int
			_, err := fmt.Sscanf(msg[1], "%v", &count)
			//count, err := strconv.Atoi(m[1])
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, idiot[rnd.Intn(len(idiot))])
			} else if count == 0 {
				s.ChannelMessageSend(m.ChannelID, "You in a hurry?")
			} else {
				if count < -5 || count > 5 {
					s.ChannelMessageSend(m.ChannelID, "The count must be a number from 1 to 5 (defaults to 5).")
				} else {
					printer(m.ChannelID, count)
				}
			}
		} else {
			printer(m.ChannelID, 5)
		}
	case "!q": // get a metaphorsum
		meta, err := getMetaphorsum()
		if err == nil {
			s.ChannelMessageSend(m.ChannelID, meta)
		}
	case "!cov": // report covid-19 stats
		if *rToken == "" {
			return
		}
		if time.Now().Sub(lastCD).Seconds() < 10 {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Please wait %.0f seconds and try again.", 10.0-time.Now().Sub(lastCD).Seconds()))
			return
		}
		var err error
		var report string
		lastCD = time.Now()
		if len(msg) > 1 {
			report, err = covid(strings.Join(msg[1:], "-"))
		} else {
			report, err = covid("usa")
		}
		if err == nil {
			s.ChannelMessageSend(m.ChannelID, report)
		}
	case "!reaper": // periodic USA death toll reports
		if len(msg) < 2 || msg[1] != "off" {
			if !isOp(m.Author.ID) {
				return
			}
			if len(msg) == 1 {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Grim Reaper reports are *on* for %s.", chanIDtoMention(m.ChannelID)))
				covChans[m.ChannelID] = struct{}{}
			} else if id, err := chanLinkToID(msg[1]); err == nil {
				covChans[id] = struct{}{}
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Grim Reaper reports are *on* for %s.", chanIDtoMention(id)))
			}
		} else if len(msg) == 2 {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Grim Reaper reports are *off* for %s.", chanIDtoMention(m.ChannelID)))
			delete(covChans, m.ChannelID)
		} else if id, err := chanLinkToID(msg[2]); err == nil {
			delete(covChans, id)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Grim Reaper reports are *off* for %s.", chanIDtoMention(id)))
		}
	case "!chans":
		if !isOp(m.Author.ID) {
			return
		}
		fmt.Printf("%+v", m.Message.Author.ID)
		for k := range covChans {
			fmt.Println(k)
			s.ChannelMessageSend(m.ChannelID, chanIDtoMention(k))
		}
	case "!op":
		if len(msg) > 1 && msg[1] == *passwd {
			if len(msg) > 2 {
				u, err := s.User(msg[2])
				if err == nil {
					covOps[u.ID] = struct{}{}
				} else {
					s.ChannelMessageSend(m.ChannelID, "Invalid user ID.")
				}
			} else {
				covOps[m.Message.Author.ID] = struct{}{}
			}
		}
	case "!deop":
		if !isOp(m.Author.ID) {
			return
		}
		if len(msg) > 1 {
			if _, ok := covOps[msg[2]]; ok {
				delete(covOps, msg[2])
				u, err := s.User(msg[2])
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("User ID %s removed from operator list.", msg[2]))
				} else {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("User %s removed from operator list.", u.Mention()))
				}
			} else {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("User ID %s is not in the operator list.", msg[2]))
			}
		}
	case "!delmsg":
		if len(msg) > 2 {
			s.ChannelMessageDelete(msg[1], msg[2])
		}
	case "!quit":
		if isOp(m.Author.ID) {
			sc <- os.Kill
		}
	}
}

func isOp(id string) bool {
	if _, ok := covOps[id]; ok || *passwd == "" {
		return true
	}
	c, err := discord.UserChannelCreate(id)
	if err == nil {
		discord.ChannelMessageSend(c.ID, "You are not an operator of this bot.")
	}
	return false
}

// Converts a channel ID to a mention. On error it returns the channel ID string.
func chanIDtoMention(id string) string {
	channel, err := discord.State.Channel(id)
	if err == nil {
		return channel.Mention()
	}
	return "channel: " + id
}

// Converts a channel link to an ID. If passed a valid ID it is returned it unchanged.
func chanLinkToID(link string) (id string, err error) {
	id = strings.Replace(strings.Replace(strings.Replace(link, "<", "", 1), ">", "", 1), "#", "", 1)
	_, err = discord.Channel(id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// "A wise– if perhaps slightly pedantic– generator of metaphor."
func getMetaphorsum() (string, error) {
	resp, err := http.Get("http://metaphorpsum.com/paragraphs/1/1")
	if err != nil {
		// handle error
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		if err == nil {
			return bodyString, nil
		}
	}
	return "", err
}

// countdown printer
func printer(ChannelID string, n int) {
	if throttle(lastCD) {
		lastCD = time.Now()
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			if n == 0 {
				break
			}
			discord.ChannelMessageSend(ChannelID, strconv.Itoa(n))
			if n < 0 {
				n++
			} else {
				n--
			}
		}
		ticker.Stop()
		discord.ChannelMessageSend(ChannelID, "_Go!_")
	} else {
		discord.ChannelMessageSend(ChannelID, "_No!_")
	}
}

// throttles responses
func throttle(lastTime time.Time) bool {
	//t1 := time.Date(2006, 1, 1, 12, 23, 10, 0, time.UTC)
	if time.Now().Sub(lastTime).Seconds() < 20 {
		return false
	}
	return true
}
