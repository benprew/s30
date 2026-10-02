package domain

import "testing"

func TestToCardRemovesExtraText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"banding", "Trample, banding (Any creatures with banding can attack in a band.)", "Trample, banding"},
		{"multiple", "Flying (Reminder.)\nVigilance (Reminder.)", "Flying\nVigilance"},
		{"nested", "Banding (Reminder (with more detail).)\nTrample", "Banding\nTrample"},
		{"middle", "Add (one mana) {G}.", "Add {G}."},
		{"reminder line", "Flying\n(Reminder.)\nVigilance", "Flying\nVigilance"},
		{"rules only", "Deal 3 damage to any target.\nDraw a card.", "Deal 3 damage to any target.\nDraw a card."},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := CardJSON{CardName: "War Elephant", Text: tt.text, FlavorText: "A long story."}
			card := data.ToCard()
			if card.Text != tt.want {
				t.Errorf("Text = %q, want %q", card.Text, tt.want)
			}
			if card.FlavorText != "" {
				t.Errorf("FlavorText = %q, want empty", card.FlavorText)
			}
			if data.Text != tt.text || data.FlavorText != "A long story." {
				t.Error("source card data changed")
			}
		})
	}
}
