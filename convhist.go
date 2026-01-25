package main

import (
    "container/list"
    "fmt"
)

var (
    // This is a map where the key is a string specifying a channel name (without the leading '#')
    // or a user's nick (without the leading '@') and the value is a List that holds the recent
    // messages in that channel or in DMs with that nick, so the AI has the conversation history.
    // Each value is a list of maps of the form:
    //
    //  {{ "role": "user",      "content": "..." }
    //   { "role": "assistant", "content": "..." }
    //   { "role": "user",      "content": "..." }
    //   { "role": "assistant", "content": "..." }
    //   ...
    //  }
    //
    // where the "role" alternates between "user" and "assistant", and the "content" is the text of
    // the message.  The newest element of each list is List.Front(), and the oldest is List.Back().
    recentMessages = make(map[string]*List, 10)
)

// This function appends a new message (from either the user or the AI) to the conversation history
// for the user (nick) or channel (channelName).  Parameter role is either "user" or "assistant".
func historySaveNewMessage(role string, message string, channelName string, nick string) {
    messageList := recentMessages[role]

    if messageList == nil {
        // Initialize a new list for this role.
        messageList = list.New()

        // Add messageList to the recentMessages map using role as the key.
        recentMessages[role] = messageList
    }

    // Add the new message to the front of the list.
    messageList.PushFront(map[string]string{ "role": role, "content": message })

    // If the length of messageList equals or exceeds maxRecentMessages + 2, remove the 2 oldest
    // elements.  We remove the 2 oldest to maintain the invariant that the list always contains
    // pairs of "user" and "assistant" elements, which alternate in the list.  We use
    // 'maxRecentMessages + 2' so that after removing the 2 oldest messages, there are
    // maxRecentMessages remaining, so the AI will see all of them on the next user query.
    if messageList.Len() >= maxRecentMessages + 2 {
        messageList.Remove(messageList.Back())
        messageList.Remove(messageList.Back())
    }

    // For debugging.
    fmt.Printf("historySaveNewMessage: recentMessages[\"%s\"].Len() = %v\n", role, messageList.Len())
}

// This function converts recentMessages[nick/channelName] (a List of maps) into a slice of maps so
// that it can be marshaled with json.Marshal.  If channelName is not the empty string, it is used
// as the key in the recentMessages map, otherwise nick is used as the key.
func historyAsSlice(nick string, channelName string) ([]map[string]string, error) {
    var keyString string

    if channelName != "" {
        keyString = channelName
    } else {
        keyString = nick
    }

    // This will hold the slice of messages to be sent in the JSON request.
    messagesSlice := make([]map[string]string, 0, recentMessages[keyString].Len())

    messageList := recentMessages[keyString]

    // Iterate over the elements of recentMessages, which is a list of maps, and append each map to
    // messagesSlice.
    for element := messageList.Back(); element != nil; element = element.Prev() {
        // Get the map from the list element.
        messageMap, ok := element.Value.(map[string]string)

        if !ok {
            // This should never happen, because we only push instances of map[string]string into
            // list recentMessages[keyString].
            msg := fmt.Sprintf("Error: getRecentMessagesAsSlice: Failed to get map from recentMessages[%s]!",
                               keyString)
            fmt.Println(msg)
            return nil, fmt.Errorf(msg)
        }

        // Append messageMap to messagesSlice.
        messagesSlice = append(messagesSlice, messageMap)
    }

    return messagesSlice, nil
}
