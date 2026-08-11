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

GitHub Actions uses the workflow run number as `versionCode` and the release tag
or supplied version as `versionName`. Each Play Store release must have a higher
`versionCode` than the previous release.
