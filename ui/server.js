const express = require("express");
const http = require("http");
const https = require("https");
const net = require("net");
const tls = require("tls");

const app = express();
const PORT = process.env.PORT || 3000;
const API_BASE = process.env.API_BASE || "http://localhost:8080";
const UI_STATIC_DIR = process.env.UI_STATIC_DIR || "public";
const target = new URL(API_BASE);
const client = target.protocol === "https:" ? https : http;

app.get("/login", (req, res) => {
  res.redirect("/login.html");
});

app.get("/register", (req, res) => {
  res.redirect("/register.html");
});

app.get("/dashboard", (req, res) => {
  res.redirect("/dashboard.html");
});

app.get("/editor", (req, res) => {
  res.redirect("/editor.html");
});

app.use(express.static(UI_STATIC_DIR));

function proxyRequest(req, res) {
  const options = {
    protocol: target.protocol,
    hostname: target.hostname,
    port: target.port,
    method: req.method,
    path: req.originalUrl,
    headers: {
      ...req.headers,
      host: target.host,
    },
  };

  const proxy = client.request(options, (proxyRes) => {
    res.writeHead(proxyRes.statusCode || 500, proxyRes.headers);
    proxyRes.pipe(res);
  });

  proxy.on("error", () => {
    res.status(502).send("Bad Gateway");
  });

  req.pipe(proxy);
}

app.use("/api", proxyRequest);
app.use("/uploads", proxyRequest);

const server = app.listen(PORT, () => {
  console.log(`UI running on http://localhost:${PORT}`);
  console.log(`Serving static files from ${UI_STATIC_DIR}`);
  console.log(`Proxying to ${API_BASE}`);
});

server.on("upgrade", (req, socket, head) => {
  if (!req.url || !req.url.startsWith("/ws/")) {
    socket.destroy();
    return;
  }

  const port =
    target.port || (target.protocol === "https:" ? "443" : "80");
  const connect =
    target.protocol === "https:"
      ? tls.connect
      : net.connect;
  const upstream = connect(port, target.hostname, () => {
    const forwardedHeaders = Object.entries(req.headers)
      .filter(([key]) => key.toLowerCase() !== "host")
      .map(([key, value]) => `${key}: ${value}`);
    const headerLines = [
      `GET ${req.url} HTTP/1.1`,
      `Host: ${target.host}`,
      ...forwardedHeaders,
      "",
      "",
    ];
    upstream.write(headerLines.join("\r\n"));
    if (head && head.length > 0) {
      upstream.write(head);
    }
  });

  upstream.on("error", () => {
    socket.destroy();
  });
  socket.on("error", () => {
    upstream.destroy();
  });

  upstream.pipe(socket);
  socket.pipe(upstream);
});
