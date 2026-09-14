package ios_test

import (
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestIOSPackageName(t *testing.T) {
	t.Parallel()

	files := map[string][]string{
		"s30.xcodeproj/project.pbxproj": {
			"PRODUCT_BUNDLE_IDENTIFIER = com.throwingbones.s30;",
			"IPHONEOS_DEPLOYMENT_TARGET = 13.0;",
			"Mobile.xcframework",
			"GameController.framework",
			"AVFoundation.framework",
			"AudioToolbox.framework",
			"Metal.framework",
		},
		"s30/Info.plist": {
			"<string>$(PRODUCT_BUNDLE_IDENTIFIER)</string>",
			"<string>Shandalar 30</string>",
			"<string>UIInterfaceOrientationLandscapeLeft</string>",
			"<string>UIInterfaceOrientationLandscapeRight</string>",
		},
		"s30/AppDelegate.m": {
			"#import \"AppDelegate.h\"",
			"#import \"MobileEbitenViewControllerWithErrorHandling.h\"",
			"#import <Mobile/Mobile.h>",
			"MobileSetSaveDir(documentsDirectory);",
			"MobileSaveGame();",
		},
		"s30/MobileEbitenViewControllerWithErrorHandling.h": {
			"#import <Mobile/Mobile.h>",
			"@interface MobileEbitenViewControllerWithErrorHandling : MobileEbitenViewController",
		},
		"s30/Base.lproj/Main.storyboard": {
			"customClass=\"MobileEbitenViewControllerWithErrorHandling\"",
		},
		"../../.github/workflows/ios.yml": {
			"-target ios",
			"-tags embedded_card_images",
			"Mobile.xcframework",
			"s30.xcodeproj",
			"s30_ios.ipa",
			"s30_ios.xcarchive.zip",
		},
		"../../.github/workflows/release.yml": {
			"uses: ./.github/workflows/ios.yml",
			"needs: [build, android, ios]",
			"artifacts/s30_ios.ipa/s30_ios.ipa",
			"artifacts/s30_ios.xcarchive.zip/s30_ios.xcarchive.zip",
		},
	}

	for path, expectedValues := range files {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}

		for _, expected := range expectedValues {
			if !strings.Contains(string(contents), expected) {
				t.Errorf("%s does not contain %q", path, expected)
			}
		}
	}
}

func TestIOSLauncherIconHasIntrinsicSize(t *testing.T) {
	t.Parallel()

	contentsJSON, err := os.ReadFile("s30/Assets.xcassets/AppIcon.appiconset/Contents.json")
	if err != nil {
		t.Fatalf("read Contents.json: %v", err)
	}
	if !strings.Contains(string(contentsJSON), `"filename" : "app-icon.png"`) {
		t.Error("Contents.json must reference app-icon.png")
	}
	if !strings.Contains(string(contentsJSON), `"1024x1024"`) {
		t.Error("Contents.json must specify 1024x1024 icon size")
	}

	iconPath := "s30/Assets.xcassets/AppIcon.appiconset/app-icon.png"
	file, err := os.Open(iconPath)
	if err != nil {
		t.Fatalf("open %s: %v", iconPath, err)
	}
	defer file.Close()

	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("decode %s: %v", iconPath, err)
	}
	if config.Width != 1024 || config.Height != 1024 {
		t.Errorf("%s is %dx%d, want 1024x1024", iconPath, config.Width, config.Height)
	}
}

func TestGitHubActionsUseNode24Runtimes(t *testing.T) {
	t.Parallel()

	files := map[string][]string{
		"../../.github/workflows/ios.yml": {
			"actions/checkout@v5",
			"actions/setup-go@v6",
			"astral-sh/setup-uv@v7",
			"actions/upload-artifact@v7",
		},
		"../../.github/workflows/release.yml": {
			"actions/checkout@v5",
			"actions/setup-go@v6",
			"astral-sh/setup-uv@v7",
		},
	}

	for path, expectedValues := range files {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}

		for _, expected := range expectedValues {
			if !strings.Contains(string(contents), expected) {
				t.Errorf("%s does not contain %q", path, expected)
			}
		}
	}
}
