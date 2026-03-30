# IHK Heilbronn – Sachkundeprüfung §34a Watcher

## How to use

```bash
go run ihk-34a-heilbronn.go -interval <time> -alarm <date>
```

### `-alarm` formats

| Format | Behaviour |
|--------|-----------|
| `2026-06-30` or `2026.06.30` | Alert for any bookable slot **on or before** that date |
| `2026.04.01-2026.06.01` | Alert for any bookable slot **between** those two dates (inclusive) |

### Examples

```bash
# All slots up to June 30th (dashes or dots both work)
go run ihk-34a-heilbronn.go -interval 3m -alarm 2026-06-30
go run ihk-34a-heilbronn.go -interval 3m -alarm 2026.06.30

# Only slots between April 1st and June 1st
go run ihk-34a-heilbronn.go -interval 5m -alarm 2026.04.01-2026.06.01
```

---

## How the script works

The script is dead simple — one HTTP GET, no browser:

1. **Calls a single API endpoint directly:**
   ```
   GET eoa2.bildung1.gfi.ihk.de/fb/api/Elvis/heilbronn-franken/Pruefung/2580270/Variante/85482017/Durchfuehrungen?anmeldungMode=SINGLE
   ```

2. **Parses the JSON response** into typed Go structs — each `Durchfuehrung` has the exam date, free slots, deadline, oral exam info, etc.

3. **Filters past dates out** (converts UTC timestamps to local time — the API stores dates as the evening before in UTC, e.g. `2026-07-15T22:00:00Z` = `16.07.2026 CEST`)

4. **Checks the alarm window** — alerts only for bookable slots (`anmeldungMoeglich = true`, `freiePlaetze > 0`) that fall within the requested date or range.

5. **Prints a table** to console + writes `termine.md`, and only reports when something actually changes (new date or slot count shifts).

6. **Loops on a timer**, polling every N minutes.

---

## How we found the endpoint — step by step

**Step 1 — Tried to fetch the IHK page directly**
The original URL returned a 403. The form was described as "JavaScript embedded", so plain HTML scraping was never going to work.

**Step 2 — Launched headless Chrome on the outer IHK page**
The form was inside an `<iframe>`. JavaScript's same-origin policy blocked us from reading the iframe's DOM — we could see that it existed but not what was inside it.

**Step 3 — Found the iframe's `src` URL in the network log**
By capturing all network requests while the outer IHK page loaded, we saw:
```
IFRAME 0: https://eoa2.bildung1.gfi.ihk.de/kammer/heilbronn-franken/anmeldung/BGP
```
That's the real app — an Angular SPA served by `eoa2.bildung1.gfi.ihk.de` (GFI software, used by many IHKs).

**Step 4 — Navigated directly to the iframe app**
By pointing the browser straight at that URL, we bypassed the cross-origin restriction entirely. Now we could interact with the Angular app and capture its API calls.

**Step 5 — Clicked the radio button properly**
Angular ignores JS `.click()` — it needs real synthesized mouse events. Using `chromedp.Click()` (which dispatches actual browser mouse events) finally got the radio selected.

**Step 6 — Captured the network request that fired on radio click**
After selecting "Gesamtprüfung", three new API calls appeared. The critical one:
```
GET /fb/api/Elvis/heilbronn-franken/Pruefung/2580270/Variante/85482017/Durchfuehrungen?anmeldungMode=SINGLE
```
*"Durchführungen" = exam instances/dates.* This fired on radio selection, before even clicking "Weiter".

**Step 7 — Verified the endpoint returns open JSON**
`-raw` showed a clean JSON array with every field we needed — no auth token, no session cookie, no CSRF header required. The server doesn't protect this data at all.

**Step 8 — Threw away chromedp entirely**
Once we had the URL, the browser was useless overhead. A plain `http.Get` with a realistic `User-Agent` header was sufficient.

> The key insight was that the IHK website is just a shell — the real app lives at `eoa2.bildung1.gfi.ihk.de` and its API is completely open. The iframe wrapper and JavaScript loading were just UX decoration around a standard REST backend.

---

*Quelle: [IHK Heilbronn-Franken Anmeldeformular](https://www.ihk.de/heilbronn-franken/produktmarken/branchen/gewerbeportal/bewachungsgewerbe/anmeldeformular-fuer-die-sachkundepruefung-fuer-besondere-bewachungstaetigkeiten-nach-34a-gewo-6050278)*