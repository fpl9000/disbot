package parley

import "testing"

// testPolicy is a small, fully specified policy used across the tests.
func testPolicy() *Policy {
	return &Policy{
		BotUserID:       "bot",
		BotName:         "Claude",
		OwnerID:         "fran",
		AllowedGuilds:   map[string]bool{"g1": true},
		AllowedChannels: map[string]bool{"general": true, "lounge": true},
		AmbientChannels: map[string]bool{"lounge": true},
		AllowedBots:     map[string]bool{"friendlybot": true},
		CommandPrefix:   "!parley",
	}
}

func TestClassify(t *testing.T) {
	p := testPolicy()
	base := Message{GuildID: "g1", ChannelID: "general", AuthorID: "sam", AuthorName: "Sam"}
	with := func(f func(*Message)) Message { m := base; f(&m); return m }

	cases := []struct {
		name string
		m    Message
		want Decision
	}{
		{"plain chatter", with(func(m *Message) { m.Content = "lunch?" }), RecordOnly},
		{"mention", with(func(m *Message) { m.MentionsBot = true }), Direct},
		{"reply to bot", with(func(m *Message) { m.RepliesToBot = true }), Direct},
		{"name as a word", with(func(m *Message) { m.Content = "what do you think, claude?" }), Direct},
		{"name inside a word", with(func(m *Message) { m.Content = "Claudette says hi" }), RecordOnly},
		{"own message", with(func(m *Message) { m.AuthorID = "bot"; m.MentionsBot = true }), RecordOnly},
		{"other bot", with(func(m *Message) { m.AuthorID = "spam"; m.AuthorIsBot = true; m.MentionsBot = true }), Ignore},
		{"allowed bot", with(func(m *Message) { m.AuthorID = "friendlybot"; m.AuthorIsBot = true; m.MentionsBot = true }), Direct},
		{"wrong guild", with(func(m *Message) { m.GuildID = "g2"; m.MentionsBot = true }), Ignore},
		{"wrong channel", with(func(m *Message) { m.ChannelID = "secret"; m.MentionsBot = true }), Ignore},
		{"thread in allowed channel", with(func(m *Message) { m.ChannelID = "t1"; m.ParentID = "general"; m.MentionsBot = true }), Direct},
		{"ambient channel chatter", with(func(m *Message) { m.ChannelID = "lounge"; m.Content = "anyone up?" }), Ambient},
		{"ambient thread chatter", with(func(m *Message) { m.ChannelID = "t2"; m.ParentID = "lounge"; m.Content = "hm" }), Ambient},
		{"owner command", with(func(m *Message) { m.AuthorID = "fran"; m.Content = "!parley status" }), OwnerCommand},
		{"command from someone else", with(func(m *Message) { m.Content = "!parley pause" }), RecordOnly},
		{"prefix not a word", with(func(m *Message) { m.AuthorID = "fran"; m.Content = "!parleyx" }), RecordOnly},
		{"owner DM", Message{IsDM: true, ChannelID: "dm1", AuthorID: "fran", Content: "hi"}, Direct},
		{"stranger DM", Message{IsDM: true, ChannelID: "dm2", AuthorID: "sam", Content: "hi"}, Ignore},
	}
	for _, tc := range cases {
		if got := p.Classify(tc.m); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// The owner is identified by ID, so copying the owner's display name grants
// nothing.
func TestOwnerIsCheckedByID(t *testing.T) {
	p := testPolicy()
	m := Message{GuildID: "g1", ChannelID: "general", AuthorID: "impostor", AuthorName: "fran", Content: "!parley pause"}
	if got := p.Classify(m); got == OwnerCommand {
		t.Fatal("display name granted owner rights")
	}
}
