# gjevt

    go install github.com/leftmike/gjevt@latest

Calls [Jev](https://openrouter.ai/typesafe) through OpenRouter's decisions API.
On first run it creates `./gjevt.hcl`; set `api_key` there.

    gjevt questions.json [state.json...]

Each state file is sent with the questions as a separate request; with no state
files, the state is read from stdin. For example:

    gjevt examples/phishing_questions.json examples/phishing_suspended.json examples/phishing_newsletter.json
