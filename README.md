# beacon-client

Go client for [beacon](https://github.com/redpotatolabs/beacon), the Red Potato Labs Discord bot that posts structured error and deploy events to a channel. The module also holds the wire types that the beacon server and its clients share, so a renamed field is a compile error on both sides.

Standard library only. Safe for concurrent use; build one `Client` and share it.

## Install

```sh
go get github.com/redpotatolabs/beacon-client@latest
```

## Use

```go
import "github.com/redpotatolabs/beacon-client"

c := beacon.New("https://beacon.example", token)

// Post an error event.
_, err := c.Error(ctx, beacon.ErrorRequest{
    Title:   "Nightly import failed",
    Message: err.Error(),
    Fields:  []beacon.FieldKV{{Name: "rows", Value: "1204", Inline: true}},
})

// Or the short form.
_, err = c.Errorf(ctx, "import failed: %v", err)

// Post a deploy event from a GitHub Actions run step.
_, err = c.DeployFromGitHubEnv(ctx, "success")
```

A non-2xx response comes back as an `*APIError` with the HTTP status and the server's `error` message, so callers can check it with `errors.As`.

## What ships

| Symbol | Purpose |
| --- | --- |
| `New(baseURL, token)` | build a `*Client` with a 10 second timeout |
| `Client.Error(ctx, ErrorRequest)` | POST `/v1/errors` |
| `Client.Errorf(ctx, format, args...)` | error event from a format string |
| `Client.Deploy(ctx, DeployRequest)` | POST `/v1/deploys` |
| `Client.DeployFromGitHubEnv(ctx, status)` | deploy event built from the `GITHUB_*` variables |
| `ErrorRequest`, `DeployRequest`, `Response`, `FieldKV` | wire types shared with the server |
| `APIError` | non-2xx response as an error |

## License

MIT, see [LICENSE](LICENSE).
