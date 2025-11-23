package screens

import (
	"strings"
	"testing"
)

func TestPaginateText(t *testing.T) {
	// Placeholder
}

func TestLoadStories(t *testing.T) {
	stories := loadStories()
	if len(stories) == 0 {
		t.Errorf("Expected stories, got none")
	}
	for i, story := range stories {
		if story == "" {
			t.Errorf("Story %d is empty", i)
		}
		if strings.Contains(story, "STARTBLOCK") || strings.Contains(story, "ENDBLOCK") {
			t.Errorf("Story %d contains delimiters", i)
		}
	}
}
