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
)

var lastCD time.Time
var start = make(chan int)
var quit = make(chan bool)

var seed = rand.NewSource(time.Now().Unix())
var rnd = rand.New(seed)

var (
	dToken    = flag.String("t", "", "discord autentication token")
	operators = flag.String("o", "", "comma separated string of operators for the bot")
	discord   *discordgo.Session
	// bot operators
	botOps map[string]struct{}
	sc     chan os.Signal
)

func getEnv() {
	*dToken = os.Getenv("DISCORDTOKEN")
	*operators = os.Getenv("DTOPS")
}

func main() {

	getEnv()
	flag.Parse() // flags override env good/bad?

	if *dToken == "" {
		fmt.Println("Usage: dist_twit -t <auth_token>")
		return
	}

	botOps = make(map[string]struct{})
	for _, c := range strings.Split(*operators, ",") {
		botOps[c] = struct{}{}
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

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc = make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

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
	case "!quit":
		if isOp(m.Author.ID) && m.Message.GuildID == "" {
			sc <- os.Kill
		}
	}
}

func showConfig(id string) {
	if isOp(id) {
		c, err := discord.UserChannelCreate(id)
		if err == nil {
			s := "operators:"
			for k := range botOps {
				s = s + " " + userIDtoMention(k)
			}
			discord.ChannelMessageSend(c.ID, s)
		}
	}
}

func isOp(id string) bool {
	if _, ok := botOps[id]; ok {
		return true
	}
	c, err := discord.UserChannelCreate(id)
	if err == nil {
		discord.ChannelMessageSend(c.ID, "You are not an operator of this bot.")
	}
	return false
}

func userIDtoMention(id string) string {
	u, err := discord.User(id)
	if err == nil {
		return u.Mention()
	}
	return id
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
