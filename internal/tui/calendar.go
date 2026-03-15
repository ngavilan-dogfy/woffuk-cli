package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/woffu"
)

// calendarGrid renders a visual monthly calendar with colored days.
type calendarGrid struct {
	year      int
	month     time.Month
	days      []woffu.CalendarDay
	cursor    int // day of month (1-31), 0 = none
	selected  map[int]bool
	width     int
}

func newCalendarGrid(year int, month time.Month, days []woffu.CalendarDay) *calendarGrid {
	today := time.Now().Day()
	currentMonth := time.Now().Month()
	cursor := 0
	if month == currentMonth {
		cursor = today
	} else {
		cursor = 1
	}

	return &calendarGrid{
		year:     year,
		month:    month,
		days:     days,
		cursor:   cursor,
		selected: make(map[int]bool),
	}
}

func (c *calendarGrid) daysInMonth() int {
	return time.Date(c.year, c.month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func (c *calendarGrid) firstWeekday() int {
	// Monday = 0, Sunday = 6
	wd := time.Date(c.year, c.month, 1, 0, 0, 0, 0, time.UTC).Weekday()
	if wd == time.Sunday {
		return 6
	}
	return int(wd) - 1
}

func (c *calendarGrid) dayInfo(day int) *woffu.CalendarDay {
	date := fmt.Sprintf("%d-%02d-%02d", c.year, c.month, day)
	for i := range c.days {
		if c.days[i].Date == date {
			return &c.days[i]
		}
	}
	return nil
}

func (c *calendarGrid) toggleSelect(day int) {
	if c.selected[day] {
		delete(c.selected, day)
	} else {
		c.selected[day] = true
	}
}

func (c *calendarGrid) clearSelection() {
	c.selected = make(map[int]bool)
}

func (c *calendarGrid) selectedDates() []string {
	var dates []string
	for d := 1; d <= c.daysInMonth(); d++ {
		if c.selected[d] {
			dates = append(dates, fmt.Sprintf("%d-%02d-%02d", c.year, c.month, d))
		}
	}
	return dates
}

func (c *calendarGrid) moveLeft() {
	if c.cursor > 1 {
		c.cursor--
	}
}

func (c *calendarGrid) moveRight() {
	if c.cursor < c.daysInMonth() {
		c.cursor++
	}
}

func (c *calendarGrid) moveUp() {
	if c.cursor > 7 {
		c.cursor -= 7
	}
}

func (c *calendarGrid) moveDown() {
	if c.cursor+7 <= c.daysInMonth() {
		c.cursor += 7
	}
}

func (c *calendarGrid) prevMonth() {
	if c.month == time.January {
		c.month = time.December
		c.year--
	} else {
		c.month--
	}
	c.cursor = 1
	c.selected = make(map[int]bool)
}

func (c *calendarGrid) nextMonth() {
	if c.month == time.December {
		c.month = time.January
		c.year++
	} else {
		c.month++
	}
	c.cursor = 1
	c.selected = make(map[int]bool)
}

func (c *calendarGrid) render() string {
	var b strings.Builder

	// Month header with navigation
	monthName := c.month.String()
	header := fmt.Sprintf("◀  %s %d  ▶", monthName, c.year)
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(header))
	b.WriteString("\n\n")

	// Day names
	dayNames := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	for _, d := range dayNames {
		b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Width(5).Align(lipgloss.Center).Render(d))
	}
	b.WriteString("\n")

	// Calendar grid
	firstDay := c.firstWeekday()
	totalDays := c.daysInMonth()
	today := time.Now()
	isCurrentMonth := c.month == today.Month() && c.year == today.Year()

	day := 1
	for week := 0; week < 6; week++ {
		if day > totalDays {
			break
		}

		for col := 0; col < 7; col++ {
			if week == 0 && col < firstDay {
				b.WriteString(strings.Repeat(" ", 5))
				continue
			}
			if day > totalDays {
				break
			}

			cell := c.renderDay(day, col, isCurrentMonth && day == today.Day())
			b.WriteString(cell)
			day++
		}
		b.WriteString("\n")
	}

	// Legend
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(lipgloss.NewStyle().Foreground(colorSuccess).Render("●") + " office  ")
	b.WriteString(lipgloss.NewStyle().Foreground(colorSecondary).Render("●") + " remote  ")
	b.WriteString(lipgloss.NewStyle().Foreground(colorDanger).Render("●") + " holiday  ")
	b.WriteString(lipgloss.NewStyle().Foreground(colorWarning).Render("●") + " absence  ")
	b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render("●") + " weekend")

	// Selected count
	if len(c.selected) > 0 {
		b.WriteString("\n\n")
		b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(
			fmt.Sprintf("%d days selected", len(c.selected))))
		b.WriteString("  " + sDimmed.Render("enter=action  x=clear"))
	}

	// Current day info
	if info := c.dayInfo(c.cursor); info != nil {
		b.WriteString("\n\n")
		b.WriteString("  " + sValue.Render(info.Date) + "  ")
		switch info.Status {
		case "working":
			if info.Mode == "remote" {
				b.WriteString(lipgloss.NewStyle().Foreground(colorSecondary).Render("Remote"))
			} else {
				b.WriteString(lipgloss.NewStyle().Foreground(colorSuccess).Render("Office"))
			}
		case "holiday":
			b.WriteString(lipgloss.NewStyle().Foreground(colorDanger).Render("Holiday"))
		case "weekend":
			b.WriteString(sDimmed.Render("Weekend"))
		case "absence":
			b.WriteString(lipgloss.NewStyle().Foreground(colorWarning).Render("Absence"))
		}
		if len(info.EventNames) > 0 {
			b.WriteString("  " + sDimmed.Render(info.EventNames[0]))
		}
	}

	return b.String()
}

func (c *calendarGrid) renderDay(day, col int, isToday bool) string {
	label := fmt.Sprintf("%2d", day)
	style := lipgloss.NewStyle().Width(5).Align(lipgloss.Center)

	info := c.dayInfo(day)

	// Color based on status
	if info != nil {
		switch info.Status {
		case "weekend":
			style = style.Foreground(colorDim)
		case "holiday":
			style = style.Foreground(colorDanger)
		case "absence":
			style = style.Foreground(colorWarning)
		case "working":
			if info.Mode == "remote" {
				style = style.Foreground(colorSecondary)
			} else {
				style = style.Foreground(colorSuccess)
			}
		}
	} else if col >= 5 {
		style = style.Foreground(colorDim)
	}

	// Today highlight
	if isToday {
		style = style.Underline(true).Bold(true)
	}

	// Selected
	if c.selected[day] {
		style = style.Background(lipgloss.Color("#7c3aed")).Foreground(lipgloss.Color("#ffffff"))
	}

	// Cursor
	if day == c.cursor {
		if !c.selected[day] {
			style = style.Background(lipgloss.Color("#374151"))
		}
		style = style.Bold(true)
	}

	return style.Render(label)
}
