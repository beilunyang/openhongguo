package com.openhongguo.signer;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import java.io.*;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.Map;

public class SignerServer {

    static final String SERVICE = "openhongguo-signer";

    private static MetasecEmulator emu;
    /** When set, /feature-hash requires "Authorization: Bearer <token>". */
    private static String token;
    private static final int MAX_FAILURES = Integer.parseInt(envOr("SIGNER_MAX_FAILURES", "3"));
    private static int failures;

    private static void sendError(HttpExchange exchange, int status, String message) throws IOException {
        sendJson(exchange, status, "{\"error\":\"" + SERVICE + ": " + escapeJson(message) + "\"}");
    }

    private static String envOr(String key, String def) {
        String v = System.getenv(key);
        return v == null || v.isEmpty() ? def : v;
    }

    private static boolean authorized(HttpExchange exchange) {
        if (token == null) return true;
        String auth = exchange.getRequestHeaders().getFirst("Authorization");
        if (auth == null || !auth.startsWith("Bearer ")) return false;
        return java.security.MessageDigest.isEqual(
                auth.substring(7).trim().getBytes(StandardCharsets.UTF_8),
                token.getBytes(StandardCharsets.UTF_8));
    }

    public static void main(String[] args) throws Exception {
        // Redirect stdin to /dev/null so unidbg's debugger doesn't block on interactive input
        System.setIn(new ByteArrayInputStream(new byte[0]));

        int port = Integer.parseInt(envOr("SIGNER_PORT", "19830"));
        String host = envOr("SIGNER_HOST", "127.0.0.1");
        String soPath = null;
        String configPath = null;

        for (int i = 0; i < args.length; i++) {
            switch (args[i]) {
                case "--port":
                    port = Integer.parseInt(args[++i]);
                    break;
                case "--host":
                    host = args[++i];
                    break;
                case "--so":
                    soPath = args[++i];
                    break;
                case "--config":
                    configPath = args[++i];
                    break;
            }
        }

        File soFile;
        if (soPath != null) {
            soFile = new File(soPath);
        } else {
            soFile = extractResource("/lib/arm64-v8a/libmetasec_ml.so");
        }

        if (!soFile.exists()) {
            System.err.println("[openhongguo-signer] .so file not found: " + soFile);
            System.exit(1);
        }

        String nativeConfig;
        if (configPath != null) {
            nativeConfig = new String(java.nio.file.Files.readAllBytes(
                    java.nio.file.Paths.get(configPath)), StandardCharsets.UTF_8).trim();
        } else {
            nativeConfig = readResource("/native-config.txt");
        }

        System.out.println("[openhongguo-signer] loading " + soFile.getAbsolutePath());
        emu = new MetasecEmulator(soFile);

        System.out.println("[openhongguo-signer] initializing SDK...");
        emu.initSdk(nativeConfig);
        System.out.println("[openhongguo-signer] SDK handle = " + emu.getSdkHandle());

        emu.waitForInit();

        token = System.getenv("SIGNER_TOKEN");
        if (token != null && token.isEmpty()) token = null;
        if (token == null && !"127.0.0.1".equals(host) && !"localhost".equals(host)) {
            System.out.println("[openhongguo-signer] no SIGNER_TOKEN: anyone who can reach " + host + ":" + port + " can sign requests");
        }
        HttpServer server = HttpServer.create(new InetSocketAddress(host, port), 0);
        server.createContext("/ready", new ReadyHandler());
        server.createContext("/feature-hash", new FeatureHashHandler());
        server.setExecutor(null);
        server.start();

        System.out.println("[openhongguo-signer] HTTP server listening on " + host + ":" + port
                + (token != null ? " (token required)" : ""));
        Thread.currentThread().join();
    }

    private static String readResource(String path) throws IOException {
        try (InputStream is = SignerServer.class.getResourceAsStream(path)) {
            if (is == null) {
                File f = new File("src/main/resources" + path);
                return new String(java.nio.file.Files.readAllBytes(f.toPath()), StandardCharsets.UTF_8).trim();
            }
            return new String(is.readAllBytes(), StandardCharsets.UTF_8).trim();
        }
    }

    private static File extractResource(String resourcePath) throws IOException {
        InputStream is = SignerServer.class.getResourceAsStream(resourcePath);
        if (is == null) {
            return new File("src/main/resources" + resourcePath);
        }
        File tmp = File.createTempFile("libmetasec_ml", ".so");
        tmp.deleteOnExit();
        try (OutputStream os = new FileOutputStream(tmp)) {
            byte[] buf = new byte[8192];
            int n;
            while ((n = is.read(buf)) != -1) {
                os.write(buf, 0, n);
            }
        }
        is.close();
        return tmp;
    }

    static class ReadyHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            if (!"GET".equals(exchange.getRequestMethod())) {
                sendError(exchange, 405, "method not allowed");
                return;
            }
            long h = emu.getSdkHandle();
            sendJson(exchange, 200, "{\"service\":\"" + SERVICE + "\",\"ready\":" + (h != 0) + "}");
        }
    }

    static class FeatureHashHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            if (!"POST".equals(exchange.getRequestMethod())) {
                sendError(exchange, 405, "method not allowed");
                return;
            }

            if (!authorized(exchange)) {
                sendError(exchange, 401, "unauthorized");
                return;
            }
            String reqBody = new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
            String url = extractJsonString(reqBody, "url");
            String bodyHex = extractJsonString(reqBody, "body_hex");

            byte[] body = null;
            if (bodyHex != null && !bodyHex.isEmpty()) {
                body = hexToBytes(bodyHex);
            }

            try {
                if (url == null || url.isEmpty()) {
                    sendError(exchange, 400, "url is required");
                    return;
                }
                Map<String, String> headers;
                // One emulator, one CPU context: sign one request at a time.
                synchronized (SignerServer.class) {
                    headers = emu.getFeatureHash(url, body);
                    String neptune = headers.get("X-Neptune");
                    if (headers.containsKey("X-Medusa")) {
                        failures = 0;
                    } else if (neptune != null && !neptune.startsWith("-8")) {
                        // The SDK refused this input (e.g. -7 for a bad URL).
                        // That says nothing about the emulator's health.
                        String code = neptune.split("\\|", 2)[0];
                        sendJson(exchange, 422, "{\"error\":\"" + SERVICE + ": signer refused the request\",\"neptune\":\"" + escapeJson(code) + "\"}");
                        return;
                    } else if (++failures >= MAX_FAILURES) {
                        // The emulator state is likely wedged. Exit so the
                        // container's restart policy brings up a fresh one.
                        System.out.println("[openhongguo-signer] " + failures + " signing failures in a row, exiting");
                        sendError(exchange, 503, "signer restarting");
                        System.exit(1);
                    }
                }
                StringBuilder sb = new StringBuilder("{\"headers\":{");
                boolean first = true;
                for (Map.Entry<String, String> e : headers.entrySet()) {
                    if (!first) sb.append(",");
                    sb.append("\"").append(escapeJson(e.getKey())).append("\":\"")
                      .append(escapeJson(e.getValue())).append("\"");
                    first = false;
                }
                sb.append("}}");
                sendJson(exchange, 200, sb.toString());
            } catch (Exception e) {
                e.printStackTrace();
                sendError(exchange, 500, "internal error: " + e.getMessage());
            }
        }
    }

    private static void sendJson(HttpExchange exchange, int code, String json) throws IOException {
        byte[] bytes = json.getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        exchange.getResponseHeaders().set("Server", SERVICE);
        exchange.sendResponseHeaders(code, bytes.length);
        exchange.getResponseBody().write(bytes);
        exchange.getResponseBody().close();
    }

    private static String extractJsonString(String json, String key) {
        String pattern = "\"" + key + "\"";
        int idx = json.indexOf(pattern);
        if (idx < 0) return null;
        idx = json.indexOf(":", idx + pattern.length());
        if (idx < 0) return null;
        idx++;
        while (idx < json.length() && json.charAt(idx) == ' ') idx++;
        if (idx >= json.length()) return null;
        if (json.charAt(idx) == 'n') return null;
        if (json.charAt(idx) != '"') return null;
        idx++;
        StringBuilder sb = new StringBuilder();
        while (idx < json.length() && json.charAt(idx) != '"') {
            char c = json.charAt(idx);
            if (c == '\\' && idx + 1 < json.length()) {
                char e = json.charAt(++idx);
                switch (e) {
                    case 'n': sb.append('\n'); break;
                    case 't': sb.append('\t'); break;
                    case 'r': sb.append('\r'); break;
                    case 'b': sb.append('\b'); break;
                    case 'f': sb.append('\f'); break;
                    case 'u':
                        if (idx + 4 < json.length()) {
                            sb.append((char) Integer.parseInt(json.substring(idx + 1, idx + 5), 16));
                            idx += 4;
                        }
                        break;
                    default: sb.append(e);
                }
            } else {
                sb.append(c);
            }
            idx++;
        }
        return sb.toString();
    }

    private static byte[] hexToBytes(String hex) {
        int len = hex.length();
        byte[] data = new byte[len / 2];
        for (int i = 0; i < len; i += 2) {
            data[i / 2] = (byte) ((Character.digit(hex.charAt(i), 16) << 4)
                    + Character.digit(hex.charAt(i + 1), 16));
        }
        return data;
    }

    private static String escapeJson(String s) {
        if (s == null) return "null";
        return s.replace("\\", "\\\\").replace("\"", "\\\"")
                .replace("\n", "\\n").replace("\r", "\\r");
    }
}
