package main

import (
    "bytes"
    "container/list"
    "encoding/json"
    "fmt"
    "net/http"
    "log"
    "os"
    "os/signal"
    "path/filepath"
    "math"
    "math/rand"
    "strings"
    "strconv"
    "sync"
    "time"

    "github.com/bwmarrin/discordgo"
)

// Type aliases.
type List = list.List

// Package scope constants.
const DEFAULT_MAX_RECENT_MESSAGES = 10

// Package scope variables.
var (
    // The base name of this executeable (e.g., 'disbot').
    Me = strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")

    // The AI's API key.  This is set from an environment variable.
    apiKey = ""

    // The bot's Discord authentication token.  This is set from an environment variable.
    botToken = ""

    // The time this bot started.  used in status messages.
    startTime = time.Now()

    // Per-conversation timestamps of the last message received. Used to throttle responses
    // so that one user's messages don't block other conversations.
    lastMessageAt  = make(map[string]time.Time)
    lastMessageMu  sync.Mutex

    // This is true if Web search is enable in the query to the AI.  Override this with switch
    // --nosearch.
    webSearchEnabled = true

    // This is true if reasoning is enabled in the query to the AI.  Override this with switch
    // --nothink.
    reasoningEnabled = true

    // The 'max_tokens' value sent in each AI request.
    maxTokens = 2048

    // When reasoningEnabled is true, this is the reasoning 'budget_tokens' value send in each AI
    // request.  If this is smaller than 1024, Claude's API fails with HTTP error 400 (Bad Request).
    thinkingMaxTokens = 1024

    // When webSearchEnabled is true, is the 'max_uses' value send in the Web search tool definition
    // in each AI request.
    maxWebSearches = 1

    // The maximum number of messages to save in each channel's or user's conversation history
    // (incuding the AI's messages).  Must be an even number, because each conversation history
    // should contain pairs of user/AI messages.  Switch --history overrides the default value of
    // this variable.
    maxRecentMessages = DEFAULT_MAX_RECENT_MESSAGES
)

// Display usage and terminate.
func usage() {
    msg := "usage: " + Me + " [ --search ] [ --think ] [ --history N ]\n\n" +
           "--nosearch   =>  Disable Web searching in the AI.\n" +
           "--nothink    =>  Disable reasoning in the AI.\n" +
           "--history N  =>  Keep N most recent user/AI messages (default: %v).\n" +
           "                 (N must be an even integer.)\n"

    fmt.Printf(msg, DEFAULT_MAX_RECENT_MESSAGES)
    os.Exit(1)
}

// Package initialization.
func init() {
    // Get the AI API key from an environment variable.
    apiKey = os.Getenv("ANTHROPIC_API_KEY")

    if apiKey == "" {
        fmt.Println("Error: Environment variable ANTHROPIC_API_KEY not set.")
        os.Exit(1)
    }

    // Get the bot's authentication token from an environment variable.
    botToken = os.Getenv("DISCORD_BOT_TOKEN")

    if botToken == "" {
        fmt.Printf("%v: Environment variable DISCORD_BOT_TOKEN is not set!\n", Me)
        os.Exit(1)
    }
}

func main() {
    // Parse the command-line switches.  This will set various package-scope variables based on the
    // command-line switches (or show usage and terminate in the case of erroneous usage).
    parseCommandLine()

    if reasoningEnabled {
        fmt.Println("Reasoning is enabled.")
    }

    if webSearchEnabled {
        fmt.Println("Web search is enabled.")
    }

    // Create a new Discord session using the bot token.
    dg, err := discordgo.New("Bot " + botToken)
    if err != nil {
        log.Fatalf("Error creating Discord session: %v", err)
    }

    // Register the handleMessageCreateEvent func as a callback for MessageCreate events.
    dg.AddHandler(handleMessageCreateEvent)

    // In this example, we only care about receiving message events from channels (aka guilds) and
    // from DMs.
    dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

    // Open a websocket connection to Discord and begin listening.
    err = dg.Open()
    if err != nil {
        log.Fatalf("Error opening connection: %v", err)
    }

    fmt.Println("Bot is running.  Press Ctrl-C to exit.")

    // Wait here until Ctrl-C or other term signal is received.
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, os.Interrupt)
    <-sc

    // Cleanly close down the Discord session.
    dg.Close()
}

// This function parses the command-line arguments and sets various package-scope variables based on
// those switches.  If the command-line arguments are invalid, it shows usage and exits the program.
func parseCommandLine() {
    if len(os.Args) > 3 {
        fmt.Println("Too many parameters!\n")
        usage()
    }

    // Check for command-line switches.
    for index := 1; index < len(os.Args); index++ {
        argument := os.Args[index]

        switch argument {
        case "--nosearch":
            // Disable Web searching in the AI.
            webSearchEnabled = false

        case "--nothink":
            // Disable reasoning in the AI.
            reasoningEnabled = false

        case "--help", "-h":
            // Show usage and exit.
            usage()

        case "--history":
            // Set the maximum number of recent messages to keep.
            index++ // Advance to the next argument, which should be the number of messages to keep.

            if index >= len(os.Args) {
                fmt.Printf("%v: Missing parameter for switch '%v'!\n\n", Me, argument)
                usage()
            }

            // Get the number of messages to keep.
            var err error
            maxRecentMessages, err = strconv.Atoi(os.Args[index])

            if err != nil || maxRecentMessages <= 0 {
                fmt.Printf("%v: Invalid parameter for switch '%v': '%v'!\n\n", Me, argument, os.Args[index])
                usage()
            }

            if maxRecentMessages % 2 != 0 {
                fmt.Printf("%v: The value for switch '%v' must be an even number!\n\n", Me, argument)
                usage()
            }

        default:
            fmt.Printf("%v: Unrecognized switch: '%v'!\n\n", Me, argument)
            usage()
        }
    }
}

// This function will be called (due to AddHandler) every time a new message is seen by this bot in
// any channel or DM.
func handleMessageCreateEvent(session *discordgo.Session, messageCreateEvent *discordgo.MessageCreate) {
    // Ignore all messages created by the bot itself
    if messageCreateEvent.Author.ID == session.State.User.ID {
        return
    }

    // Strip leading and trailing whitespace from the message.
    userMessage := strings.TrimSpace(messageCreateEvent.Content)

    // Ignore empty messages and messages that don't start with the command prefix.
    if len(userMessage) == 0 || !strings.HasPrefix(userMessage, "!") {
        return
    }

    // Break the message string into words and extract the command word.
    messageParts := strings.Fields(userMessage)
    command := strings.ToLower(messageParts[0])

    // Get the channel ID that will be used to send messages to the channel.  This is NOT the
    // channel name.
    channelID := messageCreateEvent.ChannelID

    switch command {
    case "!help":
        // Display the help message.
        sendHelpMessage(session, channelID)

    case "!status":
        // Display the status message.
        sendStatusMessage(session, channelID)

    case "!!say":
        // Process the '!!say ...' command.
        handleSayCommand(session, messageCreateEvent, messageParts)

    default:
        // For all other uses of '!...', send the message to the AI to generate a reply and then
        // send it to the channel/DM .

        // First, get the name of the channel where the message was received by this bot, the
        // nick of the user who sent the message, and whether it's a DM. If the message is a DM,
        // the channel name is the empty string. If either name is unavailable, it will be "unknown".
        channelName, nick, isDM := getChannelAndNick(session, messageCreateEvent)

        // Compute the conversation key used to store history and rate-limit state.
        convKey := conversationKey(channelID, messageCreateEvent.Author.ID, isDM)

        // Generate the response and send it.
        sendAIGeneratedResponse(session, channelID, channelName, nick, convKey, userMessage)
    }
}

// This function returns the name of the channel where the message was sent, the nick of the user
// who sent the message, and whether the message is a DM. If the message is a DM, the channel
// name is the empty string and isDM is true. If either name is unavailable due to an error, it
// is returned as "unknown" and an error is written to stdout.
func getChannelAndNick(session *discordgo.Session, messageCreateEvent *discordgo.MessageCreate) (string, string, bool) {
    // Get the channel the message was sent from.
    var channelName string
    var isDM bool

    channel, err := session.Channel(messageCreateEvent.ChannelID)

    if err != nil {
        // We don't know which channel this message was sent from.
        fmt.Printf("%v: getChannelAndNick: Error getting channel name: %v\n", Me, err)
        channelName = "unknown"
    } else {
        channelName = channel.Name // In a DM, this is the empty string.
        isDM = (channel.Type == discordgo.ChannelTypeDM || channel.Type == discordgo.ChannelTypeGroupDM)
    }

    // Get the nick of the Discord user who sent the message.
    var nick string

    if messageCreateEvent.Author == nil || messageCreateEvent.Author.GlobalName == "" {
        // We don't know the nick of the user who sent this message.
        fmt.Println("messageCreateEvent.Author = nil")
        nick = "unknown"
    } else {
        nick = messageCreateEvent.Author.GlobalName
    }

    return channelName, nick, isDM
}

// This function sends the help message to the channel/DM where messageCreateEvent came from.
func sendHelpMessage(session *discordgo.Session, channelID string) {
    helpMsg := "I'm a bot written in Go by Fran, Gemini, and Claude.  My responses are generated by Claude. " +
               "Talk to me by starting your message with '`!`'. For example:\n\n" +
               "• `!What is the mass of Jupiter?`\n" +
               "• `!What was the title of the Grateful Dead's second studio album?`\n" +
               "• `!What was George Orwell's real name?`\n\n" +
               "You can also DM me, but you must use the '`!`' prefix even in DMs. My replies will " +
               "be brief, because tokens cost money. I don't know your Discord usernames. All of you " +
               "appear to me as a single user. I have no memory of your previous messages to me (yet). " +
               "I also respond to these commands:\n\n" +
               "• `!status` - Shows my status and uptime.\n" +
               "• `!help`   - Shows this help message."

    session.ChannelMessageSend(channelID, helpMsg)
}

// This function sends a status message to the channel/DM where messageCreateEvent came from.
func sendStatusMessage(session *discordgo.Session, channelID string) {
    states := []string{"nominal", "behaving", "rocking it", "within reason", "pretty good", "being real",
                       "killing it", "grooving", "just peachy", "okey dokey", "fine, just fine",
                       "... oh never mind", "reasonable", "adequate", "plausible", "howling", "meh",
                       "superintelligent", "having a good day", "groovy", "👍", "🚀", "😎"}
    state := states[rand.Intn(len(states))]  // Get a random state string.
    uptime := time.Since(startTime)

    msg := fmt.Sprintf("All systems are %v.  I have been running for %v.", state, uptime.Round(time.Second))

    if webSearchEnabled {
        msg += " Web searching is enabled."
    }

    if reasoningEnabled {
        msg += " Extended thinking is enabled."
    }

    session.ChannelMessageSend(channelID, msg)
}

// This function handles the '!!say CHANNEL MESSAGE' command.
func handleSayCommand(session *discordgo.Session, messageCreateEvent *discordgo.MessageCreate,
                      messageParts []string) {
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

    // Remove the channel name and leading/trailing whitespace from message.
    message = strings.TrimSpace(strings.TrimPrefix(message, channelName))

    // Add a prefix saying this message is from Fran.
    message = "Fran asked me to say this: " + message

    // Send the message to the specified channel.
    errMsg := sendMessageToChannel(session, channelName, message)

    // Report status to the user to issued the '!!say ...' command.
    if errMsg != "" {
        // There was an error sending the message.  Send the error message to the channel where
        // the command was issued.
        session.ChannelMessageSend(messageCreateEvent.ChannelID, errMsg)
    } else {
        // The message was sent successfully, so send a confirmation message.
        msg := fmt.Sprintf("Message sent to channel '%v'.", channelName)
        session.ChannelMessageSend(messageCreateEvent.ChannelID, msg)
    }
}

// TODO: Move all AI-related functions to new source file ai.go.

// =============================================================================
// UNDER CONSTRUCTION
// =============================================================================

// This function sends a message generated by the AI backend in response to the user's message.
func sendAIGeneratedResponse(session *discordgo.Session, channelID string, channelName string,
                             nick string, convKey string, userMessage string) {
    // Remove the leading '!' from messageCreateEvent.Content.
    userMessage = strings.TrimPrefix(userMessage, "!")

    // Complain if userMessage is too long.
    maxUserMessageChars := 1000

    if len(userMessage) > maxUserMessageChars {
        msg := fmt.Sprintf("Sorry, I can't respond to messages that are longer than %v characters.",
                           maxUserMessageChars)
        session.ChannelMessageSend(channelID, msg)
        return
    }

    // Remember the time of this message, so we can throttle replies if messages arrive too quickly.
    thisMessageTime := time.Now()

    // Check per-conversation rate limiting so that one user's messages don't throttle everyone.
    minSecondsBetweenMessages := (10 * time.Second).Seconds()

    lastMessageMu.Lock()
    prevTime := lastMessageAt[convKey]
    lastMessageMu.Unlock()

    secondsSinceLastMessage := time.Since(prevTime).Seconds()
    secondsUntilMessagesAllowed := math.Round(minSecondsBetweenMessages - secondsSinceLastMessage + 0.5)

    if !prevTime.IsZero() && secondsSinceLastMessage < minSecondsBetweenMessages {
        // Too little time has passed since the previous message in this conversation.
        msg := fmt.Sprintf("Sorry, I'm overloaded. Please wait %v seconds before talking to me.",
                           secondsUntilMessagesAllowed)
        session.ChannelMessageSend(channelID, msg)
    } else {
        // Generate a response from the AI.
        aiResponse := getAIResponse(userMessage, channelName, nick, convKey)

        // Send the response text to the Discord server.
        session.ChannelMessageSend(channelID, aiResponse)
    }

    // Remember the time that this message was processed for this conversation.
    lastMessageMu.Lock()
    lastMessageAt[convKey] = thisMessageTime
    lastMessageMu.Unlock()
}

// This function obtains an AI-generated response to a user message received from Discord.  If
// successful, it returns the AI-generated response, otherwise it returns a string describing the
// nature of the error.  
func getAIResponse(userMessage string, channelName string, nick string, convKey string) string {
    // This is the API endpoint URL.  See https://docs.anthropic.com/en/api/overview for details
    // about the Claude API.
    url := "https://api.anthropic.com/v1/messages"

    // Save the user message as the newest element in the conversation history.  This must happen
    // before we call json.Marshal(jsonObject).
    historySaveNewMessage("user", userMessage, convKey)

    // Create the JSON request.  jsonObject will be passed to json.Marshal to convert it into JSON.
    jsonObject := make(map[string]any)

    // TODO: Switch these next two lines.
    //jsonObject["model"] = "claude-sonnet-4-0"  // This is an alias for the latest Sonnet 4 version.
    jsonObject["model"] = "claude-sonnet-4-20250514"
    jsonObject["max_tokens"] = maxTokens         // The maximum number of tokens the AI will generate.
    jsonObject["system"] = getSystemPrompt()

    // Get the message history for this channel or nick.  Use nick if channelName is the empty
    // string, which means we're handling a DM from a user to the bot.
    var recentMessagesSlice []map[string]string
    var err error

    recentMessagesSlice, err = historyAsSlice(convKey)

    if err != nil {
        msg := fmt.Sprintf("%v: getAIResponse: historyAsSlice failed: %v", Me, err)
        fmt.Println(msg)
        return msg
    }

    // Store the recent messages slice in map jsonObject.
    jsonObject["messages"] = recentMessagesSlice

    if reasoningEnabled {
        // Here, 'budget_tokens' must be smaller than 'max_tokens' above.
        jsonObject["thinking"] = map[string]any{ "type": "enabled", "budget_tokens": thinkingMaxTokens }
    }

    if webSearchEnabled {
        jsonObject["tools"] = []map[string]any{{"type": "web_search_20250305",
                                                "name": "web_search",
                                                "max_uses": maxWebSearches }}
    }

    requestBody, err := json.Marshal(jsonObject)

    if err != nil {
        msg := fmt.Sprintf("%v: getAIResponse: json.Marshal failed: %v", Me, err)
        fmt.Println(msg)
        return msg
    }

    // For debugging.
    // fmt.Println("Request JSON =", string(requestBody))

    // Create the HTTP request from the above requestBody.
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))

    if err != nil {
        msg := fmt.Sprintf("%v: getAIResponse: Error creating HTTP request: %v", Me, err)
        fmt.Println(msg)
        return msg
    }

    // Set some required HTTP headers.
    req.Header.Set("x-api-key", apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("Content-Type", "application/json")

    // Send the HTTP request to the AI and get the HTTP response.  The body of the response is JSON
    // containing the AI's response to the user's message.
    client := &http.Client{}
    httpResponse, err := client.Do(req)

    if err != nil {
        msg := fmt.Sprintf("%v: getAIResponse: Network communication error: %v", Me, err)
        fmt.Println(msg)
        return msg
    }

    // Close the HTTP connection at this function's return.
    defer httpResponse.Body.Close()

    // Handle HTTP errors.
    if httpResponse.StatusCode != http.StatusOK {
        msg := fmt.Sprintf("%v: getAIResponse: HTTP error: %v", Me, httpResponse.Status)
        fmt.Println(msg)
        return msg
    }

    // Parse the HTTP response from the AI and return the text of the response.
    return parseAIResponse(httpResponse, convKey)
}

// This function returns the system prompt to be sent in each JSON request to the AI.
func getSystemPrompt() string {
    todaysDate := time.Now().Format(time.DateOnly)

    var webSearchPrompt string

    if reasoningEnabled {
        webSearchPrompt = "Only use the Web search tool when you do not have the necessary " +
                          "knowledge to respond. "
    }

    return fmt.Sprintf("Today's date is %s. You are a helpful assistant that provides concise and " +
                       "accurate answers to user queries. Your responses should be short: only 2 or 3 " +
                       "sentences. " +
                       webSearchPrompt +
                       "The user is one of a group of people connected to a Discord server (as are you), " +
                       "but you cannot distinguish one user from another. Your output must use Discord " +
                       "markdown so that it renders correctly.", todaysDate)
}

// This function processes the HTTP response from the AI and returns the AI-generated response text.
func parseAIResponse(httpResponse *http.Response, convKey string) string {
    jsonBytes, jsonBytesCount, msg := getJSONFromHTTPResponse(httpResponse)

    if msg != "" {
        return msg
    }

    // This holds the unmarshaled JSON response from the AI.
    var response map[string]any

    // Unmarshal the JSON into object 'response'.  Must use jsonBytes[:jsonBytesCount] to avoid reading
    // beyond the end of the valid data in slice jsonBytes.
    err := json.Unmarshal(jsonBytes[:jsonBytesCount], &response)

    if err != nil {
        msg := fmt.Sprintf("Error: Error unmarshalling AI response: %s", err)
        fmt.Println(msg)
        return msg
    }

    // Check if the response contains a 'content' key.
    contentSlice, ok := response["content"].([]any)

    if !ok || len(contentSlice) == 0 {
        msg := "Error: Failed to find expected JSON (#0)."
        fmt.Println(msg)
        return msg
    }

    // This will hold the text returned by the AI.
    aiText := ""

    // This will hold the reasoning trace returned by the AI.
    thinkingText := ""

    // TODO: Refactor this for loop into a new function.

    // =============================================================================
    // UNDER CONSTRUCTION
    // =============================================================================

    // Iterate over all elements of contentSlice and concatenate the text.  contentSlice is a slice
    // of maps.  This loop extracts the text from each element of contentSlice that has a "type" key
    // with value "text", concatenates the text, and returns the concatenated text.  All other
    // "type" values are ignored (e.g., "server_tool_use", "web_search_tool", and "citations"), but
    // When reasoningEnabled is true, this also handles "type" value "thinking", which comes with
    // key "thinking" whose value is the reasoning trace.

    for index := 0; index < len(contentSlice); index++ {
        // Get the map from contentSlice[index].
        contentElement, ok := contentSlice[index].(map[string]any)

        if !ok {
            msg := "Error: Failed to find expected JSON (#1)."
            fmt.Println(msg)
            return msg
        }

        // Ignore all content types except "text" and "thinking".

        if contentElement["type"] == "text" {
            // Extract the text value associated with key "text".
            elementText, ok := contentElement["text"].(string)

            if !ok {
                msg := "Error: Failed to find expected JSON (#2)."
                fmt.Println(msg)
                return msg
            }

            aiText += elementText

            // For debugging.
            // fmt.Printf("text: %v: aiText = '%s'\n", index, aiText)
        }

        if contentElement["type"] == "thinking" {
            // Extract the text value associated with key "thinking".
            elementText, ok := contentElement["thinking"].(string)

            if !ok {
                msg := "Error: Failed to find expected JSON (#3)."
                fmt.Println(msg)
                return msg
            }

            // Append the thinking text to the AI response.
            thinkingText += elementText

            // For debugging.
            // fmt.Printf("thinking: %v: thinkingText = '%s'\n", index, thinkingText)
        }
    }

    // Update the conversation history to have the AI's response.
    historySaveNewMessage("assistant", thinkingText+"\n\n"+aiText, convKey)

    // Return the AI-generated response.
    if reasoningEnabled {
        return "**<thinking>**" + thinkingText + "\n**</thinking>**\n\n" + aiText
    } else {
        return aiText
    }
}

// This function reads the body of the HTTP response and returns the JSON as a byte slice, the
// number of bytes read, and an error if any.  If no error occurs, the string returned is "".
func getJSONFromHTTPResponse(httpResponse *http.Response) ([]byte, int, string) {
    // Get the 'Content-Length' header so we know how big to make the byte slice that will hold
    // the response body.
    contentLength := httpResponse.ContentLength

    // For debugging.
    // fmt.Printf("contentLength = %v\n", contentLength)

    if contentLength <= 0 {
        // Sometimes the Content-Length header is -1, so we have to wing it.  Hopefully, 1 MB is
        // enough space.
        contentLength = 1024 * 1024
    }

    // Get the text of the body of the response, which contains JSON.
    jsonBytes := make([]byte, contentLength)
    jsonBytesCount := 0

    // Each call to httpResponse.Body.Read will fill some (or all) of this sub-slice of jsonBytes
    // with the next group of bytes, then if necessary the start of this sub-slice will be advanced
    // along slice jsonBytes to be ready for the next call to Read.
    jsonBytesForReading := jsonBytes[:contentLength] 

    // Read the entire response body by calling httpResponse.Body.Read() in a loop until err != nil.
    for {
        bytesRead, err := httpResponse.Body.Read(jsonBytesForReading)

        // For debugging.
        // if bytesRead == 0 && err.Error() != "EOF" {
        //     fmt.Printf("WARNING: bytesRead == 0: err == %v\n", err)
        // }

        // Increment the accumulated number of  bytes we just read.
        jsonBytesCount += bytesRead

        // For debugging.
        // fmt.Printf("bytesRead = %v, jsonBytesCount = %v\n", bytesRead, jsonBytesCount)

        if err != nil {
            // If we get an EOF error reading the body, break out of the loop.  This is the normal
            // indication that we have read the entire response body.
            if err.Error() == "EOF" {
                break
            }

            // All other errors are unexpected.
            msg := fmt.Sprintf("Error: Error reading AI response body: %s", err)
            fmt.Println(msg)
            return nil, 0, msg
        }

        if int64(bytesRead) >= (contentLength - 100) {
            // We're out of space to hold the rest of the response, which means the JSON in the
            // response won't un-marshal correctly.
            msg := fmt.Sprintf("Error: AI response too large to process: bytesRead = %v", bytesRead)
            fmt.Println(msg)
            return nil, 0, msg
        }

        // Advance slice jsonBytesForReading to one byte past the bytes read so far.
        jsonBytesForReading = jsonBytes[jsonBytesCount:]
    }

    // For debugging.
    // fmt.Printf("Got %v bytes of JSON.\n", jsonBytesCount)

    return jsonBytes, jsonBytesCount, ""
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
