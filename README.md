# tghbot
Simple Github events notification bot


## Installation 
Install bot using `go get`

```bash
go get github.com/tdakkota/tghbot/cmd/tghbot
```

[Create Telegram application and get `api_id` and `api_hash `](https://core.telegram.org/api/obtaining_api_id).

Set `APP_ID`, `APP_HASH` ,`BOT_TOKEN` and `GITHUB_TOKEN` environment variables and run(it needs to be `$GOPATH/bin` in your `$PATH`)
```bash
tghbot run
```
