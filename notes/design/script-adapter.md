# Script adapter

The `script` adapter runs a shell command on a runner and captures declared outputs for later interpolation.

## Manifest shape

```yaml
components:
  create-token:
    type: script
    runner: local
    source:
      embedded: |
        echo "token=abc" >> "$ORK_OUTPUT_ENV"
        jq -n --arg url "http://localhost:8080" '{api_url: $url}' > "$ORK_OUTPUT_JSON"
    outputs:
      - name: token
      - name: api_url
        required: false
    config:
      shell: ["bash"]
```

Config:

- `shell`: optional interpreter prefix, defaults to `["sh"]`

The adapter supports embedded source and file sources. Source files are copied to the component workdir before they run.

Script execution comes from source:

- embedded source runs as one staged `script.sh`
- file sources run sequentially in manifest order
- `with` files are copied into the runner workdir as supporting context and are not executed

For script source files, the script path is appended to the shell prefix. For example, `shell: ["bash"]` runs as `bash ./script.sh`.

Scripts run source files, not inline command strings, so the default does not include `-c`:

```text
["sh"] + "./script.sh" -> sh ./script.sh
```

## Outputs

ork sets two managed environment variables for scripts:

- `ORK_OUTPUT_ENV`
- `ORK_OUTPUT_JSON`

The script can append dotenv-style values:

```sh
echo "token=abc" >> "$ORK_OUTPUT_ENV"
echo "api_url=http://localhost:8080" >> "$ORK_OUTPUT_ENV"
```

It can also write JSON:

```sh
cat > "$ORK_OUTPUT_JSON" <<'JSON'
{
  "token": "abc",
  "api_url": "http://localhost:8080"
}
JSON
```

If both files are present, ork reads dotenv first and JSON second. JSON wins when both files contain the same output key.

JSON output values may be strings, numbers, booleans, or null. Objects and arrays are rejected until output typing exists.

Because ork interpolation uses `${...}`, prefer plain shell variables such as `$TOKEN` inside manifest commands when you want the runner shell to expand them.

## Output schema

Components declare outputs as a list of objects:

```yaml
outputs:
  - name: token
    required: true
    sensitive: true
    type: string
  - name: api_url
    required: false
```

`name` is required. `required` defaults to `true`.

After every component apply, ork enforces the output schema:

- missing required outputs fail the apply
- missing optional outputs are allowed
- extra adapter-produced outputs warn and are ignored for interpolation
- sensitive outputs are available for interpolation during the current run but are dropped from state

The schema applies to every adapter, not only `script`.
