package android_test

import (
	"os"
	"strings"
	"testing"
)

func TestAndroidPackageName(t *testing.T) {
	t.Parallel()

	files := map[string][]string{
		"app/build.gradle": {
			"namespace 'com.throwingbones.s30'",
			"applicationId \"com.throwingbones.s30\"",
			"compileSdk 36",
			"targetSdk 36",
		},
		"build.gradle": {
			"com.android.tools.build:gradle:8.9.1",
		},
		"gradle/wrapper/gradle-wrapper.properties": {
			"gradle-8.11.1-bin.zip",
		},
		"app/src/main/java/com/throwingbones/s30/MainActivity.java": {
			"package com.throwingbones.s30;",
			"import com.throwingbones.s30.mobile.EbitenView;",
		},
		"app/src/main/java/com/throwingbones/s30/EbitenViewWithErrorHandling.java": {
			"package com.throwingbones.s30;",
			"import com.throwingbones.s30.mobile.EbitenView;",
		},
		"app/src/main/res/layout/activity_main.xml": {
			"com.throwingbones.s30.MainActivity",
			"com.throwingbones.s30.EbitenViewWithErrorHandling",
		},
		"../../.github/workflows/android.yml": {
			"-javapkg com.throwingbones.s30",
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

func TestAndroidReleaseSigning(t *testing.T) {
	t.Parallel()

	files := map[string][]string{
		"app/build.gradle": {
			"ANDROID_KEYSTORE_PATH",
			"ANDROID_KEYSTORE_PASSWORD",
			"ANDROID_KEY_ALIAS",
			"ANDROID_KEY_PASSWORD",
			"ANDROID_VERSION_CODE",
			"ANDROID_VERSION_NAME",
			"signingConfig signingConfigs.release",
			"enableV1Signing true",
			"enableV2Signing true",
			"enableV3Signing true",
			"enableV4Signing true",
		},
		"../../.github/workflows/android.yml": {
			"ANDROID_UPLOAD_KEYSTORE_BASE64",
			"ANDROID_VERSION_CODE: ${{ github.run_number }}",
			"name: Validate release signing secrets",
			"Missing required Android release signing secret",
			"base64 --decode",
			"gradle bundleRelease assembleRelease",
			"verify --verbose --print-certs",
			"Verified using v1 scheme (JAR signing): true",
			"Verified using v2 scheme (APK Signature Scheme v2): true",
			"Verified using v3 scheme (APK Signature Scheme v3): true",
			"Verified using v4 scheme (APK Signature Scheme v4): true",
			"s30_android.apk.idsig",
			"jarsigner -verify",
			"s30_android.aab",
		},
		"../../.github/workflows/release.yml": {
			"secrets: inherit",
			"artifacts/s30_android.apk/s30_android.apk.idsig",
			"artifacts/s30_android.aab/s30_android.aab",
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

	workflow, err := os.ReadFile("../../.github/workflows/android.yml")
	if err != nil {
		t.Fatalf("read Android workflow: %v", err)
	}

	workflowContents := string(workflow)
	validateSigning := strings.Index(workflowContents, "name: Validate release signing secrets")
	restoreKeystore := strings.Index(workflowContents, "name: Restore upload keystore")
	if validateSigning < 0 || restoreKeystore < 0 || validateSigning > restoreKeystore {
		t.Error("Android workflow must validate signing secrets before restoring the keystore")
	}

	for _, forbidden := range []string{"debug.keystore", "/secure/path", "keytool -genkey"} {
		if strings.Contains(workflowContents, forbidden) {
			t.Errorf("Android workflow contains obsolete signing configuration %q", forbidden)
		}
	}
}

func TestGitHubActionsUseNode24Runtimes(t *testing.T) {
	t.Parallel()

	files := map[string][]string{
		"../../.github/workflows/android.yml": {
			"actions/checkout@v5",
			"actions/setup-go@v6",
			"actions/setup-java@v5",
			"android-actions/setup-android@v4",
			"astral-sh/setup-uv@v7",
			"gradle/actions/setup-gradle@v5",
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
