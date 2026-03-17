const express = require("express");
const http = require("http");
const https = require("https");

const app = express();
const PORT = process.env.PORT || 3000;
const API_BASE = process.env.API_BASE || "http://localhost:8080";
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

app.use(express.static("public"));

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

app.listen(PORT, () => {
  console.log(`UI running on http://localhost:${PORT}`);
  console.log(`Proxying to ${API_BASE}`);
});
