# disbot

A Discord bot that responds by using an AI to generate its replies.

Usage:

```
$ export ANTHROPIC_API_KEY="..."
$ export DISCORD_BOT_TOKEN="..."
$ ./disbot
```

You must set environment variables `DISCORD_BOT_TOKEN` and `ANTHROPIC_API_KEY` before launching the bot.

Once the bot is connected to a Discord server, it will respond to `!help` and `!status` commands on
Discord.  Any other message starting with `!` are sent to the AI for a response.
