package main

import (
    "container/list"
    "fmt"
    "sync"
)

var (
    // This map stores per-conversation message history. The key is a conversation key
    // (produced by conversationKey()), and the value is a List of maps of the form:
    //
    //  {{ "role": "user",      "content": "..." }
    //   { "role": "assistant", "content": "..." }
    //   { "role": "user",      "content": "..." }
    //   { "role": "assistant", "content": "..." }
    //   ...
    //  }
    //
    // where the "role" alternates between "user" and "assistant", and the "content" is the
    // text of the message. The newest element of each list is List.Front(), and the oldest
    // is List.Back().
    recentMessages = make(map[string]*List, 10)

    // Mutex protecting recentMessages from concurrent access.
    recentMessagesMu sync.Mutex
)

// conversationKey returns the stable key under which a conversation's history is
// stored. DMs are keyed by user ID; channels by channel ID. IDs are used instead
// of display names because names can change or collide.
func conversationKey(channelID string, userID string, isDM bool) string {
    if isDM {
        return "dm:" + userID
    }
    return "ch:" + channelID
}

// This function appends a new message (from either the user or the AI) to the conversation
// history identified by convKey. Parameter role is either "user" or "assistant".
func historySaveNewMessage(role string, message string, convKey string) {
    recentMessagesMu.Lock()
    defer recentMessagesMu.Unlock()

    messageList := recentMessages[convKey]

    if messageList == nil {
        // Initialize a new list for this conversation.
        messageList = list.New()

        // Add messageList to the recentMessages map using the conversation key.
        recentMessages[convKey] = messageList
    }

    // Add the new message to the front of the list.
    messageList.PushFront(map[string]string{"role": role, "content": message})

    // If the length of messageList equals or exceeds maxRecentMessages + 2, remove the
    // 2 oldest elements. We remove the 2 oldest to maintain the invariant that the list
    // always contains pairs of "user" and "assistant" elements, which alternate in the
    // list. We use 'maxRecentMessages + 2' so that after removing the 2 oldest messages,
    // there are maxRecentMessages remaining, so the AI will see all of them on the next
    // user query.
    if messageList.Len() >= maxRecentMessages+2 {
        messageList.Remove(messageList.Back())
        messageList.Remove(messageList.Back())
    }

    // For debugging.
    fmt.Printf("historySaveNewMessage: recentMessages[\"%s\"].Len() = %v\n", convKey, messageList.Len())
}

// This function converts the conversation history identified by convKey (a List of maps)
// into a slice of maps so that it can be marshaled with json.Marshal.
func historyAsSlice(convKey string) ([]map[string]string, error) {
    recentMessagesMu.Lock()
    defer recentMessagesMu.Unlock()

    messageList := recentMessages[convKey]

    // If there is no history for this conversation yet, return an empty slice.
    if messageList == nil {
        return []map[string]string{}, nil
    }

    // This will hold the slice of messages to be sent in the JSON request.
    messagesSlice := make([]map[string]string, 0, messageList.Len())

    // Iterate over the elements of the list from oldest to newest and append each
    // map to messagesSlice.
    for element := messageList.Back(); element != nil; element = element.Prev() {
        // Get the map from the list element.
        messageMap, ok := element.Value.(map[string]string)

        if !ok {
            // This should never happen, because we only push instances of
            // map[string]string into the list.
            msg := fmt.Sprintf("Error: historyAsSlice: Failed to get map from "+
                "recentMessages[\"%s\"]!", convKey)
            fmt.Println(msg)
            return nil, fmt.Errorf("%s", msg)
        }

        // Append messageMap to messagesSlice.
        messagesSlice = append(messagesSlice, messageMap)
    }

    return messagesSlice, nil
}
