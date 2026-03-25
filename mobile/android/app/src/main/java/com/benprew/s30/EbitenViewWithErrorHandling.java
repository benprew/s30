package com.benprew.s30;

import android.content.Context;
import android.util.AttributeSet;

import com.benprew.s30.mobile.EbitenView;

class EbitenViewWithErrorHandling extends EbitenView {

    public EbitenViewWithErrorHandling(Context context) {
        super(context);
    }

    public EbitenViewWithErrorHandling(Context context, AttributeSet attributeSet) {
        super(context, attributeSet);
    }

    @Override
    protected void onErrorOnGameUpdate(Exception e) {
        super.onErrorOnGameUpdate(e);
    }
}
