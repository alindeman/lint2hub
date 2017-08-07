# lint2hub

[![GoDoc](https://godoc.org/github.com/alindeman/lint2hub?status.png)](https://godoc.org/github.com/alindeman/lint2hub)

**lint2hub** automates creating GitHub pull request review comments in response to code linters. While [other](https://github.com/jenkinsci/violation-comments-to-github-plugin) [solutions](https://github.com/dgraph-io/lint) [exist](https://linthub.io/), **lint2hub** is both free software and not tied to any specific linter.

## Usage

**lint2hub** is both a command line tool and a [go client](https://godoc.org/github.com/alindeman/lint2hub) for commenting on pull requests diffs.

```bash
export LINT2HUB_GITHUB_ACCESS_TOKEN="abc123"
lint2hub -owner alindeman \
  -repo lint2hub \
  -pull-request 1234 \
  -sha "adc83b19e793491b1c6ea0fd8b46cd9f32e592fc" \
  -file "foo.go" \
  -line 8 \
  -body "exported method Foo.Bar should have comment or be unexported"
```

**lint2hub** can also accept linter output from standard input, parsed with a custom [regular expression](https://godoc.org/regexp) to extract file, line and comment body. For instance, to pipe [gometalinter](https://github.com/alecthomas/gometalinter):

```bash
gometalinter ./... |
  lint2hub -owner alindeman \
    -repo lint2hub \
    -pull-request 1234 \
    -sha "adc83b19e793491b1c6ea0fd8b46cd9f32e592fc" \
    -pattern '^(?P<file>[^:]+):(?P<line>[\d]+):(?P<column>\d*): (?P<body>.*)$'
```
