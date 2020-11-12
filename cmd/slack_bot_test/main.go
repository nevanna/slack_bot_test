package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"gopkg.in/yaml.v2"
)

var (
	config = flag.String("config", "", "path/to/config/file")
)

type ConfigData struct {
	Slack *Slack `yaml:"slack"`
}

type Slack struct {
	Token   string `yaml: "token"`
	Channel string `yaml: "channel"`
}

func main() {
	flag.Parse()
	configFileStruct := applayConfig()
	//envCheck()
	if err := flagChecker(config, "configFile"); err != nil {
		errors.Wrap(err, "error check flags")
		fmt.Printf("flag error: %v\n", err)
		return
	}
	checkConfig(configFileStruct.Slack)

	slackClient := slack.New(
		configFileStruct.Slack.Token,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)))
	rtm := slackClient.NewRTM()
	go rtm.ManageConnection()

	var botID string
	isFirstMsg := true
	for msg := range rtm.IncomingEvents {
		fmt.Println("Event Received")
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			fmt.Println("Hello event received")
		case *slack.ConnectedEvent:
			user, err := rtm.GetUserInfo(ev.Info.User.ID)
			if err != nil {
				return
			}
			botID = user.Profile.BotID
			fmt.Println("Infos: ", ev.Info)
			fmt.Println("Connection counter ", ev.ConnectionCount)
		case *slack.MessageEvent:
			//check that msg is not from bot
			if ev.BotID != botID {
				info := rtm.GetInfo()
				text := ev.Text
				text = strings.TrimSpace(text)
				fmt.Printf("time stamp of msg %v\n", ev.Timestamp)
				fmt.Printf("initial msg ThreadTimestamp %v\n", ev.ThreadTimestamp)
				if isFirstMsg {
					firstMSG(ev, info, rtm, configFileStruct.Slack)
					isFirstMsg = false
				}
				replayInChat(ev, &text, info, rtm)
				replayInThread(ev, &text, info, rtm)
			} else {
				fmt.Println("bot's message")
			}

		case *slack.DesktopNotificationEvent:
			fmt.Printf("Message: %v\n", ev)
		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())
		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			return
		default:
			fmt.Printf("unexpected event: %v\n", msg.Data)
		}
	}

}

func flagChecker(fl *string, flName string) error {
	if len(*fl) == 0 || *fl == "" {
		return fmt.Errorf("%s is empty or invalid [%s]", flName, *fl)
	}
	return nil
}

var answers = [...]string{"Hello", "Good morning", "Good evening", "Hi", "Greetings", "I miss you"}

func randomAnswer() *string {
	idx := rand.Intn(len(answers))
	return &answers[idx]
}

func replayInChat(ev *slack.MessageEvent, text *string, info *slack.Info, rtm *slack.RTM) {
	containedLink := strings.Contains(*text, "link")
	reactToMsg(ev, rtm)
	if ev.User != info.User.ID && containedLink {
		rtm.SendMessage(rtm.NewOutgoingMessage("`Link` is detected in message :100:", ev.Channel))

	}
}

func firstMSG(ev *slack.MessageEvent, info *slack.Info, rtm *slack.RTM, s *Slack) {
	if ev.User != info.User.ID {
		message := fmt.Sprintf("SLACK_CHANNEL ---> %s\n", s.Channel)
		rtm.SendMessage(rtm.NewOutgoingMessage(message, ev.Channel))

	}
}

func reactToMsg(ev *slack.MessageEvent, rtm *slack.RTM) {
	ts := ev.Timestamp
	if ev.ThreadTimestamp != "" {
		ts = ev.ThreadTimestamp
	}
	reactionTarget := slack.ItemRef{
		Channel:   ev.Channel,
		Timestamp: ts,
	}
	if err := rtm.AddReaction("100", reactionTarget); err != nil {
		fmt.Printf("error add reaction %v\n", err)
	}
}

func replayInThread(ev *slack.MessageEvent, text *string, info *slack.Info, rtm *slack.RTM) {
	containedHi := strings.Contains(*text, "hi")
	if ev.User != info.User.ID && containedHi {
		ts := ev.Timestamp
		if ev.ThreadTimestamp != "" {
			ts = ev.ThreadTimestamp
		}
		answer := rtm.NewOutgoingMessage(*randomAnswer(), ev.Channel)
		answer.ThreadTimestamp = ts
		fmt.Printf("ThreadTimestamp %v\n", answer.ThreadTimestamp)
		rtm.SendMessage(answer)
	}
}

func envCheck() {
	slackChannel := os.Getenv("SLACK_CHANNEL")
	fmt.Printf("SLACK_CHANNEL ---> %s\n", slackChannel)
}

func checkConfig(s *Slack) {
	fmt.Printf("token is -> \"%s\" \n", s.Token)
	fmt.Printf("channel is -> \"%s\" \n", s.Channel)
}

func applayConfig() *ConfigData {
	if *config == "" {
		log.Fatalf("error: no config file")
	}
	content, err := ioutil.ReadFile(*config)
	if err != nil {
		fmt.Println("error reading")
		log.Fatalf("error: %v", err)
	}
	fmt.Println(content)
	config := ConfigData{}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		fmt.Println("error reading")
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- t:\n%v\n\n", config)
	return &config
}
