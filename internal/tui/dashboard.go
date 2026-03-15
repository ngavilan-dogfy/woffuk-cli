package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/config"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/notify"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/woffu"
)

// Messages
type dataMsg struct {
	signInfo *woffu.SignInfo
	events   []woffu.AvailableUserEvent
	profile  *woffu.UserProfile
}
type errMsg struct{ err error }
type signDoneMsg struct{}
type flashMsg struct{ text string; isErr bool }
type clearFlashMsg struct{}
type tickMsg time.Time

// Dashboard is the main TUI model.
type Dashboard struct {
	client        *woffu.Client
	companyClient *woffu.Client
	cfg           *config.Config
	password      string

	// State
	loading  bool
	token    string
	signInfo *woffu.SignInfo
	events   []woffu.AvailableUserEvent
	profile  *woffu.UserProfile
	err      error
	signing  bool

	// Flash
	flash    string
	flashErr bool

	// Layout
	width  int
	height int
}

func NewDashboard(client, companyClient *woffu.Client, cfg *config.Config, password string) *Dashboard {
	return &Dashboard{
		client:        client,
		companyClient: companyClient,
		cfg:           cfg,
		password:      password,
		loading:       true,
	}
}

func (d *Dashboard) Init() tea.Cmd {
	return tea.Batch(d.fetchData(), d.tick())
}

func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case tea.KeyMsg:
		return d.handleKey(msg)

	case dataMsg:
		d.loading = false
		d.signInfo = msg.signInfo
		d.events = msg.events
		d.profile = msg.profile

	case signDoneMsg:
		d.signing = false
		d.flash = "Signed successfully!"
		d.flashErr = false
		return d, tea.Batch(d.fetchData(), d.clearFlashAfter(3*time.Second))

	case errMsg:
		d.loading = false
		d.signing = false
		d.flash = msg.err.Error()
		d.flashErr = true
		return d, d.clearFlashAfter(5*time.Second)

	case clearFlashMsg:
		d.flash = ""

	case tickMsg:
		return d, d.tick()
	}

	return d, nil
}

func (d *Dashboard) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return d, tea.Quit

	case "s":
		if !d.signing && d.signInfo != nil && d.signInfo.IsWorkingDay {
			d.signing = true
			d.flash = "Signing..."
			d.flashErr = false
			return d, d.doSign()
		}
		if d.signInfo != nil && !d.signInfo.IsWorkingDay {
			d.flash = "Not a working day"
			d.flashErr = true
			return d, d.clearFlashAfter(3*time.Second)
		}

	case "r":
		d.loading = true
		d.flash = ""
		return d, d.fetchData()

	case "o":
		// Open Woffu in browser
		return d, d.openWoffu()
	}

	return d, nil
}

func (d *Dashboard) View() string {
	if d.width == 0 {
		return ""
	}

	var sections []string

	// Header
	header := d.renderHeader()
	sections = append(sections, header)

	if d.loading {
		sections = append(sections, "\n"+sDimmed.Render("  Loading..."))
	} else if d.err != nil {
		sections = append(sections, "\n"+sDanger.Render(fmt.Sprintf("  Error: %s", d.err)))
	} else {
		// Status panel
		sections = append(sections, d.renderStatus())

		// Events panel
		if evts := d.renderEvents(); evts != "" {
			sections = append(sections, evts)
		}

		// Next events
		if next := d.renderNextEvents(); next != "" {
			sections = append(sections, next)
		}

		// Schedule
		sections = append(sections, d.renderSchedule())
	}

	// Flash message
	if d.flash != "" {
		icon := sFlashSuccess.Render("  ✓ ")
		if d.flashErr {
			icon = sFlashError.Render("  ✗ ")
		}
		sections = append(sections, "\n"+icon+d.flash)
	}

	// Help bar
	sections = append(sections, d.renderHelp())

	return strings.Join(sections, "\n")
}

func (d *Dashboard) renderHeader() string {
	name := "woffuk"
	if d.profile != nil {
		name = fmt.Sprintf("woffuk — %s", d.profile.FullName)
	}

	left := sTitle.Render(name)
	right := sDimmed.Render(time.Now().Format("15:04"))

	gap := d.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}

	return lipgloss.NewStyle().
		Background(colorBarBg).
		Width(d.width).
		Render(left + strings.Repeat(" ", gap) + right)
}

func (d *Dashboard) renderStatus() string {
	info := d.signInfo
	if info == nil {
		return ""
	}

	workingDay := sSuccess.Render("yes")
	if !info.IsWorkingDay {
		workingDay = sDanger.Render("no")
	}

	mode := sSignOut.Render(fmt.Sprintf("%s %s", info.Mode.Emoji(), info.Mode.Label()))
	if info.Mode == woffu.SignModeRemote {
		mode = sSignIn.Render(fmt.Sprintf("%s %s", info.Mode.Emoji(), info.Mode.Label()))
	}

	rows := []string{
		sLabel.Render("Date") + sValue.Render(info.Date),
		sLabel.Render("Working day") + workingDay,
		sLabel.Render("Mode") + mode,
	}

	if info.IsWorkingDay {
		rows = append(rows,
			sLabel.Render("Coordinates")+sDimmed.Render(fmt.Sprintf("%.4f, %.4f", info.Latitude, info.Longitude)))
	}

	return "\n" + sBox.Render(strings.Join(rows, "\n"))
}

func (d *Dashboard) renderEvents() string {
	if len(d.events) == 0 {
		return ""
	}

	var rows []string
	for _, e := range d.events {
		name := lipgloss.NewStyle().Foreground(colorMuted).Width(40).Render(e.Name)
		val := sValue.Render(fmt.Sprintf("%6.0f %s", e.Available, e.Unit))
		rows = append(rows, "  "+name+val)
	}

	return "\n" + sSubtitle.Render("  Available events") + "\n" + strings.Join(rows, "\n")
}

func (d *Dashboard) renderNextEvents() string {
	if d.signInfo == nil || len(d.signInfo.NextEvents) == 0 {
		return ""
	}

	var rows []string
	limit := 6
	for i, e := range d.signInfo.NextEvents {
		if i >= limit {
			rows = append(rows, sDimmed.Render(fmt.Sprintf("    ... +%d more", len(d.signInfo.NextEvents)-limit)))
			break
		}
		name := ""
		if len(e.Names) > 0 {
			name = " " + e.Names[0]
		}
		rows = append(rows, "  "+sDimmed.Render("  "+e.Date)+name)
	}

	return "\n" + sSubtitle.Render("  Upcoming") + "\n" + strings.Join(rows, "\n")
}

func (d *Dashboard) renderSchedule() string {
	s := d.cfg.Schedule
	days := []struct {
		name string
		day  config.DaySchedule
	}{
		{"Mon", s.Monday}, {"Tue", s.Tuesday}, {"Wed", s.Wednesday},
		{"Thu", s.Thursday}, {"Fri", s.Friday},
	}

	var parts []string
	for _, dd := range days {
		if !dd.day.Enabled {
			continue
		}
		var times []string
		for i, t := range dd.day.Times {
			if i%2 == 0 {
				times = append(times, sSignIn.Render("▶")+t.Time)
			} else {
				times = append(times, sSignOut.Render("■")+t.Time)
			}
		}
		parts = append(parts, sDimmed.Render("  "+dd.name+" ")+strings.Join(times, " "))
	}

	return "\n" + sSubtitle.Render("  Schedule") + "\n" + strings.Join(parts, "\n")
}

func (d *Dashboard) renderHelp() string {
	hints := []string{
		hint("s", "sign"),
		hint("r", "refresh"),
		hint("o", "open woffu"),
		hint("q", "quit"),
	}

	if d.signInfo != nil && !d.signInfo.IsWorkingDay {
		hints = hints[1:] // Remove sign hint
	}

	return "\n" + sDimmed.Render("  ") + strings.Join(hints, "  ")
}

// Commands

func (d *Dashboard) fetchData() tea.Cmd {
	return func() tea.Msg {
		token, err := woffu.Authenticate(d.client, d.companyClient, d.cfg.WoffuEmail, d.password)
		if err != nil {
			return errMsg{err}
		}
		d.token = token

		profile, _ := woffu.GetUserProfile(d.companyClient, token)

		info, err := woffu.GetSignInfo(d.companyClient, token,
			d.cfg.Latitude, d.cfg.Longitude, d.cfg.HomeLatitude, d.cfg.HomeLongitude)
		if err != nil {
			return errMsg{err}
		}

		events, err := woffu.GetAvailableEvents(d.companyClient, token)
		if err != nil {
			return errMsg{err}
		}

		return dataMsg{signInfo: info, events: events, profile: profile}
	}
}

func (d *Dashboard) doSign() tea.Cmd {
	return func() tea.Msg {
		err := woffu.DoSign(d.companyClient, d.token, d.signInfo.Latitude, d.signInfo.Longitude)
		if err != nil {
			return errMsg{err}
		}

		telegramCfg := notify.TelegramConfig{
			BotToken: d.cfg.Telegram.BotToken,
			ChatID:   d.cfg.Telegram.ChatID,
		}
		_ = notify.SendSignedNotification(telegramCfg, d.signInfo)

		return signDoneMsg{}
	}
}

func (d *Dashboard) openWoffu() tea.Cmd {
	return func() tea.Msg {
		url := d.cfg.WoffuCompanyURL + "/v2"
		openBrowserCmd(url)
		return nil
	}
}

func (d *Dashboard) tick() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (d *Dashboard) clearFlashAfter(dur time.Duration) tea.Cmd {
	return tea.Tick(dur, func(t time.Time) tea.Msg {
		return clearFlashMsg{}
	})
}

func openBrowserCmd(url string) {
	exec.Command("open", url).Start()
}

