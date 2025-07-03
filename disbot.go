package main

import (
    "fmt"
    "log"
    "os"
    "os/signal"
    "path/filepath"
    "strings"
    "time"

    "github.com/bwmarrin/discordgo"
)

var (
    Me = filepath.Base(os.Args[0])
    BotToken = ""
    startTime = time.Now()
    prevCommandTime time.Time
)

func main() {
    // Get the bot's auth token from the environment variable.
    BotToken = os.Getenv("DISCORD_BOT_TOKEN")

    if BotToken == "" {
        fmt.Printf("%s: Environment variable DISCORD_BOT_TOKEN is not set!\n", Me)
        os.Exit(1)
    }

    // Create a new Discord session using the bot token.
    dg, err := discordgo.New("Bot " + BotToken)
    if err != nil {
        log.Fatalf("Error creating Discord session: %v", err)
    }

    // Register the messageCreate func as a callback for MessageCreate events.
    dg.AddHandler(messageCreate)

    // In this example, we only care about receiving message events.
    dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

    // Open a websocket connection to Discord and begin listening.
    err = dg.Open()
    if err != nil {
        log.Fatalf("Error opening connection: %v", err)
    }

    fmt.Println("Bot is now running. Press CTRL-C to exit.")

    // Wait here until CTRL-C or other term signal is received.
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, os.Interrupt)
    <-sc

    // Cleanly close down the Discord session.
    dg.Close()
}

// This function will be called (due to AddHandler) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(session *discordgo.Session, m *discordgo.MessageCreate) {
    // Ignore all messages created by the bot itself
    if m.Author.ID == session.State.User.ID {
        return
    }

    // Ignore messages that don't start with the command prefix.
    if !strings.HasPrefix(m.Content, "!") {
        return
    }

    // Parse the command and arguments.
    parts := strings.Fields(m.Content)

    if len(parts) == 0 {
        return
    }

    command := strings.ToLower(parts[0])

    switch command {
    case "!help":
        helpMessage := `I'm a bot operated by Fran and powered by an AI.  Talk to me by starting your message with '^!^'.
Examples:

^!What is the mass of Jupiter?^
^!In philosophy, what is the Hard Problem of Consciousness?^
^!In 'The Lord of the Rings', who was Saruman?^

My replies will be brief.

I also respond to these commands:

^!help^   - Shows this help message.
^!status^ - Shows my status and uptime.`

        // Replace all '^'s in helpMessage with '`'.
        helpMessage = strings.ReplaceAll(helpMessage, "^", "`")

        session.ChannelMessageSend(m.ChannelID, helpMessage)

    case "!status":
        uptime := time.Since(startTime)
        msg := fmt.Sprintf("All systems are nominal.  I have been running for %s.", uptime.Round(time.Second))
        session.ChannelMessageSend(m.ChannelID, msg)

    default:
        // For all other uses of '!...' remove the leading '!' and send the rest to Claude to get a
        // reply.

        // Rmember the time of this command, so we can throttle replies if commands arrive to quickly.
        thisCommandTime := time.Now()

        if (!prevCommandTime.IsZero() && time.Since(prevCommandTime) < 5 * time.Second) {
            // There was a previous command and less than 5 seconds have passed since it was received.
            session.ChannelMessageSend(m.ChannelID, "I'm overloaded.  Please wait a bit before sending another command.")
        } else {
            session.ChannelMessageSend(m.ChannelID, "Sorry, I can't reply, because I'm not yet connected to my AI backend.")
        }

        prevCommandTime = thisCommandTime
    }
}
