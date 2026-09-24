package com.dualsense.gamepad;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Context;
import android.content.SharedPreferences;
import android.graphics.Color;
import android.net.wifi.WifiManager;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.os.VibrationEffect;
import android.os.Vibrator;
import android.view.View;
import android.view.Window;
import android.view.WindowManager;
import android.webkit.JavascriptInterface;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceError;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.FrameLayout;

import java.net.DatagramPacket;
import java.net.DatagramSocket;
import java.net.InetAddress;
import java.net.InterfaceAddress;
import java.net.NetworkInterface;
import java.util.Enumeration;
import java.util.concurrent.atomic.AtomicBoolean;

public class MainActivity extends Activity {
    private static final String PREFS_NAME = "DualSensePrefs";
    private static final String KEY_LAST_HOST = "last_host";
    private static final String KEY_LAST_SLOT = "last_slot";
    private static final int UDP_PORT = 9050;

    private WebView webView;
    private Vibrator vibrator;
    private WifiManager.MulticastLock multicastLock;
    private final AtomicBoolean isDiscovering = new AtomicBoolean(false);
    private Thread discoveryThread = null;
    private boolean isControllerActive = false;
    private SharedPreferences prefs;
    private final Handler mainHandler = new Handler(Looper.getMainLooper());

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        // Window & Fullscreen setup
        requestWindowFeature(Window.FEATURE_NO_TITLE);
        getWindow().setFlags(WindowManager.LayoutParams.FLAG_FULLSCREEN, WindowManager.LayoutParams.FLAG_FULLSCREEN);
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);

        // Extend into camera cutout / notch
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            getWindow().getAttributes().layoutInDisplayCutoutMode =
                WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
        }

        setImmersive();

        prefs = getSharedPreferences(PREFS_NAME, MODE_PRIVATE);
        vibrator = (Vibrator) getSystemService(Context.VIBRATOR_SERVICE);

        // Multicast lock for UDP broadcast reception
        try {
            WifiManager wifi = (WifiManager) getApplicationContext().getSystemService(Context.WIFI_SERVICE);
            if (wifi != null) {
                multicastLock = wifi.createMulticastLock("DualSense_UDP_Lock");
                multicastLock.setReferenceCounted(true);
                multicastLock.acquire();
            }
        } catch (Exception ignored) {}

        // Setup WebView
        webView = new WebView(this);
        webView.setBackgroundColor(Color.parseColor("#06080E"));
        setupWebView();

        FrameLayout layout = new FrameLayout(this);
        layout.setBackgroundColor(Color.parseColor("#06080E"));
        layout.addView(webView, new FrameLayout.LayoutParams(
            FrameLayout.LayoutParams.MATCH_PARENT,
            FrameLayout.LayoutParams.MATCH_PARENT
        ));
        setContentView(layout);

        // Load Lobby View initially
        loadLobby();
    }

    private void setupWebView() {
        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setDatabaseEnabled(true);
        settings.setAllowFileAccess(true);
        settings.setAllowContentAccess(true);
        settings.setMediaPlaybackRequiresUserGesture(false);
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);
        settings.setCacheMode(WebSettings.LOAD_NO_CACHE);
        settings.setUseWideViewPort(true);
        settings.setLoadWithOverviewMode(true);

        // Enable multi-touch and disable zoom controls
        settings.setSupportZoom(false);
        settings.setBuiltInZoomControls(false);
        settings.setDisplayZoomControls(false);

        webView.setWebChromeClient(new WebChromeClient());
        webView.setWebViewClient(new WebViewClient() {
            @Override
            public void onPageFinished(WebView view, String url) {
                super.onPageFinished(view, url);
                setImmersive();

                // If loading the live controller from PC, inject native hardware vibration hook
                if (url != null && !url.contains("lobby.html")) {
                    injectHapticsBridge();
                }
            }

            @Override
            public void onReceivedError(WebView view, WebResourceRequest request, WebResourceError error) {
                super.onReceivedError(view, request, error);
                if (isControllerActive && request.isForMainFrame()) {
                    mainHandler.post(() -> {
                        new AlertDialog.Builder(MainActivity.this)
                            .setTitle("Connection Error")
                            .setMessage("Could not connect to PC server. Make sure DualSenseServer.exe is running on your PC and both devices are on the same Wi-Fi/Hotspot.")
                            .setPositiveButton("Back to Lobby", (d, w) -> loadLobby())
                            .setCancelable(false)
                            .show();
                    });
                }
            }
        });

        webView.addJavascriptInterface(new NativeInterface(), "AndroidNative");
    }

    private void injectHapticsBridge() {
        String js = "(function() {" +
            "  if (window.__nativeHapticsInjected) return;" +
            "  window.__nativeHapticsInjected = true;" +
            "  window.triggerNativeHardwareVibrate = function(heavy, light) {" +
            "    var amp = Math.max(heavy, light);" +
            "    if (amp > 0 && window.AndroidNative) {" +
            "      window.AndroidNative.vibrate(90, Math.min(255, Math.floor(amp * 1.2)));" +
            "    }" +
            "  };" +
            "  if (window.navigator && window.AndroidNative) {" +
            "    var origVibrate = navigator.vibrate;" +
            "    navigator.vibrate = function(pattern) {" +
            "      if (typeof pattern === 'number') {" +
            "        window.AndroidNative.vibrate(pattern, 200);" +
            "      } else if (Array.isArray(pattern) && pattern.length > 0) {" +
            "        window.AndroidNative.vibrate(pattern[0], 200);" +
            "      }" +
            "      if (origVibrate) origVibrate.apply(navigator, arguments);" +
            "    };" +
            "  }" +
            "})();";
        webView.evaluateJavascript(js, null);
    }

    private void loadLobby() {
        isControllerActive = false;
        webView.loadUrl("file:///android_asset/lobby.html");
        startAutoDiscovery();
    }

    private void startAutoDiscovery() {
        if (isDiscovering.getAndSet(true)) return;

        discoveryThread = new Thread(() -> {
            DatagramSocket socket = null;
            try {
                socket = new DatagramSocket();
                socket.setBroadcast(true);
                socket.setSoTimeout(1500);

                byte[] pingData = "VIBE_PING".getBytes();
                byte[] recvBuf = new byte[256];

                while (isDiscovering.get()) {
                    // Send broadcast ping to common subnets
                    try {
                        // 1. General 255.255.255.255
                        DatagramPacket pingPacket = new DatagramPacket(
                            pingData, pingData.length,
                            InetAddress.getByName("255.255.255.255"), UDP_PORT
                        );
                        socket.send(pingPacket);

                        // 2. Windows Mobile Hotspot default subnet (192.168.137.255)
                        DatagramPacket hotspotPacket = new DatagramPacket(
                            pingData, pingData.length,
                            InetAddress.getByName("192.168.137.255"), UDP_PORT
                        );
                        socket.send(hotspotPacket);

                        // 3. Local interface broadcast addresses
                        Enumeration<NetworkInterface> interfaces = NetworkInterface.getNetworkInterfaces();
                        if (interfaces != null) {
                            while (interfaces.hasMoreElements()) {
                                NetworkInterface nif = interfaces.nextElement();
                                if (nif.isLoopback() || !nif.isUp()) continue;
                                for (InterfaceAddress ifAddr : nif.getInterfaceAddresses()) {
                                    InetAddress broadcast = ifAddr.getBroadcast();
                                    if (broadcast != null) {
                                        socket.send(new DatagramPacket(pingData, pingData.length, broadcast, UDP_PORT));
                                    }
                                }
                            }
                        }
                    } catch (Exception ignored) {}

                    // Receive response
                    long waitEnd = System.currentTimeMillis() + 1800;
                    while (System.currentTimeMillis() < waitEnd && isDiscovering.get()) {
                        try {
                            DatagramPacket resp = new DatagramPacket(recvBuf, recvBuf.length);
                            socket.receive(resp);
                            String msg = new String(resp.getData(), 0, resp.getLength());
                            if (msg.startsWith("VIBE_PONG")) {
                                String hostIp = resp.getAddress().getHostAddress();
                                mainHandler.post(() -> {
                                    webView.evaluateJavascript("if (window.onHostDiscovered) window.onHostDiscovered('" + hostIp + "', 8080);", null);
                                });
                                // Discovered! Reduce poll frequency
                                Thread.sleep(3000);
                                break;
                            }
                        } catch (Exception ignored) {}
                    }

                    Thread.sleep(1200);
                }
            } catch (Exception e) {
                // socket closed or error
            } finally {
                if (socket != null && !socket.isClosed()) {
                    socket.close();
                }
                isDiscovering.set(false);
            }
        });
        discoveryThread.start();
    }

    private void stopAutoDiscovery() {
        isDiscovering.set(false);
        if (discoveryThread != null) {
            discoveryThread.interrupt();
            discoveryThread = null;
        }
    }

    private void setImmersive() {
        runOnUiThread(() -> {
            getWindow().getDecorView().setSystemUiVisibility(
                View.SYSTEM_UI_FLAG_LAYOUT_STABLE
                | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
                | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
                | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
                | View.SYSTEM_UI_FLAG_FULLSCREEN
                | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
            );
        });
    }

    @Override
    public void onWindowFocusChanged(boolean hasFocus) {
        super.onWindowFocusChanged(hasFocus);
        if (hasFocus) {
            setImmersive();
        }
    }

    @Override
    public void onBackPressed() {
        if (isControllerActive) {
            new AlertDialog.Builder(this)
                .setTitle("Disconnect")
                .setMessage("Do you want to disconnect and return to the connection lobby?")
                .setPositiveButton("Disconnect", (d, w) -> loadLobby())
                .setNegativeButton("Cancel", null)
                .show();
        } else {
            super.onBackPressed();
        }
    }

    @Override
    protected void onDestroy() {
        stopAutoDiscovery();
        if (multicastLock != null && multicastLock.isHeld()) {
            try { multicastLock.release(); } catch (Exception ignored) {}
        }
        if (webView != null) {
            webView.destroy();
        }
        super.onDestroy();
    }

    // JavaScript Interface exposed to WebView
    public class NativeInterface {
        @JavascriptInterface
        public void connect(String ip, String port, int slot) {
            stopAutoDiscovery();
            isControllerActive = true;

            // Save preferences
            prefs.edit()
                .putString(KEY_LAST_HOST, ip)
                .putInt(KEY_LAST_SLOT, slot)
                .apply();

            String targetUrl = "http://" + ip + ":" + port + "/controller?slot=" + slot;
            mainHandler.post(() -> {
                webView.loadUrl(targetUrl);
            });
        }

        @JavascriptInterface
        public void vibrate(long durationMs, int amplitude) {
            if (vibrator != null && vibrator.hasVibrator()) {
                try {
                    int clampedAmp = Math.max(1, Math.min(255, amplitude));
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                        vibrator.vibrate(VibrationEffect.createOneShot(Math.max(10, durationMs), clampedAmp));
                    } else {
                        vibrator.vibrate(durationMs);
                    }
                } catch (Exception ignored) {}
            }
        }

        @JavascriptInterface
        public String getLastHost() {
            return prefs.getString(KEY_LAST_HOST, "");
        }

        @JavascriptInterface
        public int getLastSlot() {
            return prefs.getInt(KEY_LAST_SLOT, 1);
        }

        @JavascriptInterface
        public void startAutoDiscovery() {
            MainActivity.this.startAutoDiscovery();
        }
    }
}
