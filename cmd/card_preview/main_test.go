package main

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestPreviewRendersAllViews(t *testing.T) {
	domain.ClearCardImageCache()
	t.Cleanup(domain.ClearCardImageCache)
	card := domain.FindCardByName("Serra Angel")
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, 186, 118)))
	preview, err := newCardPreview("Serra Angel")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.views) != 4 {
		t.Fatalf("got %d views, want 4", len(preview.views))
	}
	for _, view := range preview.views {
		if view.img == nil || view.img.Bounds().Size() != image.Point(view.size) {
			t.Fatalf("wrong image size for %s", view.name)
		}
	}
}

func TestPreviewRejectsUnknownCard(t *testing.T) {
	if _, err := newCardPreview("No such card"); err == nil {
		t.Fatal("unknown card was accepted")
	}
}
