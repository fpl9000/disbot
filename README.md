# disbot

A Discord bot that responds by using an AI to generate its replies.

Usage: `disbot [ --think ] [ --search ]`

- `--think`: Use Claude's extended thinking when generating responses.
- `--search`: Use Claude's Web search tool when generating responses.

Note that `--search` will result in many thousands of additional tokens being generated, which will
increase the cost of the API calls.

You must set environment variables `DISCORD_BOT_TOKEN` and `ANTHROPIC_API_KEY` before launching the
bot, as follows:

```
$ export DISCORD_BOT_TOKEN="..."
$ export ANTHROPIC_API_KEY="..."
$ ./disbot
```

Once the bot is connected to a Discord server, it will respond to `!help` and `!status` commands on
Discord.  Any other message starting with `!` are sent to the AI (with the `!` removed) to get a
response.
