package github

import (
	"strings"
	"testing"

	"github.com/ngavilan-dogfy/woffux/internal/config"
)

func standardSchedule() config.Schedule {
	monThu := config.DaySchedule{
		Enabled: true,
		Times: []config.ScheduleEntry{
			{Time: "08:30"}, {Time: "13:30"}, {Time: "14:15"}, {Time: "17:30"},
		},
	}
	fri := config.DaySchedule{
		Enabled: true,
		Times: []config.ScheduleEntry{
			{Time: "08:00"}, {Time: "15:00"},
		},
	}
	return config.Schedule{
		Monday: monThu, Tuesday: monThu, Wednesday: monThu, Thursday: monThu,
		Friday: fri,
	}
}

func TestGenerateCrons_DSTZone(t *testing.T) {
	crons := GenerateCrons(standardSchedule(), "Europe/Madrid")

	// 4 Mon-Thu times + 2 Friday times = 6 entries
	if len(crons) != 6 {
		t.Fatalf("expected 6 cron entries, got %d", len(crons))
	}

	// All entries for DST zone should have comma-separated hours
	for _, c := range crons {
		if !strings.Contains(c.Cron, ",") {
			t.Errorf("DST cron should have comma-separated hours: %s", c.Cron)
		}
		// Should NOT contain month ranges
		parts := strings.Fields(c.Cron)
		if len(parts) == 5 && parts[3] != "*" {
			t.Errorf("cron should have * for month field, got: %s", parts[3])
		}
	}
}

func TestGenerateCrons_CETAlias(t *testing.T) {
	cronsAlias := GenerateCrons(standardSchedule(), "CET")
	cronsFull := GenerateCrons(standardSchedule(), "Europe/Madrid")

	if len(cronsAlias) != len(cronsFull) {
		t.Fatalf("CET alias should produce same count as Europe/Madrid: %d vs %d",
			len(cronsAlias), len(cronsFull))
	}

	for i := range cronsAlias {
		if cronsAlias[i].Cron != cronsFull[i].Cron {
			t.Errorf("entry %d differs: %q vs %q", i, cronsAlias[i].Cron, cronsFull[i].Cron)
		}
	}
}

func TestGenerateCrons_NoDSTZone(t *testing.T) {
	crons := GenerateCrons(standardSchedule(), "UTC")

	if len(crons) != 6 {
		t.Fatalf("expected 6 cron entries, got %d", len(crons))
	}

	// No DST → no comma-separated hours
	for _, c := range crons {
		parts := strings.Fields(c.Cron)
		if strings.Contains(parts[1], ",") {
			t.Errorf("non-DST cron should not have comma-separated hours: %s", c.Cron)
		}
	}
}

func TestGenerateCrons_EST(t *testing.T) {
	schedule := config.Schedule{
		Monday: config.DaySchedule{
			Enabled: true,
			Times:   []config.ScheduleEntry{{Time: "09:00"}},
		},
	}
	crons := GenerateCrons(schedule, "America/New_York")

	if len(crons) != 1 {
		t.Fatalf("expected 1 cron entry, got %d", len(crons))
	}

	// EST = UTC-5, EDT = UTC-4. 09:00 → UTC 14 (EST) or 13 (EDT)
	c := crons[0]
	if !strings.Contains(c.Cron, "13,14") {
		t.Errorf("expected hours 13,14 for 09:00 EST/EDT, got: %s", c.Cron)
	}
}

func TestGenerateCrons_SouthernHemisphere(t *testing.T) {
	schedule := config.Schedule{
		Monday: config.DaySchedule{
			Enabled: true,
			Times:   []config.ScheduleEntry{{Time: "08:00"}},
		},
	}
	crons := GenerateCrons(schedule, "Australia/Sydney")

	if len(crons) != 1 {
		t.Fatalf("expected 1 cron entry, got %d", len(crons))
	}

	// AEST=UTC+10, AEDT=UTC+11. 08:00 → UTC 22 (prev day, AEST) or 21 (AEDT)
	c := crons[0]
	if !strings.Contains(c.Cron, "21,22") {
		t.Errorf("expected hours 21,22 for 08:00 AEST/AEDT, got: %s", c.Cron)
	}
}

func TestGenerateCrons_DeterministicOrder(t *testing.T) {
	schedule := standardSchedule()

	// Run multiple times and ensure same output
	first := GenerateCrons(schedule, "CET")
	for i := 0; i < 20; i++ {
		got := GenerateCrons(schedule, "CET")
		if len(got) != len(first) {
			t.Fatalf("iteration %d: length changed %d vs %d", i, len(got), len(first))
		}
		for j := range first {
			if got[j].Cron != first[j].Cron {
				t.Errorf("iteration %d, entry %d: %q vs %q", i, j, got[j].Cron, first[j].Cron)
			}
		}
	}
}

func TestGenerateCrons_DisabledDays(t *testing.T) {
	schedule := config.Schedule{
		Monday:  config.DaySchedule{Enabled: true, Times: []config.ScheduleEntry{{Time: "08:00"}}},
		Tuesday: config.DaySchedule{Enabled: false},
	}
	crons := GenerateCrons(schedule, "UTC")

	if len(crons) != 1 {
		t.Fatalf("expected 1 entry (only Monday), got %d", len(crons))
	}

	// Should only have day 1 (Monday)
	if !strings.HasSuffix(crons[0].Cron, " 1") {
		t.Errorf("expected day 1 only, got: %s", crons[0].Cron)
	}
}

func TestSignTimes(t *testing.T) {
	times := signTimes(standardSchedule())

	expected := map[string]bool{
		"08:30": true, "13:30": true, "14:15": true, "17:30": true,
		"08:00": true, "15:00": true,
	}

	if len(times) != len(expected) {
		t.Fatalf("expected %d unique times, got %d: %v", len(expected), len(times), times)
	}

	for _, tt := range times {
		if !expected[tt] {
			t.Errorf("unexpected sign time: %s", tt)
		}
	}
}

func TestLocalToUTC(t *testing.T) {
	tests := []struct {
		hour, offset, want int
	}{
		{8, 1, 7},    // CET
		{8, 2, 6},    // CEST
		{8, -5, 13},  // EST
		{0, 1, 23},   // midnight CET → 23 UTC
		{23, -1, 0},  // 23:00 UTC-1 → 00 UTC
		{8, 10, 22},  // AEST → previous day UTC
	}

	for _, tt := range tests {
		got := localToUTC(tt.hour, tt.offset)
		if got != tt.want {
			t.Errorf("localToUTC(%d, %d) = %d, want %d", tt.hour, tt.offset, got, tt.want)
		}
	}
}

func TestHasDST(t *testing.T) {
	if !hasDST("Europe/Madrid") {
		t.Error("Europe/Madrid should have DST")
	}
	if !hasDST("CET") {
		t.Error("CET alias should have DST")
	}
	if hasDST("UTC") {
		t.Error("UTC should not have DST")
	}
}

func TestGenerateWorkflowYAML_ContainsGuard(t *testing.T) {
	yaml := GenerateWorkflowYAML(standardSchedule(), "CET")

	if !strings.Contains(yaml, "Timezone guard") {
		t.Error("DST workflow should contain timezone guard step")
	}
	if !strings.Contains(yaml, "CRON_MIN") {
		t.Error("guard should extract CRON_MIN")
	}
	if !strings.Contains(yaml, `printf "%02d:%02d"`) {
		t.Error("guard should format HH:MM with printf")
	}
}

func TestGenerateWorkflowYAML_NoDSTNoGuard(t *testing.T) {
	yaml := GenerateWorkflowYAML(standardSchedule(), "UTC")

	if strings.Contains(yaml, "Timezone guard") {
		t.Error("non-DST workflow should NOT contain timezone guard")
	}
}

func TestGenerateWorkflowYAML_ValidBashSyntax(t *testing.T) {
	yaml := GenerateWorkflowYAML(standardSchedule(), "CET")

	// The %% bug: ensure no double %% in the output (only single %)
	if strings.Contains(yaml, "RANDOM %%") {
		t.Error("YAML contains %% which is invalid bash — should be single %")
	}
	if !strings.Contains(yaml, "RANDOM % ") {
		t.Error("YAML should contain RANDOM % (valid bash modulo)")
	}
}

func TestGenerateWorkflowYAML_ContainsExpectedGuard(t *testing.T) {
	yaml := GenerateWorkflowYAML(standardSchedule(), "CET")

	// Should contain the case statement for --expected flag
	if !strings.Contains(yaml, `--expected "$EXPECTED"`) {
		t.Error("workflow should pass --expected flag to woffux sign")
	}
	if !strings.Contains(yaml, `case "$CRON" in`) {
		t.Error("workflow should contain case statement for cron→action mapping")
	}
	// Check that IN and OUT actions are both present
	if !strings.Contains(yaml, `EXPECTED="in"`) {
		t.Error("workflow should map some crons to 'in' action")
	}
	if !strings.Contains(yaml, `EXPECTED="out"`) {
		t.Error("workflow should map some crons to 'out' action")
	}
}

func TestGenerateCrons_ActionField(t *testing.T) {
	crons := GenerateCrons(standardSchedule(), "CET")

	// Mon-Thu: 08:30=in, 13:30=out, 14:15=in, 17:30=out
	// Fri: 08:00=in, 15:00=out
	// Sorted by cron key: Fri first (08:00, 15:00), then Mon-Thu (08:30, 13:30, 14:15, 17:30)
	expectedActions := []string{"in", "out", "in", "out", "in", "out"}

	if len(crons) != len(expectedActions) {
		t.Fatalf("expected %d entries, got %d", len(expectedActions), len(crons))
	}

	for i, c := range crons {
		if c.Action != expectedActions[i] {
			t.Errorf("entry %d (%s): action = %q, want %q", i, c.Comment, c.Action, expectedActions[i])
		}
	}
}
