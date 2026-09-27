# gjevt

    go install github.com/leftmike/gjevt@latest

Calls [Jev](https://openrouter.ai/typesafe) through OpenRouter's decisions API.
On first run it creates `./gjevt.hcl`; set `api_key` there.

    gjevt [-json] [-skip] questions.json [state...]

Each state file is sent with the questions as a separate request; with no state
files, the state is read from stdin. A state that starts with `{` is JSON;
anything else is sent as plain text. Use `-json` to print the full responses as
JSON. A state that is too long for Jev is truncated, or skipped with `-skip`.
For example:

    gjevt examples/phishing_questions.json examples/phishing_suspended.json examples/phishing_newsletter.json

## Library

The `jev` package can be used directly:

    import "github.com/leftmike/gjevt/jev"

    c := jev.NewClient(jev.Config{APIKey: key})
    resp, truncated, err := c.DecideTruncated(ctx, jev.DecisionRequest{
        State:     map[string]any{"ticket": "I was charged twice"},
        Questions: map[string]jev.Question{
            "angry": {Type: "noul", Instructions: "Is the customer angry?"},
        },
    })

Use `Decide` instead of `DecideTruncated` to get `jev.ErrTooLong` rather than
truncating state that is too long. `jev.WithHTTPClient` sets a custom
`*http.Client`.
