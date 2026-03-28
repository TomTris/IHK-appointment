// IHK Heilbronn – Sachkundeprüfung §34a Watcher
// No dependencies beyond the Go standard library.
//
//   go mod init ihk-watcher
//   go run ihk-34a-heilbronn.go
//
// Flags:
//   -once              run once and exit
//   -interval 5m       poll interval (default 5m)
//   -alarm 2026-06-30  alert if any slot is on or before this date (default: last day of June 2026)

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const apiURL = "https://eoa2.bildung1.gfi.ihk.de/fb/api/Elvis/heilbronn-franken/Pruefung/2580270/Variante/85482017/Durchfuehrungen?anmeldungMode=SINGLE"

var (
	once      = flag.Bool("once", false, "run once and exit")
	interval  = flag.Duration("interval", 5*time.Minute, "poll interval")
	alarmDate = flag.String("alarm", "2026-06-30", "alert if any open slot falls on or before this date (YYYY-MM-DD)")
)

type Standort struct {
	Name    string `json:"name"`
	Strasse string `json:"strasse"`
	Hausnr  string `json:"hausnummer"`
	PLZ     string `json:"plz"`
	Ort     string `json:"ort"`
}

type Durchfuehrung struct {
	ID                    int      `json:"id"`
	Name                  string   `json:"name"`
	Datum                 string   `json:"datum"`
	Anmeldefrist          string   `json:"anmeldefrist"`
	AnmeldungMoeglich     bool     `json:"anmeldungMoeglich"`
	FreiePlaetze          int      `json:"freiePlaetze"`
	MaxTeilnehmer         int      `json:"maximaleTeilnehmer"`
	AngemeldeteTeilnehmer int      `json:"angemeldeteTeilnehmer"`
	Zusatzinfo            string   `json:"zusatzinfoOnlineanmeldung"`
	Standort              Standort `json:"standort"`
}

type ResponseItem struct {
	Durchfuehrungen []Durchfuehrung `json:"durchfuehrungen"`
}

func main() {
	flag.Parse()
	log.SetFlags(log.Ltime)

	threshold, err := time.ParseInLocation("2006-01-02", *alarmDate, time.Local)
	if err != nil {
		log.Fatalf("Ungültiges Alarm-Datum %q: %v", *alarmDate, err)
	}
	// include the whole alarm day
	threshold = threshold.Add(24*time.Hour - time.Second)

	log.Println("IHK Sachkundeprüfung §34a – Watcher gestartet")
	log.Printf("Alarm wenn Termin ≤ %s", threshold.Format("02.01.2006"))
	if !*once {
		log.Printf("Abfrage alle %s  (Ctrl+C zum Beenden)\n", *interval)
	}

	var prev []Durchfuehrung
	for {
		termine, err := fetchTermine()
		if err != nil {
			log.Printf("FEHLER: %v", err)
		} else {
			report(termine, prev)
			checkEarlySlots(termine, threshold)
			prev = termine
		}
		if *once {
			break
		}
		log.Printf("Nächste Abfrage in %s", *interval)
		time.Sleep(*interval)
	}
}

// checkEarlySlots fires an alert for every bookable slot on or before threshold.
func checkEarlySlots(termine []Durchfuehrung, threshold time.Time) {
	for _, d := range termine {
		if !d.AnmeldungMoeglich || d.FreiePlaetze <= 0 {
			continue
		}
		t, err := parseDate(d.Datum)
		if err != nil || t.After(threshold) {
			continue
		}

		msg := fmt.Sprintf("Früher Termin verfügbar: %s (%d Plätze frei) – Frist %s",
			d.Name, d.FreiePlaetze, frist(d.Anmeldefrist))

		log.Printf("🚨 ALARM: %s", msg)

		// Terminal bell
		fmt.Print("\a\a\a")

		// macOS system notification (no-op on other platforms)
		sendNotification("IHK §34a – Früher Termin!", msg)
	}
}

// sendNotification shows a macOS Notification Center banner and plays an
// audible alarm: a looping alert sound + text-to-speech announcement.
func sendNotification(title, body string) {
	esc := func(s string) string { return strings.ReplaceAll(s, "'", "'\\''") }

	// 1. Notification Center banner with sound
	script := fmt.Sprintf(
		`display notification "%s" with title "%s" sound name "Sosumi"`,
		esc(body), esc(title),
	)
	exec.Command("osascript", "-e", script).Run()

	// 2. Play the system alert sound 3× in a row (loud, hard to miss)
	//    /System/Library/Sounds/ contains: Basso, Blow, Bottle, Frog, Funk,
	//    Glass, Hero, Morse, Ping, Pop, Purr, Sosumi, Submarine, Tink
	go func() {
		sound := "/System/Library/Sounds/Sosumi.aiff"
		for i := 0; i < 3; i++ {
			exec.Command("afplay", sound).Run()
			time.Sleep(300 * time.Millisecond)
		}
	}()

	// 3. Text-to-speech so you hear it even in another room
	speech := fmt.Sprintf("Achtung! Früher IHK Prüfungstermin verfügbar: %s", body)
	go exec.Command("say", "-v", "Anna", speech).Run()
}

func fetchTermine() ([]Durchfuehrung, error) {
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://eoa2.bildung1.gfi.ihk.de/")
	req.Header.Set("Origin", "https://eoa2.bildung1.gfi.ihk.de")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/124 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var items []ResponseItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("JSON parse: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)
	var out []Durchfuehrung
	for _, item := range items {
		for _, d := range item.Durchfuehrungen {
			if t, err := parseDate(d.Datum); err == nil && !t.Before(today) {
				out = append(out, d)
			}
		}
	}
	return out, nil
}

func parseDate(s string) (time.Time, error) {
	for _, lay := range []string{
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(lay, s); err == nil {
			return t.Local(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable: %s", s)
}

func frist(s string) string {
	t, err := parseDate(s)
	if err != nil {
		return s
	}
	return t.Format("02.01.2006")
}

func report(curr, prev []Durchfuehrung) {
	now := time.Now().Format("15:04:05")

	if len(curr) == 0 {
		log.Printf("[%s] Keine zukünftigen Termine gefunden", now)
		return
	}

	changed := len(curr) != len(prev)
	if !changed {
		for i := range curr {
			if i >= len(prev) || curr[i].ID != prev[i].ID || curr[i].FreiePlaetze != prev[i].FreiePlaetze {
				changed = true
				break
			}
		}
	}
	if !changed {
		log.Printf("[%s] Keine Änderung – %d Termin(e) verfügbar", now, len(curr))
		return
	}

	type line struct{ console, md string }
	var lines []line
	add := func(con, md string) { lines = append(lines, line{con, md}) }

	bar := strings.Repeat("─", 76)
	add(bar, "")
	add(fmt.Sprintf("  IHK Heilbronn – Sachkundeprüfung §34a   [%s]", now),
		fmt.Sprintf("# IHK Heilbronn – Sachkundeprüfung §34a\n\n_Stand: %s_\n", now))
	add(bar, "")
	add("", "| # | Prüfungstag | Mündliche Prüfung | Plätze frei | Gesamt | Anmeldefrist | Status |")
	add("", "|---|-------------|-------------------|-------------|--------|--------------|--------|")

	for i, d := range curr {
		nameParts := strings.SplitN(d.Name, " ", 2)
		examDate := d.Name
		if len(nameParts) == 2 {
			examDate = nameParts[1]
		}
		status := "✅ buchbar"
		if !d.AnmeldungMoeglich {
			status = "⚠️ nicht möglich"
		}

		add(fmt.Sprintf("  %d.  %s", i+1, examDate), "")
		add(fmt.Sprintf("       %s", d.Zusatzinfo), "")
		add(fmt.Sprintf("       Plätze: %d frei / %d gesamt  │  Anmeldefrist: %s",
			d.FreiePlaetze, d.MaxTeilnehmer, frist(d.Anmeldefrist)), "")
		if i < len(curr)-1 {
			add("", "")
		}

		add("", fmt.Sprintf("| %d | **%s** | %s | %d | %d | %s | %s |",
			i+1, examDate, d.Zusatzinfo,
			d.FreiePlaetze, d.MaxTeilnehmer, frist(d.Anmeldefrist), status))
	}
	add(bar, "")

	for _, l := range lines {
		if l.console != "" {
			fmt.Println(l.console)
		}
	}
	fmt.Println()

	var md strings.Builder
	for _, l := range lines {
		if l.md != "" {
			md.WriteString(l.md + "\n")
		}
	}
	mdFile := "termine.md"
	os.WriteFile(mdFile, []byte(md.String()), 0644)
	log.Printf("Markdown gespeichert: %s", mdFile)

	f, err := os.OpenFile("appointments.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		fmt.Fprintf(f, "[%s] %d Termin(e)\n", time.Now().Format(time.RFC3339), len(curr))
		for _, d := range curr {
			fmt.Fprintf(f, "  %-15s  %d/%d Plätze  Frist: %s  %s\n",
				d.Name, d.FreiePlaetze, d.MaxTeilnehmer, frist(d.Anmeldefrist), d.Zusatzinfo)
		}
		fmt.Fprintln(f)
	}
}
