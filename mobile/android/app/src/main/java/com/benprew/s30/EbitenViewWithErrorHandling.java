package com.benprew.s30;

import android.content.Context;
import android.util.AttributeSet;
import android.util.Log;

import com.benprew.s30.mobile.EbitenView;

class EbitenViewWithErrorHandling extends EbitenView {

    private static final String TAG = "S30";

    public EbitenViewWithErrorHandling(Context context) {
        super(context);
    }

    public EbitenViewWithErrorHandling(Context context, AttributeSet attributeSet) {
        super(context, attributeSet);
    }

    @Override
    protected void onErrorOnGameUpdate(Exception e) {
        Log.e(TAG, "Game update error", e);
        super.onErrorOnGameUpdate(e);
    }
}
