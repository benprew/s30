package screens

import "testing"

func TestQuestRewardContinuesForPointerClickOrKeyboard(t *testing.T) {
	tests := []struct {
		name    string
		clicked bool
		space   bool
		escape  bool
		want    bool
	}{
		{name: "no input"},
		{name: "pointer click", clicked: true, want: true},
		{name: "space", space: true, want: true},
		{name: "escape", escape: true, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := questRewardContinues(test.clicked, test.space, test.escape); got != test.want {
				t.Fatalf("questRewardContinues() = %t, want %t", got, test.want)
			}
		})
	}
}
