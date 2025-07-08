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
    // The base name of this executeable (e.g., 'disbot').
    Me = filepath.Base(os.Args[0])

    // My Claude API key.  This will be set from an environment variable.
    apiKey = ""

    // The time this bot started.  used in status messages.
    startTime = time.Now()

    // The time of the last message received from a Discord user.  Used to throttle responses.
    prevMessageTime time.Time
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

    fmt.Println("Bot is running.  Press Ctrl-C to exit.")

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

    // For debugging.
    // fmt.Println("messageCreateEvent.Author.ID =", messageCreateEvent.Author.ID)

    // Strip leading and trailing whitespace.
    messageCreateEvent.Content = strings.TrimSpace(messageCreateEvent.Content)

    // Ignore empty messages.
    if len(messageCreateEvent.Content) == 0 {
        return
    }

    // Ignore messages that don't start with the command prefix.
    if !strings.HasPrefix(messageCreateEvent.Content, "!") {
        return
    }

    // Break the message string into words and extract the command word.
    messageParts := strings.Fields(messageCreateEvent.Content)
    command := strings.ToLower(messageParts[0])

    switch command {
    case "!help":
        // Display the help message.
        sendHelpMessage(session, messageCreateEvent)

    case "!status":
        // Display the status message.
        sendStatusMessage(session, messageCreateEvent)

    case "!!say":
        // Only Fran can use the '!!say' command.
        if messageCreateEvent.Author.ID != "555030984706359296" {
            msg := "Sorry, only Fran can use the '!!say' command."
            session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
            return
        }

        if len(messageParts) < 3 {
            msg := "Too few parameters.  Usage: `!!say CHANNELNAME MESSAGE`"
            session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
            return
        }

        // Get the message to send by removing '!say ' from the start of the message.
        message := strings.TrimPrefix(messageCreateEvent.Content, "!say ")

        // Get the channel name, which is the second word in the message.
        channelName := messageParts[1]

        // Remove the channel name from message.
        message = strings.TrimPrefix(message, channelName)

        // Remove all leading and trailing whitespace from message.
        message = strings.TrimSpace(message)

        // Send the message to the specified channel.
        errMsg := sendMessageToChannel(session, channelName, message)

        // Report status to the user to issued the '!!say ...' command.
        if errMsg != "" {
            // There was an error sending the message.  Send the error message to the channel where
            // the command was issued.
            session.ChannelMessageSend(messageCreateEvent.ChannelID, errMsg)
        } else {
            // The message was sent successfully, so send a confirmation message.
            msg := fmt.Sprintf("Message sent to channel '%s'.", channelName)
            session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
        }
        return

    default:
        // For all other uses of '!...', send the message to the AI to generate a reply and then
        // send it to the channel/DM .
        sendAIMessage(session, messageCreateEvent)
    }
}

// This function sends the help message to the channel/DM where messageCreateEvent came from.
func sendHelpMessage(session *discordgo.Session, messageCreateEvent *discordgo.MessageCreate) {
    // Must use '^' where we want a '`', due to Go's backtick quote syntax here.
    helpMsg := `I'm a bot written by Fran, Gemini, and Claude and powered by Claude. Talk to me by starting your message with '^!^'. For example:

• ^!What is the mass of Jupiter?^
• ^!What was the title of the Grateful Dead's second studio album?^
• ^!What was George Orwell's real name?^

You can also DM me, but you must use the '^!^' prefix even in DMs. My replies will be brief, because I use Fran's API key to access Claude, and tokens cost money. I don't know your Discord usernames. All of you appear to me as a single user. I have no memory of your previous messages to me (yet). I also respond to these commands:

^!status^ - Shows my status and uptime.
^!help^   - Shows this help message.`

    // Replace all '^'s in helpMsg with '`'.
    helpMsg = strings.ReplaceAll(helpMsg, "^", "`")

    session.ChannelMessageSend(messageCreateEvent.ChannelID, helpMsg)
}

// This function sends a status message to the channel/DM where messageCreateEvent came from.
func sendStatusMessage(session *discordgo.Session, messageCreateEvent *discordgo.MessageCreate) {
    states := []string{"nominal", "behaving", "rocking it", "within reason", "pretty good", "not too bad",
        "killing it", "grooving", "just peachy", "okey dokey", "fine, just fine",
        "... oh never mind", "reasonable", "adequate", "plausible", "howling", "superintelligent",
        "having a good day", "groovy", "👍", "🚀", "😎"}
    state := states[rand.Intn(len(states))]  // Get a random state string.
    uptime := time.Since(startTime)

    msg := fmt.Sprintf("All systems are %v.  I have been running for %v.", state, uptime.Round(time.Second))
    session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
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

// This function sends a message generated by the AI backend in response to the user's message.
func sendAIMessage(session *discordgo.Session, messageCreateEvent *discordgo.MessageCreate) {
    // Remove the leading '!' from messageCreateEvent.Content.
    userMessage := strings.TrimPrefix(messageCreateEvent.Content, "!")

    // Complain if userMessage is too long.
    maxUserMessageChars := 1000

    if len(userMessage) > maxUserMessageChars {
        msg := fmt.Sprintf("Sorry, I can't respond to messages that are longer than %v characters.",
            maxUserMessageChars)
        session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
        return
    }

    // Rmember the time of this message, so we can throttle replies if messages arrive to quickly.
    thisMessageTime := time.Now()

    // Do not allow users to message the bot too frequently.
    rateLimitWindow := 15 * time.Second
    timeSinceLastMessage := time.Since(prevMessageTime)
    timeUntilMessagesAllowed := rateLimitWindow - timeSinceLastMessage + 1

    if (!prevMessageTime.IsZero() && timeSinceLastMessage < rateLimitWindow) {
        // Too little time has passed since the previous message to this bot.
        msg := fmt.Sprintf("Arrghhh! I'm overloaded. Please wait %v seconds before talking to me.",
            timeUntilMessagesAllowed)
        session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
    } else {
        // Generate a response from the AI.
        aiResponse := generateResponse(userMessage)

        // Send the response text to the Discord server.
        session.ChannelMessageSend(messageCreateEvent.ChannelID, aiResponse)
    }

    // Remember the time that this message was processed.
    prevMessageTime = thisMessageTime
}

// This function obtains an AI-generated response to a user message received from Discord.  If
// successful, it returns the AI-generated response, otherwise it returns a string describing the
// nature of the error.
func generateResponse(userMessage string) string {
    // Get the AI API key.
    if apiKey == "" {
        apiKey = os.Getenv("ANTHROPIC_API_KEY")

        if apiKey == "" {
            msg := "Error: Environment variable ANTHROPIC_API_KEY not set."
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
        msg := fmt.Sprintf("Error: Error creating request body: %s", err)
        fmt.Println(msg)
        return msg
    }

    // Create the HTTP request from the above requestBody.
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))

    if err != nil {
        msg := fmt.Sprintf("Error: Error creating HTTP request: %s", err)
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
        msg := fmt.Sprintf("Error: Network communication error: %s", err)
        fmt.Println(msg)
        return msg
    }

    // Close the HTTP connection at this function's return.
    defer resp.Body.Close()

    // Handle HTTP errors.
    if resp.StatusCode != http.StatusOK {
        msg := fmt.Sprintf("Error: HTTP error: %s", resp.Status)
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
        msg := fmt.Sprintf("Error: Error reading AI response: %s", err)
        fmt.Println(msg)
        return msg
    }

    // For debugging.
    // fmt.Println("Got JSON:", string(jsonBytes[:bytesRead]))

    var response map[string]interface{}

    // Unmarshal the JSON.  Must use jsonBytes[:bytesRead] to avoid reading beyond the end of the
    // data in the slice (see above make([]byte, ...)).
    err = json.Unmarshal(jsonBytes[:bytesRead], &response)

    if err != nil {
        msg := fmt.Sprintf("Error: Error unmarshalling AI response: %s", err)
        fmt.Println(msg)
        return msg
    }

    // Check if the response contains content.
    content, ok := response["content"].([]interface{})

    if !ok || len(content) == 0 {
        msg := "Error: AI response does not contain the expected JSON."
        fmt.Println(msg)
        return msg
    }

    // Extract the text from the first content item.
    firstContent := content[0].(map[string]interface{})
    text, ok := firstContent["text"].(string)

    if !ok {
        msg := "Error: AI response content does not contain text."
        fmt.Println(msg)
        return msg
    }

    // For debugging.
    // fmt.Printf("AI response: %s\n", text)

    // Return the AI-generated response text.
    return text
}

// This function sends a message to an arbitrary channel.  Returns the empty string if successful,
// otherwise returns an error message string.
func sendMessageToChannel(session *discordgo.Session, channelName string, message string) string {
    // Get all channels in the server.
    channels, err := session.GuildChannels("840286104296489000")

    if err != nil {
        return fmt.Sprintf("Error: Failed to get server channel list: %v", err)
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
        return fmt.Sprintf("Error: Channel '%s' not found!", channelName)
    }

    // Send the message to the found channel
    _, err = session.ChannelMessageSend(targetChannelID, message)

    if err != nil {
        return fmt.Sprintf("Error: Failed to send message to channel: %v", err)
    }

    // Return the empty string on success.
    return ""
}
