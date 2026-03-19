package woffu

import "testing"

func TestIsSignedIn(t *testing.T) {
	tests := []struct {
		name  string
		slots []SignSlot
		want  bool
	}{
		{"no slots", nil, false},
		{"empty slots", []SignSlot{}, false},
		{"open slot", []SignSlot{{In: "2026-03-19T08:15:00.000", Out: ""}}, true},
		{"closed slot", []SignSlot{{In: "2026-03-19T08:15:00.000", Out: "2026-03-19T13:00:00.000"}}, false},
		{"multiple slots, last open", []SignSlot{
			{In: "2026-03-19T08:15:00.000", Out: "2026-03-19T13:00:00.000"},
			{In: "2026-03-19T14:00:00.000", Out: ""},
		}, true},
		{"multiple slots, all closed", []SignSlot{
			{In: "2026-03-19T08:15:00.000", Out: "2026-03-19T13:00:00.000"},
			{In: "2026-03-19T14:00:00.000", Out: "2026-03-19T17:30:00.000"},
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSignedIn(tt.slots)
			if got != tt.want {
				t.Errorf("IsSignedIn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldSkipSign(t *testing.T) {
	openSlot := []SignSlot{{In: "2026-03-19T08:15:00.000", Out: ""}}
	closedSlot := []SignSlot{{In: "2026-03-19T08:15:00.000", Out: "2026-03-19T13:00:00.000"}}

	tests := []struct {
		name     string
		slots    []SignSlot
		expected SignAction
		want     bool
	}{
		// Expected IN cases
		{"expect in, already in → skip", openSlot, SignActionIn, true},
		{"expect in, signed out → sign", closedSlot, SignActionIn, false},
		{"expect in, no slots → sign", nil, SignActionIn, false},

		// Expected OUT cases
		{"expect out, already in → sign", openSlot, SignActionOut, false},
		{"expect out, signed out → skip", closedSlot, SignActionOut, true},
		{"expect out, no slots → skip", nil, SignActionOut, true},

		// No expected action
		{"no expected, any state → never skip", openSlot, "", false},
		{"no expected, empty → never skip", nil, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipSign(tt.slots, tt.expected)
			if got != tt.want {
				t.Errorf("ShouldSkipSign() = %v, want %v", got, tt.want)
			}
		})
	}
}
