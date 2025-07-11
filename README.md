# disbot

A Discord bot that responds by using an AI to generate its replies.

Usage: `disbot [ --think ] [ --search ]`

- `--think`: Use Claude's extended thinking when generating responses.
- `--search`: Use Claude's Web search tool when generating responses.

You must set environment variables `DISCORD_BOT_TOKEN` and `ANTHROPIC_API_KEY` before launching the
bot, as follows:

```
$ export ANTHROPIC_API_KEY="..."
$ export DISCORD_BOT_TOKEN="..."
$ ./disbot
```

Once the bot is connected to a Discord server, it will respond to `!help` and `!status` commands on
Discord.  Any other message starting with `!` are sent to the AI (with the `!` removed) to get a
response.
