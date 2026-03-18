package github

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ngavilan-dogfy/woffux/internal/config"
)

// CronEntry represents a single cron schedule line with a comment.
type CronEntry struct {
	Cron    string
	Comment string
}

// GenerateCrons converts the schedule config into GitHub Actions cron expressions.
// For DST timezones, generates a single cron per sign time with comma-separated
// UTC hours (e.g., "30 6,7 * * 1-4") covering both standard and DST offsets.
// A timezone guard step at runtime verifies which offset is active.
func GenerateCrons(schedule config.Schedule, tz string) []CronEntry {
	var entries []CronEntry

	stdOff, dstOff := utcOffsets(tz)
	isDST := stdOff != dstOff

	// Group days by their schedule to produce compact cron expressions
	type dayTimes struct {
		days  []int
		names []string
		times []config.ScheduleEntry
	}

	allDays := []struct {
		day  config.DaySchedule
		num  int
		name string
	}{
		{schedule.Monday, 1, "Mon"},
		{schedule.Tuesday, 2, "Tue"},
		{schedule.Wednesday, 3, "Wed"},
		{schedule.Thursday, 4, "Thu"},
		{schedule.Friday, 5, "Fri"},
	}

	// Group days with identical schedules
	groups := make(map[string]*dayTimes)
	for _, d := range allDays {
		if !d.day.Enabled {
			continue
		}
		key := timesKey(d.day.Times)
		if g, ok := groups[key]; ok {
			g.days = append(g.days, d.num)
			g.names = append(g.names, d.name)
		} else {
			groups[key] = &dayTimes{
				days:  []int{d.num},
				names: []string{d.name},
				times: d.day.Times,
			}
		}
	}

	// Sort groups by key for deterministic output
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		g := groups[key]
		daysStr := intSliceJoin(g.days, ",")
		namesStr := strings.Join(g.names, "-")

		for _, t := range g.times {
			hour, minute := parseTime(t.Time)

			utcHourStd := localToUTC(hour, stdOff)
			if isDST {
				utcHourDST := localToUTC(hour, dstOff)
				if utcHourStd == utcHourDST {
					cron := fmt.Sprintf("%d %d * * %s", minute, utcHourStd, daysStr)
					comment := fmt.Sprintf("%s %s (UTC%+d)", namesStr, t.Time, stdOff)
					entries = append(entries, CronEntry{Cron: cron, Comment: comment})
				} else {
					h1, h2 := utcHourStd, utcHourDST
					if h1 > h2 {
						h1, h2 = h2, h1
					}
					cron := fmt.Sprintf("%d %d,%d * * %s", minute, h1, h2, daysStr)
					comment := fmt.Sprintf("%s %s (UTC%+d/UTC%+d)", namesStr, t.Time, stdOff, dstOff)
					entries = append(entries, CronEntry{Cron: cron, Comment: comment})
				}
			} else {
				cron := fmt.Sprintf("%d %d * * %s", minute, utcHourStd, daysStr)
				comment := fmt.Sprintf("%s %s (UTC%+d)", namesStr, t.Time, stdOff)
				entries = append(entries, CronEntry{Cron: cron, Comment: comment})
			}
		}
	}

	return entries
}

// localToUTC converts a local hour to UTC given an offset in hours.
func localToUTC(hour, offsetHours int) int {
	utc := hour - offsetHours
	if utc < 0 {
		utc += 24
	} else if utc >= 24 {
		utc -= 24
	}
	return utc
}

// signTimes collects all unique sign times (HH:MM) from the schedule config.
func signTimes(schedule config.Schedule) []string {
	seen := make(map[string]bool)
	for _, day := range []config.DaySchedule{
		schedule.Monday, schedule.Tuesday, schedule.Wednesday,
		schedule.Thursday, schedule.Friday,
	} {
		if !day.Enabled {
			continue
		}
		for _, t := range day.Times {
			seen[t.Time] = true
		}
	}
	var times []string
	for t := range seen {
		times = append(times, t)
	}
	return times
}

// GenerateWorkflowYAML generates the auto-sign GitHub Actions workflow.
// For DST zones, each cron covers both UTC offsets via comma-separated hours.
// A timezone guard verifies the exact local HH:MM at runtime to prevent
// double signing during DST transitions.
func GenerateWorkflowYAML(schedule config.Schedule, tz string, opts ...int) string {
	// Optional random delay in seconds (default 90)
	randomDelay := 90
	if len(opts) > 0 && opts[0] > 0 {
		randomDelay = opts[0]
	}
	crons := GenerateCrons(schedule, tz)
	isDST := hasDST(tz)

	var cronLines []string
	for _, c := range crons {
		cronLines = append(cronLines, fmt.Sprintf("    - cron: '%s'  # %s", c.Cron, c.Comment))
	}

	// Resolve IANA timezone for the guard step
	ianaZone := tz
	if alias, ok := tzAliases[strings.ToUpper(tz)]; ok {
		ianaZone = alias
	}

	// Build the timezone guard step (only for DST zones)
	guardYAML := ""
	guardCondition := ""
	if isDST {
		times := signTimes(schedule)
		timesStr := strings.Join(times, " ")

		guardYAML = fmt.Sprintf(`
      - name: Timezone guard
        id: tz
        run: |
          CRON_MIN=$(echo "%s" | awk '{print $1}')
          CRON_HOUR=$(echo "%s" | awk '{print $2}')
          OFFSET=$(TZ=%s date +%%z | sed 's/00$//;s/^+0/+/;s/^+//')
          LOCAL_HOUR=$(( CRON_HOUR + OFFSET ))
          if [ "$LOCAL_HOUR" -lt 0 ]; then LOCAL_HOUR=$(( LOCAL_HOUR + 24 )); fi
          if [ "$LOCAL_HOUR" -ge 24 ]; then LOCAL_HOUR=$(( LOCAL_HOUR - 24 )); fi
          LOCAL_TIME=$(printf "%%02d:%%02d" "$LOCAL_HOUR" "$CRON_MIN")
          SIGN_TIMES="%s"
          MATCH=false
          for t in $SIGN_TIMES; do
            if [ "$LOCAL_TIME" = "$t" ]; then MATCH=true; break; fi
          done
          if [ "$MATCH" = "false" ]; then
            echo "Skipping: cron $CRON_HOUR:$CRON_MIN + offset $OFFSET = local $LOCAL_TIME, not a configured sign time"
            echo "skip=true" >> "$GITHUB_OUTPUT"
          else
            echo "skip=false" >> "$GITHUB_OUTPUT"
          fi
`, "${{ github.event.schedule }}", "${{ github.event.schedule }}", ianaZone, timesStr)
		guardCondition = "\n        if: steps.tz.outputs.skip != 'true'"
	}

	// Failure notification condition: always on failure, but skip if guard skipped
	failureCondition := "\n        if: failure()"
	if isDST {
		failureCondition = "\n        if: failure() && steps.tz.outputs.skip != 'true'"
	}

	return fmt.Sprintf(`name: Auto Sign

on:
  schedule:
%s

concurrency:
  group: sign
  cancel-in-progress: true

jobs:
  sign:
    name: Sign in Woffu
    runs-on: ubuntu-latest
    steps:%s

      - name: Download woffux%s
        run: |
          curl -fsSL "https://github.com/ngavilan-dogfy/woffux/releases/latest/download/woffux-linux-amd64" -o woffux
          chmod +x woffux

      - name: Random delay%s
        run: sleep $(( RANDOM %% %d + 1 ))

      - name: Sign%s
        run: ./woffux sign
        env:
          WOFFU_URL: ${{ secrets.WOFFU_URL }}
          WOFFU_COMPANY_URL: ${{ secrets.WOFFU_COMPANY_URL }}
          WOFFU_EMAIL: ${{ secrets.WOFFU_EMAIL }}
          WOFFU_PASSWORD: ${{ secrets.WOFFU_PASSWORD }}
          WOFFU_LATITUDE: ${{ secrets.WOFFU_LATITUDE }}
          WOFFU_LONGITUDE: ${{ secrets.WOFFU_LONGITUDE }}
          WOFFU_HOME_LATITUDE: ${{ secrets.WOFFU_HOME_LATITUDE }}
          WOFFU_HOME_LONGITUDE: ${{ secrets.WOFFU_HOME_LONGITUDE }}
          TELEGRAM_BOT_TOKEN: ${{ secrets.TELEGRAM_BOT_TOKEN }}
          TELEGRAM_CHAT_ID: ${{ secrets.TELEGRAM_CHAT_ID }}

      - name: Notify failure%s
        run: |
          if [ -n "$TELEGRAM_BOT_TOKEN" ] && [ -n "$TELEGRAM_CHAT_ID" ]; then
            MSG="❌ woffux auto-sign failed at $(TZ=%s date '+%%H:%%M %%Z %%Y-%%m-%%d')"
            curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" \
              -d chat_id="${TELEGRAM_CHAT_ID}" -d text="${MSG}" > /dev/null
          fi
        env:
          TELEGRAM_BOT_TOKEN: ${{ secrets.TELEGRAM_BOT_TOKEN }}
          TELEGRAM_CHAT_ID: ${{ secrets.TELEGRAM_CHAT_ID }}
`, strings.Join(cronLines, "\n"), guardYAML, guardCondition, guardCondition, randomDelay, guardCondition, failureCondition, ianaZone)
}

// GenerateManualWorkflowYAML generates the manual sign workflow.
func GenerateManualWorkflowYAML() string {
	return `name: Manual Sign

on:
  workflow_dispatch:

concurrency:
  group: sign
  cancel-in-progress: true

jobs:
  sign:
    name: Sign in Woffu
    runs-on: ubuntu-latest
    steps:
      - name: Download woffux
        run: |
          curl -fsSL "https://github.com/ngavilan-dogfy/woffux/releases/latest/download/woffux-linux-amd64" -o woffux
          chmod +x woffux

      - name: Sign
        run: ./woffux sign
        env:
          WOFFU_URL: ${{ secrets.WOFFU_URL }}
          WOFFU_COMPANY_URL: ${{ secrets.WOFFU_COMPANY_URL }}
          WOFFU_EMAIL: ${{ secrets.WOFFU_EMAIL }}
          WOFFU_PASSWORD: ${{ secrets.WOFFU_PASSWORD }}
          WOFFU_LATITUDE: ${{ secrets.WOFFU_LATITUDE }}
          WOFFU_LONGITUDE: ${{ secrets.WOFFU_LONGITUDE }}
          WOFFU_HOME_LATITUDE: ${{ secrets.WOFFU_HOME_LATITUDE }}
          WOFFU_HOME_LONGITUDE: ${{ secrets.WOFFU_HOME_LONGITUDE }}
          TELEGRAM_BOT_TOKEN: ${{ secrets.TELEGRAM_BOT_TOKEN }}
          TELEGRAM_CHAT_ID: ${{ secrets.TELEGRAM_CHAT_ID }}
`
}

// GenerateKeepaliveWorkflowYAML prevents GitHub from auto-disabling scheduled workflows.
func GenerateKeepaliveWorkflowYAML() string {
	return `name: Keepalive

on:
  schedule:
    - cron: '0 12 1 */2 *'

jobs:
  keepalive:
    name: Keep workflows alive
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - name: Keepalive commit
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git commit --allow-empty -m "chore: keepalive [skip ci]"
          git push
`
}

// tzAliases maps common abbreviations to IANA zone names.
var tzAliases = map[string]string{
	"CET":  "Europe/Madrid",
	"CEST": "Europe/Madrid",
	"WET":  "Europe/Lisbon",
	"EET":  "Europe/Athens",
	"GMT":  "Europe/London",
	"EST":  "America/New_York",
	"CST":  "America/Chicago",
	"MST":  "America/Denver",
	"PST":  "America/Los_Angeles",
}

// loadTimezone resolves a timezone string to a *time.Location.
// Accepts IANA names (Europe/Madrid) and common aliases (CET).
func loadTimezone(tz string) *time.Location {
	if tz == "UTC" || tz == "" {
		return time.UTC
	}
	// Try alias first
	if iana, ok := tzAliases[strings.ToUpper(tz)]; ok {
		tz = iana
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		// Fallback: try parsing as numeric offset (+1, -5, etc.)
		if n, err := strconv.Atoi(tz); err == nil {
			return time.FixedZone(fmt.Sprintf("UTC%+d", n), n*3600)
		}
		return time.FixedZone("CET", 3600) // safe default
	}
	return loc
}

// hasDST returns true if the timezone observes DST.
func hasDST(tz string) bool {
	loc := loadTimezone(tz)
	jan := time.Date(2026, time.January, 15, 12, 0, 0, 0, loc)
	jul := time.Date(2026, time.July, 15, 12, 0, 0, 0, loc)
	_, offJan := jan.Zone()
	_, offJul := jul.Zone()
	return offJan != offJul
}

// utcOffsets returns the standard and DST UTC offsets in hours for the timezone.
// For non-DST zones, both values are the same.
func utcOffsets(tz string) (stdOffset, dstOffset int) {
	loc := loadTimezone(tz)
	jan := time.Date(2026, time.January, 15, 12, 0, 0, 0, loc)
	jul := time.Date(2026, time.July, 15, 12, 0, 0, 0, loc)
	_, offJan := jan.Zone()
	_, offJul := jul.Zone()
	return offJan / 3600, offJul / 3600
}

func parseTime(t string) (hour, minute int) {
	parts := strings.Split(t, ":")
	if len(parts) != 2 {
		return 0, 0
	}
	hour, _ = strconv.Atoi(parts[0])
	minute, _ = strconv.Atoi(parts[1])
	return
}

func timesKey(times []config.ScheduleEntry) string {
	var parts []string
	for _, t := range times {
		parts = append(parts, t.Time)
	}
	return strings.Join(parts, ",")
}

func intSliceJoin(ints []int, sep string) string {
	var parts []string
	for _, i := range ints {
		parts = append(parts, strconv.Itoa(i))
	}
	return strings.Join(parts, sep)
}
