package com.benprew.s30;

import androidx.appcompat.app.AppCompatActivity;
import android.os.Bundle;
import android.util.Log;

import go.Seq;
import com.benprew.s30.mobile.EbitenView;

public class MainActivity extends AppCompatActivity {

    private static final String TAG = "S30";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        Log.i(TAG, "onCreate: setting content view");
        setContentView(R.layout.activity_main);
        Log.i(TAG, "onCreate: setting Seq context");
        Seq.setContext(getApplicationContext());
        Log.i(TAG, "onCreate: done");
    }

    private EbitenView getEbitenView() {
        return (EbitenView) this.findViewById(R.id.ebitenview);
    }

    @Override
    protected void onPause() {
        super.onPause();
        this.getEbitenView().suspendGame();
    }

    @Override
    protected void onResume() {
        super.onResume();
        this.getEbitenView().resumeGame();
    }
}
