# Nexus Scrims Bot

Discord bot for organising Nexus Scrims signups and scrim coordination, built on [discord.js](https://discord.js.org).

## Setup

```bash
npm install
```

### Configuration

The bot reads its Discord credentials from a `config.json` file in this folder. **Never commit this file** — it is listed in the root `.gitignore`.

1. Copy the example:

   ```bash
   cp config.example.json config.json
   ```

2. Fill in the three fields:

   | Key | Where to find it |
   |---|---|
   | `token` | [Discord Developer Portal](https://discord.com/developers/applications) → your application → **Bot** → **Reset Token** |
   | `clientId` | Developer Portal → your application → **General Information** → Application ID |
   | `guildId` | Right-click your Discord server icon → **Copy Server ID** (requires Developer Mode enabled in Discord settings) |

> If you prefer not to commit even a sample, you can also supply these values via environment variables and modify `src/index.ts` to read from `process.env` instead of importing `config.json`.

## Scripts

| Script | Purpose |
|---|---|
| `npm run build` | Compile TypeScript into `dist/` |
| `npm run dev` | Recompile on save and hot-restart via `nodemon` |
| `npm start` | Run the compiled bot from `dist/index.js` |
| `npm test` | Run the Jest test suite |
| `node dist/deploy-commands.js` | Register/refresh the bot's slash commands with Discord (run after `npm run build`) |

## Adding a command

1. Add a new `.ts` file under `src/commands/<category>/`.
2. Export a `data` (SlashCommandBuilder) and an async `execute(interaction)` function.
3. Build and run `node dist/deploy-commands.js` to sync the command to your guild.
