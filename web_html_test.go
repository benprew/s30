package main

import (
	"os"
	"strings"
	"testing"
)

func TestWebPagesCenterAndScaleGame(t *testing.T) {
	tests := []struct {
		path     string
		required []string
		forbid   []string
	}{
		{
			path: "index.html",
			required: []string{
				"background: #111;",
				"display: grid;",
				"place-items: center;",
				"width: min(100vw, calc(100vh * 4 / 3));",
				"height: min(100vh, calc(100vw * 3 / 4));",
			},
			forbid: []string{`width="1024"`, `height="768"`},
		},
		{
			path: "main.html",
			required: []string{
				"place-items: center;",
				"canvas",
				"width: min(100vw, calc(100vh * 4 / 3)) !important;",
				"height: min(100vh, calc(100vw * 3 / 4)) !important;",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			contents, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}

			page := string(contents)
			for _, required := range test.required {
				if !strings.Contains(page, required) {
					t.Errorf("missing responsive web style %q", required)
				}
			}
			for _, forbidden := range test.forbid {
				if strings.Contains(page, forbidden) {
					t.Errorf("fixed game dimension remains: %q", forbidden)
				}
			}
		})
	}
}
