package com.musicroom.music_room

import android.os.Build
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            "com.musicroom/device_info",
        ).setMethodCallHandler { call, result ->
            when (call.method) {
                "getModel"        -> result.success(Build.MODEL)
                "getManufacturer" -> result.success(Build.MANUFACTURER)
                "getDeviceName"   -> result.success("${Build.MANUFACTURER} ${Build.MODEL}")
                else              -> result.notImplemented()
            }
        }
    }
}
