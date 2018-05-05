package main

import (
	"fmt"
	"os"
	"strings"

	"log"

	_ "github.com/joho/godotenv/autoload"
	"github.com/nlopes/slack"
)

type User struct {
	Info   slack.User
	Rating int
}

type Token struct {
	Token string `json:"token"`
}

type Message struct {
	ChannelId string
	Timestamp string
	Payload   string
	Rating    int
	User      User
}

type BotCentral struct {
	Channel *slack.Channel
	Event   *slack.MessageEvent
	UserId  string
}

type AttachmentChannel struct {
	Channel      *slack.Channel
	Attachment   *slack.Attachment
	DisplayTitle string
}

type Messages []Message

func (u Messages) Len() int {
	return len(u)
}
func (u Messages) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}
func (u Messages) Less(i, j int) bool {
	return u[i].Rating > u[j].Rating
}

var (
	api               *slack.Client
	botKey            Token
	botId             string
	botCommandChannel chan *BotCentral
	botReplyChannel   chan AttachmentChannel
)

func handleBotCommands(c chan AttachmentChannel) {
	commands := map[string]string{
		"whoami": "The smell of a vampire, the touch of butterfly...",
		"help":   "See the available bot commands.",
	}

	var attachmentChannel AttachmentChannel

	for {
		botChannel := <-botCommandChannel
		attachmentChannel.Channel = botChannel.Channel
		commandArray := strings.Fields(botChannel.Event.Text)
		log.Println("[DEBUG] received command", commandArray)
		switch commandArray[1] {
		case "help":
			attachmentChannel.DisplayTitle = "Tömat Vörben"
			fields := make([]slack.AttachmentField, 0)
			for k, v := range commands {
				fields = append(fields, slack.AttachmentField{
					Title: "<bot> " + k,
					Value: v,
				})
			}
			attachment := &slack.Attachment{
				Pretext: "Bot Command List",
				Color:   "#B733FF",
				Fields:  fields,
			}
			attachmentChannel.Attachment = attachment
			c <- attachmentChannel

		case "whoami":
			fmt.Println("[INFO] whoami")

			fields := []slack.AttachmentField{
				slack.AttachmentField{
					Title: "I, Tömat Vörben",
					Value: "https://lh3.googleusercontent.com/-Cyv9swstRns/Wu48DXCY_8I/AAAAAAAAiVE/U-mpXFbQGxwD1hvl3gnpbd-kHC861yhzwCK8BGAs/s386/2018-05-05.jpg",
					Short: true,
				},
			}
			attachment := &slack.Attachment{
				Pretext: "Who am i?",
				Color:   "#0a84c1",
				// ImageURL: "https://lh3.googleusercontent.com/-Cyv9swstRns/Wu48DXCY_8I/AAAAAAAAiVE/U-mpXFbQGxwD1hvl3gnpbd-kHC861yhzwCK8BGAs/s386/2018-05-05.jpg",
				Fields: fields,
			}

			attachmentChannel.Attachment = attachment
			c <- attachmentChannel
		}
	}
}

func handleBotReply() {
	for {
		params := slack.PostMessageParameters{}
		params.AsUser = true
		ac := <-botReplyChannel

		if ac.Attachment != nil {
			params.Attachments = []slack.Attachment{*ac.Attachment}
		} else {
			log.Println("[ERROR] could not retrieve list")
			attachment := &slack.Attachment{
				Pretext: "Unable to retrieve list",
				Color:   "#B733FF",
			}
			params.Attachments = []slack.Attachment{*attachment}

		}
		_, _, errPostMessage := api.PostMessage(ac.Channel.Name, ac.DisplayTitle, params)
		if errPostMessage != nil {
			log.Println("[ERROR]", errPostMessage)
		}
	}
}

func main() {

	api = slack.New(os.Getenv("SLACK_API_KEY"))
	rtm := api.NewRTM()

	botCommandChannel = make(chan *BotCentral)
	botReplyChannel = make(chan AttachmentChannel)

	go rtm.ManageConnection()
	go handleBotCommands(botReplyChannel)
	go handleBotReply()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				botId = ev.Info.User.ID
				log.Println(ev.Info.User)
				log.Println("connected", botId)
			case *slack.TeamJoinEvent:
				log.Println("Team join event")
			case *slack.MessageEvent:
				log.Println("message")
				channelInfo, err := api.GetChannelInfo(ev.Channel)
				if err != nil {
					log.Fatalln(err)
				}

				botCentral := &BotCentral{
					Channel: channelInfo,
					Event:   ev,
					UserId:  ev.User,
				}

				if ev.Type == "message" && strings.HasPrefix(ev.Text, "<@"+botId+">") {
					botCommandChannel <- botCentral
				}

				log.Println("reaction")

			case *slack.ReactionRemovedEvent:
				log.Println("reaction removed")

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				// Ignore other events..
				//fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}
