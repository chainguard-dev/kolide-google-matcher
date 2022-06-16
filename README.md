# googkol

Find &amp; warn on mismatches in Google/Kolide records

## usage

`go run . --kolide-csv ~/Downloads/kolide.csv --google-csv ~/Downloads/devices.csv`

If the environment variable `SLACK_WEBHOOK_URL` is provided, the output of this
run will be posted to the provided Slack webhook URL.
