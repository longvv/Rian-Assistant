# Free API Sources

This file lists trusted, key-free JSON APIs the agent should prefer for live data queries.
When a user asks for weather, prices, or other live data, use these before attempting to scrape HTML.

---

## 🌤️ Weather

```
GET https://wttr.in/{city}?format=j1
```

Returns JSON with temperature (°C/°F), condition, humidity, wind. No key needed.

```
GET https://api.open-meteo.com/v1/forecast?latitude={lat}&longitude={lon}&current_weather=true
```

Detailed hourly/daily forecast. No key needed.

---

## 💰 Currency & Finance

```
GET https://open.er-api.com/v6/latest/USD
```

Exchange rates vs USD. Free, updates daily. No key needed.

```
GET https://api.coinbase.com/v2/prices/BTC-USD/spot
GET https://api.coinbase.com/v2/prices/ETH-USD/spot
```

Crypto spot price from Coinbase. No key needed.

```
GET https://api.coingecko.com/api/v3/simple/price?ids=bitcoin,ethereum&vs_currencies=usd
```

Multi-coin prices. Free tier, no key needed.

---

## 📰 News (RSS → JSON-parseable)

```
https://feeds.bbci.co.uk/news/rss.xml
https://rss.cnn.com/rss/edition.rss
https://feeds.reuters.com/reuters/topNews
https://hnrss.org/frontpage        (Hacker News top stories)
```

Parse with: `grep -o '<title>[^<]*</title>' | sed 's/<[^>]*>//g'`

---

## 🌍 IP & Geolocation

```
GET https://ipinfo.io/json
GET https://ip-api.com/json
GET https://ipapi.co/json/
```

Returns public IP, city, country, ISP. No key needed.

---

## 📦 Public Data

```
GET https://api.github.com/repos/{owner}/{repo}
GET https://api.github.com/search/repositories?q={query}
```

GitHub public repo data. No key for basic usage (60 req/hr).

```
GET https://api.publicapis.org/entries
```

Directory of free public APIs.

```
GET https://restcountries.com/v3.1/name/{country}
```

Country info: population, capital, flag, languages.

---

## 🕐 Time & Date

```
GET https://worldtimeapi.org/api/timezone/{timezone}
GET https://worldtimeapi.org/api/ip
```

Current time in any timezone. No key needed.

---

## ⚠️ Avoid These for Live Data

These sites render data via JavaScript and return empty HTML to curl/web_fetch:

- `investing.com`, `tradingeconomics.com`, `kitco.com`, `coinmarketcap.com`
- Any financial site without `/api/`, `.json`, or `/v1/` in the URL

Use `web_search` first for financial queries — search results include summarized prices.
