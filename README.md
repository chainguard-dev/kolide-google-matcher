# kolide-google-matcher

Find machines that have recently logged into a Google Workspace, but lack a matching Kolide agent.

## Installation

```shell
go install github.com/chainguard-dev/kolide-google-matcher@latest
```

## Example Output

```log
2022/11/07 11:35:44 Kolide: found 60 devices
2022/11/07 11:35:44 Google: found 58 devices
2022/11/07 11:35:44 inky@chainguard.dev mismatch: Google sees 2 macOS devices, Kolide sees 1
Google:
  Inky's MacBookPro18,2 (MacOS 13.0.0)                        [Sep 2, 2022 to Nov 7, 2022]
  Inky's Mac (macOS 10.15.7)                                  [Sep 2, 2022 to Nov 6, 2022]
Kolide:
  Inkys-MacBook-Pro-2 (macOS 13.0 Ventura Build: 22A380)      [Sep 5, 2022 to Nov 7, 2022]

2022/11/07 11:35:44 wolfi@chainguard.dev mismatch: Google sees 1 devices, Kolide sees 0!
Google:
  Wolfi's Mac (macOS 10.15.7)                                 [Nov 4, 2022 to Nov 4, 2022]

2022/11/07 11:35:44 found 2 total mismatches
```

## Requirements

* A Kolide API key [https://k2.kolide.com/3361/settings/admin/developers/api_keys](Create a New Key)
* Access to a Google Workspace admin console
* The Go Programming Language

## Usage

Gather the current list of Desktop machines according to Google:

1. Visit [https://admin.google.com/ac/devices/list?default=true&category=desktop](Google Admin: Mobile Devices)
2. Click the Download button (top right)
3. Select All columns
4. Select Comma-separated values (.csv)
5. Click "Download CSV"

Then you can run the `kolide-google-matcher`:

```shell
export KOLIDE_API_KEY="..."
kolide-google-matcher --endpoints-csv=</path/to/csv>
```
