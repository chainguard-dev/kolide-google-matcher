# googkol

Find & warn on mismatches in Google/Kolide records

## Usage

Gather the current list of Desktop machines according to Google:

1. Visit [https://admin.google.com/ac/devices/list?default=true&category=desktop](Google Admin: Mobile Devices)
2. Click the Download button (top right)
3. Select All columns
4. Select Comma-separated values (.csv)
5. Click "Download CSV"

```shell
export KOLIDE_API_KEY="..."
export SLACK_WEBHOOK_URL="..."

go run . --endpoints-csv=</path/to/csv>
```
