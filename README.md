# woffuk

Automatic clock in/out for [Woffu](https://app.woffu.com). Install it, run the setup, and forget about it.

## Install

### One-liner (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/ngavilan-dogfy/woffuk-cli/main/install.sh | sh
```

This downloads the latest binary for your system and installs it to `/usr/local/bin`.

### Download manually

Go to [Releases](https://github.com/ngavilan-dogfy/woffuk-cli/releases), download the binary for your platform, and move it to your PATH:

| Platform | Binary |
|---|---|
| macOS Apple Silicon (M1/M2/M3/M4) | `woffuk-darwin-arm64` |
| macOS Intel | `woffuk-darwin-amd64` |
| Linux x64 | `woffuk-linux-amd64` |
| Linux ARM | `woffuk-linux-arm64` |

```bash
chmod +x woffuk-darwin-arm64
sudo mv woffuk-darwin-arm64 /usr/local/bin/woffuk
```

### From source (requires Go 1.24+)

```bash
go install github.com/ngavilan-dogfy/woffuk-cli@latest
```

## Prerequisites

Before running `woffuk setup`, you need:

1. **A Woffu account** — your company email and password
2. **GitHub CLI** — for auto-signing via GitHub Actions

Install `gh`:

```bash
# macOS
brew install gh

# Ubuntu / Debian
sudo apt install gh

# Fedora
sudo dnf install gh

# Or see https://cli.github.com
```

Then login:

```bash
gh auth login
```

## Setup

```bash
woffuk setup
```

That's it. The wizard handles everything:

```
woffuk setup

┃ Login to Woffu
┃ Email: ngavilan@dogfydiet.com
┃ Password: ••••••••

◯ Connecting to Woffu...

✓ Logged in as NAHUEL GAVILAN BERNAL
→ Dogfy Diet — IT, Senior Platform Engineer
→ Office: Oficinas Landmark

┃ Home location
┃ > Open map / Search by text

  (browser opens with interactive map — click to pick your location)

✓ 41.190452, 1.597968

┃ Use default schedule? Yes
┃ Enable Telegram? Skip
┃ Set up GitHub Actions? Yes

◯ Setting up GitHub...

✓ Fork: yourusername/woffuk-cli
✓ Secrets + workflows configured

All set!
  Mon-Thu: 08:30, 13:30, 14:15, 17:30
  Fri: 08:00, 15:00
```

What happens behind the scenes:

1. Logs into Woffu and detects your office, department, and role
2. Resolves office coordinates from Woffu (or geocodes the office name)
3. Opens an interactive map in your browser to pick your home location
4. Forks this repo to your GitHub account
5. Configures all secrets and enables GitHub Actions

After setup, Woffu will be clocked in/out automatically every workday. No further action needed.

## Commands

| Command | What it does |
|---|---|
| `woffuk` | Interactive TUI dashboard |
| `woffuk status` | Today's date, mode (office/remote), working day |
| `woffuk events` | Remaining vacations, hours, personal days |
| `woffuk sign` | Clock in/out right now |
| `woffuk schedule` | View current auto-sign times |
| `woffuk schedule edit` | Change sign times and push to GitHub |
| `woffuk sync` | Re-sync secrets and workflows to your fork |
| `woffuk setup` | Re-run the full setup wizard |

## Auto-signing

GitHub Actions runs the signing on schedule:

| Day | Default times (CET) |
|---|---|
| Mon — Thu | 08:30, 13:30, 14:15, 17:30 |
| Fri | 08:00, 15:00 |

Each run adds a random 2–5 min delay so it doesn't sign at the exact same second every day.

Change times with `woffuk schedule edit` — it updates your config and pushes new workflows to GitHub.

You can also trigger a manual sign from the GitHub **Actions** tab at any time.

## How it works

1. Authenticates with Woffu
2. Fetches your calendar — holidays, absences, telework
3. Detects telework (approved **or pending**) and picks home/office coordinates
4. Signs with the correct GPS coordinates
5. Sends a Telegram notification (if configured)

## Telegram notifications (optional)

Get a message on every sign:

```
✅ Fichaje realizado correctamente
📅 2026-03-17
🏠 Teletrabajo
```

To set up:

1. Create a bot with [@BotFather](https://t.me/BotFather) on Telegram
2. Get your chat ID from [@userinfobot](https://t.me/userinfobot)
3. Run `woffuk setup` and enter the token + chat ID when prompted
4. Or edit `~/.woffuk.yaml` manually and run `woffuk sync`

## Configuration

| What | Where |
|---|---|
| Config | `~/.woffuk.yaml` |
| Password | OS keychain (macOS Keychain / Linux keyring / Windows Credential Manager) |
| GitHub secrets | Set automatically by `woffuk setup` |

### Environment variables (CI / GitHub Actions)

| Variable | Description |
|---|---|
| `WOFFU_URL` | `https://app.woffu.com/api` |
| `WOFFU_COMPANY_URL` | `https://yourcompany.woffu.com` |
| `WOFFU_EMAIL` | Woffu login email |
| `WOFFU_PASSWORD` | Woffu password |
| `WOFFU_LATITUDE` | Office latitude |
| `WOFFU_LONGITUDE` | Office longitude |
| `WOFFU_HOME_LATITUDE` | Home latitude |
| `WOFFU_HOME_LONGITUDE` | Home longitude |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token (optional) |
| `TELEGRAM_CHAT_ID` | Telegram chat ID (optional) |

## License

MIT
