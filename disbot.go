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
        helpMessage := `I'm a bot written by Fran and powered by Claude.  Talk to me by starting your message with '^!^'.
Examples:

^!What is the mass of Jupiter?^
^!In philosophy, what is the Hard Problem of Consciousness?^
^!In 'The Lord of the Rings', who was Saruman?^

My replies will be brief, because I'm using Fran's API key to access Claude, and tokens cost money.

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

        // Remove the leading '!' from m.Content.
        userMessage := strings.TrimPrefix(m.Content, "!")

        // Complain if userMessage is longer than 1500 characters.
        if len(userMessage) > 1500 {
            session.ChannelMessageSend(m.ChannelID, "Sorry, I can't respond to messages that are longer than 1500 characters.")
            return
        }

        // Rmember the time of this command, so we can throttle replies if commands arrive to quickly.
        thisCommandTime := time.Now()

        if (!prevCommandTime.IsZero() && time.Since(prevCommandTime) < 30 * time.Second) {
            // There was a previous command and less than 5 seconds have passed since it was received.
            session.ChannelMessageSend(m.ChannelID, "Arrghhh!  I'm overloaded.  Please wait 30 seconds before trying again.")
        } else {
            // Generate a response from the AI and send it to the Discord server.
            aiResponse := generateResponse(userMessage)
            session.ChannelMessageSend(m.ChannelID, aiResponse)
        }

        prevCommandTime = thisCommandTime
    }
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
    systemPrompt := "You are a helpful assistant that provides concise and accurate answers to user queries.  The user is one of a set of Discord users connected to a single Discord server, but you cannot distinguish one user from another.  Your responses should short: no longer than 2 or 3 sentences.  If necessary, include links to Web sites in your responses."

    // Create the JSON request.
    requestBody, err := json.Marshal(map[string]interface{}{
        "model": "claude-sonnet-4-0",  // This is an alias for the latest Sonnet 4 version.
        "max_tokens": 1536,  // The maximum number of tokens the AI will generate.
        "system": systemPrompt,
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
    //         {
    //             "type": "text",
    //             "text": "..."
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
