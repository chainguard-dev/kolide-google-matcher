# kolide-google-matcher

Find machines that have recently logged into a Google Workspace, but lack a matching Kolide agent.

## Installation

```shell
go install github.com/chainguard-dev/kolide-google-matcher@latest
```

## Example Output

```shell
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

```shell
export KOLIDE_API_KEY="..."

kolide-google-matcher --endpoints-csv=</path/to/csv>
```
