# Gmail automation

## Boring setup

Follow [these quickstart steps](https://developers.google.com/workspace/gmail/api/quickstart/go) to enable the Gmail API, configure the OAuth Consent Screen for this application in the Google API Console, and authorise credentials.

This last step is important: It creates a credentials.json file, which should be kept in this directory (and is gitignored per the `*.json` entry).

## Send mail

Copy .env.example to .env and populate like:
```
RECIPIENTS=user1@example.com,user2@example.com
SUBJECT="deez nuts"
MESSAGE="deez nuts"
```

Run with `go run main.go`

When prompted, visit the link in your browser and give the application the request access.
It will cache credentials in token.json and send an email to all addresses in .env in a comma-separated list

## Attachments

Accepts one flag `--attachments` as a list of space-delimited filenames in current working directory

e.g. use send-encrypted.sh to convert all markdown files in current working directory (skips README.md) to pdf, encrypt them, and run main.go with `--attachments` option to attach all encrypted PDFs. In the environment set `USER_PASS`/`OWNER_PASS`. Requires: pandoc, qpdf
