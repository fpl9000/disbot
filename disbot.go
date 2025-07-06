package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "log"
    "os"
    "os/signal"
    "path/filepath"
    "math/rand"
    "strings"
    "time"

    "github.com/bwmarrin/discordgo"
)

// Package scope variables.
var (
    Me = filepath.Base(os.Args[0])
    apiKey = ""
    startTime = time.Now()
    prevCommandTime time.Time
)

func main() {
    // Get the bot's auth token from the environment variable.
    botToken := os.Getenv("DISCORD_BOT_TOKEN")

    if botToken == "" {
        fmt.Printf("%s: Environment variable DISCORD_BOT_TOKEN is not set!\n", Me)
        os.Exit(1)
    }

    // Create a new Discord session using the bot token.
    dg, err := discordgo.New("Bot " + botToken)
    if err != nil {
        log.Fatalf("Error creating Discord session: %v", err)
    }

    // Register the handleMessageCreateEvent func as a callback for MessageCreate events.
    dg.AddHandler(handleMessageCreateEvent)

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
func handleMessageCreateEvent(session *discordgo.Session, messageCreateEvent *discordgo.MessageCreate) {
    // Ignore all messages created by the bot itself
    if messageCreateEvent.Author.ID == session.State.User.ID {
        return
    }

    // Ignore messages that don't start with the command prefix.
    if !strings.HasPrefix(messageCreateEvent.Content, "!") {
        return
    }

    // Parse the command and arguments.
    parts := strings.Fields(messageCreateEvent.Content)

    if len(parts) == 0 {
        return
    }

    command := strings.ToLower(parts[0])

    switch command {
    case "!help":
        // Must use '^' where we want a '`', due to Go's backtick quote syntax here.
        helpMsg := `I'm a bot written by Fran and powered by Claude.  Talk to me by starting your message with '^!^'. For example:

^!What is the mass of Jupiter?^
^!In 'The Lord of the Rings', who was Saruman?^

My replies will be brief, because I'm using Fran's API key to access Claude, and tokens cost money.  I also respond to these commands:

^!help^   - Shows this help message.
^!status^ - Shows my status and uptime.`

        // Replace all '^'s in helpMsg with '`'.
        helpMsg = strings.ReplaceAll(helpMsg, "^", "`")

        session.ChannelMessageSend(messageCreateEvent.ChannelID, helpMsg)

    case "!status":
        states := []string{"nominal", "behaving", "normal", "operational", "operating as expected", "crazy good",
                           "within reason", "pretty good, given the state of the world", "not too bad", "killing it",
                           "grooving", "just peachy", "okey dokey", "fine, just fine", "... oh never mind"}
        state := states[rand.Intn(len(states))]  // Get a random state string.
        uptime := time.Since(startTime)

        msg := fmt.Sprintf("All systems are %v.  I have been running for %v.", state, uptime.Round(time.Second))
        session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)

    default:
        // For all other uses of '!...' remove the leading '!' and send the rest to Claude to get a
        // reply.

        // Remove the leading '!' from messageCreateEvent.Content.
        userMessage := strings.TrimPrefix(messageCreateEvent.Content, "!")

        // Complain if userMessage is longer than 1500 characters.
        if len(userMessage) > 1500 {
            msg := "Sorry, I can't respond to messages that are longer than 1500 characters."
            session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
            return
        }

        // Rmember the time of this command, so we can throttle replies if commands arrive to quickly.
        thisCommandTime := time.Now()

        if (!prevCommandTime.IsZero() && time.Since(prevCommandTime) < 30 * time.Second) {
            // There was a previous command and less than 5 seconds have passed since it was received.
            msg := "Arrghhh!  I'm overloaded.  Please wait 30 seconds before trying again."
            session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
        } else {
            // Generate a response from the AI.
            aiResponse := generateResponse(userMessage)

            // Send the response text to the Discord server.
            session.ChannelMessageSend(messageCreateEvent.ChannelID, aiResponse)
        }

        prevCommandTime = thisCommandTime
    }
}

// This function returns the system prompt to be sent in each JSON request to the AI.
func getSystemPrompt() string {
    todaysDate := time.Now().Format(time.DateOnly)

    sysPrompt := fmt.Sprintf("Today's date is %s.  You are a helpful assistant that provides concise and accurate " +
        "answers to user queries.  Your responses should be short: only 2 or 3 sentences.  Your user is one of a " +
        "set of people connected to a Discord server (as are you), but you cannot distinguish one user from another.",
        todaysDate)

    return sysPrompt
}

// This function obtains an AI-generated response to a user message received from Discord.  If
// successful, it returns the AI-generated response, otherwise it returns a string describing the
// nature of the error.
func generateResponse(userMessage string) string {
    // Get the AI API key.
    if apiKey == "" {
        apiKey = os.Getenv("ANTHROPIC_API_KEY")

        if apiKey == "" {
            msg := "Oops: Environment variable ANTHROPIC_API_KEY not set."
            fmt.Println(msg)
            return msg
        }
    }

    // See https://docs.anthropic.com/en/api/overview for details.

    // This is the API endpoint URL.
    url := "https://api.anthropic.com/v1/messages"

    // This is the AI's system prompt.
    systemPrompt := getSystemPrompt()

    // Create the JSON request.
    requestBody, err := json.Marshal(map[string]interface{}{
        "model": "claude-sonnet-4-0",  // This is an alias for the latest Sonnet 4 version.
        "max_tokens": 1536,  // The maximum number of tokens the AI will generate.
        "system": systemPrompt,
//      "thinking": {
//          "type": "enabled",
//          "budget_tokens": 10000  // Must be smaller than 'max_tokens' above.
//      },
        "messages": []map[string]string{  // An array of maps.
            { "role": "user",
              "content": userMessage },
        },
    })

    if err != nil {
        msg := fmt.Sprintf("Oops: Error creating request body: %s", err)
        fmt.Println(msg)
        return msg
    }

    // Create the HTTP request from the above requestBody.
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))

    if err != nil {
        msg := fmt.Sprintf("Oops: Error creating HTTP request: %s", err)
        fmt.Println(msg)
        return msg
    }

    // Set some required HTTP headers.
    req.Header.Set("x-api-key", apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("Content-Type", "application/json")

    // Send the HTTP request to the AI.
    client := &http.Client{}
    resp, err := client.Do(req)

    if err != nil {
        msg := fmt.Sprintf("Oops: Network communication error: %s", err)
        fmt.Println(msg)
        return msg
    }

    // Close the HTTP connection at this function's return.
    defer resp.Body.Close()

    // Handle HTTP errors.
    if resp.StatusCode != http.StatusOK {
        msg := fmt.Sprintf("Oops: HTTP error: %s", resp.Status)
        fmt.Println(msg)
        return msg
    }

    // Get the 'Content-Length' header.
    contentLength := resp.ContentLength

    if contentLength <= 0 {
        contentLength = 50 * 1024   // 50 kb should be enough.
    }

    // Get the text of the body of the response.
    jsonBytes := make([]byte, contentLength)

    bytesRead, err := resp.Body.Read(jsonBytes)

    if err != nil {
        msg := fmt.Sprintf("Oops: Error reading AI response: %s", err)
        fmt.Println(msg)
        return msg
    }

    // For debugging.
    // fmt.Println("Got JSON:", string(jsonBytes[:bytesRead]))

    // Unmarshal the JSON in the response.  The JSON in the response has this form:
    //
    // {
    //     "id": "msg_01FLtpFj1qRsnPKs5UygTinR",
    //     "type": "message",
    //     "role": "assistant",
    //     "model": "claude-sonnet-4-20250514",
    //     "content": [
    //         {   // Array element 0.  Only present when thinking is enabled.
    //             "type": "thinking",
    //             "thinking": "<THINKING TEXT HERE>...",
    //             "signature": "WaUjzkypQ2mUEVM36O2TxuC06KN8xyfbJwyem2dw3UjavL...."
    //         },
    //         {   // Array element 1.
    //             "type": "text",
    //             "text": "<AI RESPONSE HERE>..."
    //         }
    //     ],
    //     "stop_reason": "end_turn",
    //     "stop_sequence": null,
    //     "usage": {
    //         "input_tokens": 23,
    //         "cache_creation_input_tokens": 0,
    //         "cache_read_input_tokens": 0,
    //         "output_tokens": 127,
    //         "service_tier": "standard"
    //     }
    // }

    var response map[string]interface{}

    // Unmarshal the JSON.  Must use jsonBytes[:bytesRead] to avoid reading beyond the end of the
    // data in the slice (see above make([]byte, ...)).
    err = json.Unmarshal(jsonBytes[:bytesRead], &response)

    if err != nil {
        msg := fmt.Sprintf("Oops: Error unmarshalling AI response: %s", err)
        fmt.Println(msg)
        return msg
    }

    // Check if the response contains content.
    content, ok := response["content"].([]interface{})

    if !ok || len(content) == 0 {
        msg := "Oops: AI response does not contain the expected JSON."
        fmt.Println(msg)
        return msg
    }

    // Extract the text from the first content item.
    firstContent := content[0].(map[string]interface{})
    text, ok := firstContent["text"].(string)

    if !ok {
        msg := "Oops: AI response content does not contain text."
        fmt.Println(msg)
        return msg
    }

    // For debugging.
    // fmt.Printf("AI response: %s\n", text)

    // Return the AI-generated response text.
    return text
}

// This function sends a message to an arbitrary channel.  Returns nil if successful, otherwise an
// instance of type error.
func sendMessageToChannel(session *discordgo.Session, channelName string, message string) error {
    // Get all channels in the server.
    channels, err := session.GuildChannels("840286104296489000")

    if err != nil {
        return fmt.Errorf("Error: Failed to get server channel list: %v", err)
    }

    // Find the channel ID from the channel name.
    var targetChannelID string

    for _, channel := range channels {
        // Check if this is a text channel (not voice, category, etc.) and matches the specified name.
        if channel.Type == discordgo.ChannelTypeGuildText && channel.Name == channelName {
            targetChannelID = channel.ID
            break
        }
    }

    // Check if we found the channel
    if targetChannelID == "" {
        return fmt.Errorf("Error: Channel '%s' not found in guild", channelName)
    }

    // Send the message to the found channel
    _, err = session.ChannelMessageSend(targetChannelID, message)

    if err != nil {
        return fmt.Errorf("Error: Failed to send message to channel: %v", err)
    }

    fmt.Printf("Message sent successfully to channel '%s'\n", channelName)
    return nil
}
