# woffuk-cli

Automatic clock in/out for [Woffu](https://app.woffu.com). Install, run the wizard, done. It handles holidays, absences, and telework days for you.

## Install

### Option A: Go install (requires Go 1.24+)

```bash
go install github.com/ngavilan-dogfy/woffuk-cli@latest
```

### Option B: Download binary

Go to [Releases](https://github.com/ngavilan-dogfy/woffuk-cli/releases) and download the binary for your OS.

### Option C: Build from source

```bash
git clone https://github.com/ngavilan-dogfy/woffuk-cli.git
cd woffuk-cli
go build -o woffuk .
mv woffuk /usr/local/bin/  # or anywhere in your PATH
```

### Prerequisites

- [gh](https://cli.github.com/) CLI — needed for auto-sign setup. Install:
  ```bash
  # macOS
  brew install gh

  # Linux
  sudo apt install gh   # Debian/Ubuntu
  sudo dnf install gh   # Fedora

  # Windows
  winget install GitHub.cli
  ```
  Then authenticate: `gh auth login`

## Setup (one-time)

```bash
woffuk setup
```

The wizard walks you through everything:

```
=== woffuk setup ===

Woffu email: you@company.com
Woffu password: ••••••••
Company name: yourcompany
  -> https://yourcompany.woffu.com

=== Office location ===

Where is your office?: Passeig Zona Franca 28 Barcelona
  Searching...

  1) Passeig de la Zona Franca, Sants-Montjuic, Barcelona
  2) Passeig de la Zona Franca, la Marina del Prat Vermell, Barcelona
  0) None of these — search again

  Pick a number: 1
  Coordinates: 41.361979, 2.137788

=== Home location ===

Where is your home?: Carrer Vistula 12 Segur de Calafell
  Searching...

  1) Carrer del Vistula, Segur de Calafell, Calafell, Tarragona
  2) Carrer del Vistula, Segur de Calafell Platja, Calafell
  0) None of these — search again

  Pick a number: 1
  Coordinates: 41.190452, 1.597968

=== Auto-sign schedule ===

Use default schedule? [Y/n]: y

=== Telegram notifications (optional) ===

Telegram Bot Token (Enter to skip):
  Skipped

Fork repo and configure GitHub Actions? [Y/n]: y
  Forking repo...
  Fork: yourusername/woffuk-cli
  Secrets configured
  GitHub Actions enabled

Setup complete! Auto-signing is active.
```

After setup, your Woffu account will be clocked in/out automatically every workday. No further action needed.

## Commands

| Command | What it does |
|---|---|
| `woffuk` | Interactive TUI dashboard |
| `woffuk status` | Today's date, mode (office/remote), working day |
| `woffuk events` | Remaining vacations, hours, personal days |
| `woffuk sign` | Clock in/out right now |
| `woffuk schedule` | View current auto-sign times |
| `woffuk schedule edit` | Change sign times + push to GitHub |
| `woffuk sync` | Re-sync secrets and workflows to your fork |
| `woffuk setup` | Re-run the full setup wizard |

## Auto-signing

After setup, GitHub Actions handles signing automatically:

| Day | Default times (CET) |
|---|---|
| Mon — Thu | 08:30, 13:30, 14:15, 17:30 |
| Fri | 08:00, 15:00 |

Each run adds a random 2-5 min delay so it doesn't sign at the exact same second every day.

To change times: `woffuk schedule edit`. It updates your config and pushes the new workflow to GitHub.

You can also trigger a manual sign from the GitHub Actions tab.

## How it works

1. Authenticates with Woffu
2. Fetches your calendar (holidays, absences, telework)
3. Determines if today is a working day
4. Detects telework (approved **or pending**) and picks coordinates accordingly
5. Signs with the correct GPS coordinates
6. Sends a Telegram notification (if configured)

## Telegram notifications (optional)

Get a message every time you clock in:

```
Fichaje realizado correctamente
2026-03-17
Teletrabajo
```

To set up:
1. Create a bot with [@BotFather](https://t.me/BotFather) on Telegram
2. Get your chat ID from [@userinfobot](https://t.me/userinfobot)
3. Add both during `woffuk setup`, or edit `~/.woffuk.yaml`:
   ```yaml
   telegram:
     bot_token: "123456:ABC-DEF..."
     chat_id: "987654321"
   ```
4. Run `woffuk sync` to push the secrets to GitHub

## Configuration

All config lives in `~/.woffuk.yaml`. Password is stored in your OS keychain (macOS Keychain / Linux keyring / Windows Credential Manager), never in plain text.

In GitHub Actions, config comes from repository secrets — these are set automatically by `woffuk setup`.

### Environment variables (CI)

| Variable | Description |
|---|---|
| `WOFFU_URL` | `https://app.woffu.com/api` (don't change) |
| `WOFFU_COMPANY_URL` | `https://yourcompany.woffu.com` |
| `WOFFU_EMAIL` | Your Woffu login email |
| `WOFFU_PASSWORD` | Your Woffu password |
| `WOFFU_LATITUDE` | Office GPS latitude |
| `WOFFU_LONGITUDE` | Office GPS longitude |
| `WOFFU_HOME_LATITUDE` | Home GPS latitude |
| `WOFFU_HOME_LONGITUDE` | Home GPS longitude |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token (optional) |
| `TELEGRAM_CHAT_ID` | Telegram chat ID (optional) |

## License

MIT
