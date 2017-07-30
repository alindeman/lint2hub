# lint2hub

**lint2hub** automates creating GitHub pull request review comments in response to code linters. While [other](https://github.com/jenkinsci/violation-comments-to-github-plugin) [solutions](https://github.com/dgraph-io/lint) [exist](https://linthub.io/), **lint2hub** is both free software and not tied to any specific linter.

## Usage

```bash
export LINE2HUB_GITHUB_ACCESS_TOKEN="abc123"
lint2hub -owner alindeman \
  -repo lint2hub \
  -pull-request 1234 \
  -sha "adc83b19e793491b1c6ea0fd8b46cd9f32e592fc" \
  -file "foo.go" \
  -line 8 \
  -body "exported method Foo.Bar should have comment or be unexported"
```

Because **lint2hub** just accepts arguments at the command line, it can be hooked up to most any linter with a bit of scripting:

```bash
#!/usr/bin/env bash
set -euxo pipefail

# foo.go:25:1: exported method Foo.Bar should have comment or be unexported
while read -r line; do
  if [[ $line =~ ^(.+):([0-9]+):([0-9]+):\ (.*)$ ]]; then
    file="${BASH_REMATCH[1]}"
    line="${BASH_REMATCH[2]}"
    comment="${BASH_REMATCH[4]}"

    lint2hub -owner "$OWNER" \
      -repo "$REPO" \
      -pull-request "$PULL_REQUEST" \
      -sha "$SHA" \
      -file "$file" \
      -line "$line" \
      -body "$comment"
  fi
done < <(golint .)
```

### Batch Mode

**lint2hub** makes several API requests to the GitHub API. To minimize the number of duplicate requests sent for a single pull request, **lint2hub** offers a *batch mode* where multiple comments are posted in a single invocation.

In batch mode **lint2hub** reads the following format from standard input:

```
filename<tab>line number<tab>comment
filename<tab>line number<tab>comment
...
```

For example:

```
printf "foo.go\t8\texported method Foo.Bar should have comment or be unexported
bar.go\t12\texported method Bar.Foo should have comment or be unexported" | \
  lint2hub -owner "$OWNER" \
    -repo "$REPO" \
    -pull-request "$PULL_REQUEST" \
    -sha "$SHA" \
    -batch
```

To post a multi-line comment in batch mdoe, replace newline characters with `<br>`.
