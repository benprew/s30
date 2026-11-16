package com.throwingbones.s30;

import android.content.Intent;
import android.content.pm.PackageInfo;
import android.net.Uri;
import android.os.Build;
import android.os.Looper;
import android.provider.Settings;
import android.widget.Toast;

import androidx.core.content.FileProvider;

import com.throwingbones.s30.mobile.UpdateInstaller;

import java.io.File;
import java.io.IOException;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

final class AndroidUpdateInstaller implements UpdateInstaller {
    static final int INSTALL_PERMISSION_REQUEST = 310;
    private final MainActivity activity;

    AndroidUpdateInstaller(MainActivity activity) {
        this.activity = activity;
    }

    @Override
    public void installUpdate(String apkPath) throws Exception {
        File apk = validateApk(apkPath);
        onMainThread(() -> {
            activity.getPreferences(0).edit().putString("pending_update", apk.getAbsolutePath()).apply();
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O
                    && !activity.getPackageManager().canRequestPackageInstalls()) {
                Intent settings = new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                        Uri.parse("package:" + activity.getPackageName()));
                activity.startActivityForResult(settings, INSTALL_PERMISSION_REQUEST);
            } else {
                openInstaller(apk);
            }
        });
    }

    private File validateApk(String apkPath) throws IOException {
        File apk = new File(apkPath).getCanonicalFile();
        File updates = new File(activity.getCacheDir(), "updates").getCanonicalFile();
        if (!apk.getPath().startsWith(updates.getPath() + File.separator) || !apk.isFile()) {
            throw new IOException("The update file is outside the private update directory");
        }
        PackageInfo candidate = activity.getPackageManager().getPackageArchiveInfo(apk.getPath(), 0);
        if (candidate == null || !activity.getPackageName().equals(candidate.packageName)) {
            throw new IOException("The APK has a different application ID");
        }
        long code = Build.VERSION.SDK_INT >= Build.VERSION_CODES.P
                ? candidate.getLongVersionCode() : candidate.versionCode;
        if (code <= BuildConfig.VERSION_CODE) {
            throw new IOException("The APK version code must increase");
        }
        return apk;
    }

    private void openInstaller(File apk) {
        Uri uri = FileProvider.getUriForFile(activity, activity.getPackageName() + ".updates", apk);
        Intent install = new Intent(Intent.ACTION_VIEW);
        install.setDataAndType(uri, "application/vnd.android.package-archive");
        install.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
        activity.startActivity(install);
        activity.getPreferences(0).edit().remove("pending_update").apply();
    }

    void onPermissionResult() {
        String path = activity.getPreferences(0).getString("pending_update", null);
        activity.getPreferences(0).edit().remove("pending_update").apply();
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O
                && !activity.getPackageManager().canRequestPackageInstalls()) {
            Toast.makeText(activity, "Allow S30 to install updates, then tap Install update again.", Toast.LENGTH_LONG).show();
            return;
        }
        if (path == null) {
            return;
        }
        new Thread(() -> {
            try {
                File apk = validateApk(path);
                onMainThread(() -> openInstaller(apk));
            } catch (Exception error) {
                activity.runOnUiThread(() -> Toast.makeText(activity,
                        "Could not open the update: " + error.getMessage(), Toast.LENGTH_LONG).show());
            }
        }, "s30-update-installer").start();
    }

    @Override
    public void openDownload(String url) throws Exception {
        onMainThread(() -> activity.startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(url))));
    }

    private void onMainThread(Runnable action) throws Exception {
        if (Looper.myLooper() == Looper.getMainLooper()) {
            action.run();
            return;
        }
        CountDownLatch done = new CountDownLatch(1);
        AtomicReference<RuntimeException> failure = new AtomicReference<>();
        activity.runOnUiThread(() -> {
            try {
                action.run();
            } catch (RuntimeException error) {
                failure.set(error);
            } finally {
                done.countDown();
            }
        });
        if (!done.await(10, TimeUnit.SECONDS)) {
            throw new IOException("The Android installer did not respond");
        }
        if (failure.get() != null) {
            throw failure.get();
        }
    }
}
