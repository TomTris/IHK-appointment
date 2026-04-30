# IHK Heilbronn §34a Appointment Watcher

A Go program that monitors available appointment slots for the IHK Heilbronn Sachkundeprüfung §34a (security guard certification exam).

## Features

- Monitors the IHK Heilbronn API for available exam slots
- Alerts when slots become available within specified date ranges
- Generates markdown reports of current appointments
- macOS notifications with sound alerts
- Runs continuously with configurable polling intervals
- Ideal for early-appointment seekers who want to book immediately without checking the site constantly

Due to alerting and because you should book the appointment asap, this script should be executed locally.

## Installation

### Prerequisites

- Go 1.21 or later
- Mac (Not sure with other OS)

### Build from source

```bash
git clone https://github.com/tomtris/ihk-appointment-watcher.git
dir ihk-appointment-watcher
make build
```

Or build directly with Go:

```bash
go build -o bin/ihk-watcher ./cmd/ihk-watcher
```

## Usage

```bash
# run the built binary from the bin directory
./bin/ihk-watcher -interval 5m -alarm 2026-06-30
```

If you prefer not to build first, use Go directly:

```bash
go run ./cmd/ihk-watcher -interval 5m -alarm 2026-06-30
```

This is especially useful if you want an early appointment and want to be notified immediately when a slot opens, without having to refresh the website hundreds of times a day.

### Command Line Flags

- `-once`: Run once and exit (default: false)
- `-interval duration`: Poll interval (default: 5m)
- `-alarm date`: Alert date format:
  - Single date: `2026-06-30` or `2026.06.30` (alert for slots on or before this date)
  - Date range: `2026.04.01-2026.06.01` (alert for slots between these dates)

### Examples

```bash
# Check once for slots up to June 30th
./ihk-watcher -once -alarm 2026-06-30

# Monitor continuously every 3 minutes for slots up to June 30th
./ihk-watcher -interval 3m -alarm 2026-06-30

# Monitor for slots between April 1st and June 1st
./ihk-watcher -alarm 2026.04.01-2026.06.01
```

## How it works

The program makes direct HTTP requests to the IHK's API endpoint, bypassing the web interface. It parses the JSON response to extract appointment information and checks for available slots within the specified criteria.

## Output

- Console output with appointment details
- `termine.md`: Markdown table of current appointments
- `appointments.log`: Log file with historical data
- macOS notifications when slots become available

## API Endpoint

The program uses this public API endpoint:
```
GET https://eoa2.bildung1.gfi.ihk.de/fb/api/Elvis/heilbronn-franken/Pruefung/2580270/Variante/85482017/Durchfuehrungen?anmeldungMode=SINGLE
```

## License

MIT License - see LICENSE file for details.

## Disclaimer

This tool is for personal use only. Please respect the IHK's terms of service and avoid excessive API requests.