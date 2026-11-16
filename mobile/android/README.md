# Android releases

Release APKs and Android App Bundles are signed with the same upload key. The
GitHub Actions workflow restores that key from repository secrets and passes the
signing configuration to Gradle. Release APKs use v1, v2, v3, and v4 signing.
The v4 signature is published as the separate `s30_android.apk.idsig` file that
Android's v4 scheme requires.

Configure these GitHub Actions repository secrets:

- `ANDROID_UPLOAD_KEYSTORE_BASE64`: the base64-encoded upload keystore
- `ANDROID_UPLOAD_KEYSTORE_PASSWORD`: the keystore password
- `ANDROID_UPLOAD_KEY_ALIAS`: the upload key alias
- `ANDROID_UPLOAD_KEY_PASSWORD`: the upload key password

Create the encoded keystore value without line breaks:

```bash
base64 < /secure/path/s30-upload.jks | tr -d '\n'
```

For a local signed build, set `ANDROID_KEYSTORE_PATH`,
`ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, and `ANDROID_KEY_PASSWORD`,
build the Ebiten AAR, and then run:

```bash
gradle bundleRelease assembleRelease
```

Gradle can still produce an unsigned local release when none of the signing
environment variables are set. Supplying only some of them is treated as an
error.

For stable release builds, GitHub Actions derives `versionCode` from the tag:
`major * 1,000,000 + minor * 1,000 + patch`. For example, `v1.2.3` gives
`1002003`. Minor and patch numbers must be below 1000, and the major number must
be below 2100. This keeps codes ordered across the Android and main release
workflows. Branch builds still use their workflow run number. The release tag or
supplied version becomes `versionName`. Every distributed update must have a
higher `versionCode` than the installed app.

The game downloads the universal APK into its private update cache and verifies
the signed manifest, file size, and SHA-256 checksum. Updates start from the title
screen before a game is created or loaded. Android 8 and later can first ask the player to allow
S30 to install unknown apps; returning from that setting opens the pending APK.
The player must approve the installation. Cancelling it leaves the current app
installed; **Install update** opens the downloaded APK again.

The update provider exposes only `cache/updates/`, with temporary read access for
the installer. Android checks the APK's application ID and signing certificate
before replacing the app. Keep the same signing key for direct APK releases.
Builds signed by Play App Signing or another distributor can have a different
certificate and must use that distributor's update path. Never uninstall the app
to work around a signature mismatch, because that can remove its private saves.

Configure `UPDATE_SIGNING_KEY` and `UPDATE_PUBLIC_KEY` as described in the root
README. The update-manifest signing key does not replace the APK signing key.
