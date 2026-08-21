package elements

import (
	"testing"
)

func TestTextInputOperations(t *testing.T) {
	input := NewTextInput(100, 100, 400, 200, "Describe the bug...")

	if input.Text() != "" {
		t.Errorf("Initial text = %q, want empty", input.Text())
	}

	input.SetText("First line")
	if input.Text() != "First line" {
		t.Errorf("Text = %q, want First line", input.Text())
	}

	input.AppendText("\nSecond line")
	if input.Text() != "First line\nSecond line" {
		t.Errorf("Text = %q, want multiline text", input.Text())
	}

	input.HandleBackspace()
	if input.Text() != "First line\nSecond lin" {
		t.Errorf("Text after backspace = %q", input.Text())
	}

	input.InsertChar('e')
	if input.Text() != "First line\nSecond line" {
		t.Errorf("Text after insert = %q", input.Text())
	}

	input.Clear()
	if input.Text() != "" {
		t.Errorf("Text after clear = %q, want empty", input.Text())
	}
}
