# codesnippetd

A REST API server that exposes [Universal Ctags](https://ctags.io/) tag databases over HTTP.

## Security Notice

This program has no security measures of any kind. Do not expose confidential source code through it. Ensure that only trusted hosts can connect by configuring your firewall or network access controls appropriately.

Be aware that certain queries may place a heavy load on the tag search process, which could be exploited as a denial-of-service attack vector.

## Usage

```
codesnippetd [-addr <listen-address>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-listen`, `--listen` | `:8999` | TCP address to listen on (host:port) |
| `-l` | — | Shorthand for `-listen` |
| `-port`, `--port` | — | Port number to listen on; overrides `-listen` when set |
| `-p` | — | Shorthand for `-port` |
| `--tree-sitter` | `false` | Use tree-sitter to resolve end lines when ctags does not provide them (supports Rust `.rs`, JavaScript `.js`, TypeScript `.ts`, Haskell `.hs`, OCaml `.ml`/`.mli`, PHP `.php`, and Kotlin `.kt`) |

The server resolves tag files relative to its **current working directory**:

- Default: `./tags`
- With `?context=<path>`: `./<path>/tags`

## API Reference

### Health Check

```
GET /healthz
```

Returns the server health status.

**Response**

`200 OK`

```json
{"status": "ok"}
```

---

### List All Tags

```
GET /tags
```

Returns all tags from the tags file.

**Query Parameters**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `context` | No | Subdirectory path to look up the tags file. If omitted, `./tags` is used; otherwise `./<context>/tags` is used. |

**Responses**

| Status | Description |
|--------|-------------|
| `200 OK` | JSON array of tag objects |
| `404 Not Found` | Tags file does not exist |
| `500 Internal Server Error` | Failed to load or parse the tags file |

**Example**

```
GET /tags
GET /tags?context=myproject
```

```json
[
  {
    "_type": "tag",
    "name": "MyFunc",
    "path": "main.go",
    "pattern": "^func MyFunc() {$",
    "language": "Go",
    "kind": "function",
    "line": 10
  }
]
```

---

### Look Up a Tag by Name

```
GET /tags/{name}
```

Returns all tags matching the given name.

**Path Parameters**

| Parameter | Description |
|-----------|-------------|
| `name` | Tag name to search for |

**Query Parameters**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `context` | No | Subdirectory path to look up the tags file. If omitted, `./tags` is used; otherwise `./<context>/tags` is used. |

**Responses**

| Status | Description |
|--------|-------------|
| `200 OK` | JSON array of matching tag objects |
| `404 Not Found` | Tag not found, or tags file does not exist |
| `500 Internal Server Error` | Failed to run `readtags` or parse the tags file |

**Example**

```
GET /tags/MyFunc
GET /tags/MyFunc?context=myproject
```

```json
[
  {
    "_type": "tag",
    "name": "MyFunc",
    "path": "main.go",
    "pattern": "^func MyFunc() {$",
    "language": "Go",
    "kind": "function",
    "line": 10
  }
]
```

---

### Get a Code Snippet by Tag Name

```
GET /snippets/{name}
```

Returns code snippets extracted from the source files for all tags matching the given name.

The end line is resolved in the following order of priority:

1. The `end` extension field recorded in the ctags tags file.
2. Tree-sitter analysis (when `--tree-sitter` is enabled and the file is a supported language: `.rs`, `.js`, `.ts`, `.hs`, `.ml`, `.mli`, `.php`, or `.kt`).
3. If neither source provides an end line, `end` is returned as `0` and the snippet contains only the single line at `start`.

**Path Parameters**

| Parameter | Description |
|-----------|-------------|
| `name` | Tag name to search for |

**Query Parameters**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `context` | No | Subdirectory path to look up the tags file. If omitted, `./tags` is used; otherwise `./<context>/tags` is used. |

**Responses**

| Status | Description |
|--------|-------------|
| `200 OK` | JSON array of snippet objects |
| `404 Not Found` | Tag not found, or tags file does not exist |
| `500 Internal Server Error` | Failed to look up the tag or read the source file |

**Example**

```
GET /snippets/MyFunc
GET /snippets/MyFunc?context=myproject
```

```json
[
  {
    "name": "MyFunc",
    "path": "main.go",
    "start": 10,
    "end": 15,
    "code": "func MyFunc() {\n\treturn 42\n}"
  }
]
```

---

### Get Line Range by Tag Name

```
GET /lines/{name}
```

Returns the start and end line numbers for all tags matching the given name, without the source code content.

The end line is resolved in the following order of priority:

1. The `end` extension field recorded in the ctags tags file.
2. Tree-sitter analysis (when `--tree-sitter` is enabled and the file is a supported language: `.rs`, `.js`, `.ts`, `.hs`, `.ml`, `.mli`, `.php`, or `.kt`).
3. If neither source provides an end line, `end` is returned as `0`, indicating that the end of the definition is unknown.

**Path Parameters**

| Parameter | Description |
|-----------|-------------|
| `name` | Tag name to search for |

**Query Parameters**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `context` | No | Subdirectory path to look up the tags file. If omitted, `./tags` is used; otherwise `./<context>/tags` is used. |

**Responses**

| Status | Description |
|--------|-------------|
| `200 OK` | JSON array of line range objects |
| `404 Not Found` | Tag not found, or tags file does not exist |
| `500 Internal Server Error` | Failed to look up the tag or read the source file |

**Example**

```
GET /lines/MyFunc
GET /lines/MyFunc?context=myproject
```

```json
[
  {
    "name": "MyFunc",
    "path": "main.go",
    "start": 10,
    "end": 15
  }
]
```

---

### Write to the Pipe Buffer

```
POST /pipe
```

Stores a JSON snippet object in the in-memory pipe buffer. By default the buffer is replaced; pass `?mode=append` to append instead.

**Query Parameters**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `mode` | No | If set to `append`, the body is appended to the existing buffer. Any other value (or omitted) replaces the buffer. |

**Request Body**

A JSON object with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Tag name (may be empty) |
| `path` | string | Source file path |
| `start` | number | Start line of the selection (1-based, inclusive) |
| `end` | number | End line of the selection (1-based, inclusive) |
| `code` | string | Selected source code |

**Responses**

| Status | Description |
|--------|-------------|
| `204 No Content` | Buffer updated successfully |
| `500 Internal Server Error` | Failed to read the request body |

**Example**

```
POST /pipe
Content-Type: application/json

{
  "name": "",
  "path": "main.go",
  "start": 10,
  "end": 15,
  "code": "func MyFunc() {\n\treturn 42\n}"
}
```

---

### Read from the Pipe Buffer

```
GET /pipe
```

Returns the current contents of the in-memory pipe buffer. Returns an empty body if the buffer has not been written to yet.

**Responses**

| Status | Description |
|--------|-------------|
| `200 OK` | Buffer contents (may be empty) |

**Example**

```
GET /pipe
```

```json
{"name":"","path":"main.go","start":10,"end":15,"code":"func MyFunc() {\n\treturn 42\n}"}
```

---

### Get Pipe Buffer Status

```
GET /pipe/status
```

Returns whether the in-memory pipe buffer is empty.

**Responses**

| Status | Description |
|--------|-------------|
| `200 OK` | JSON object with `empty` field |

**Example**

```json
{"empty": false}
```

| Field | Type | Description |
|-------|------|-------------|
| `empty` | boolean | `true` if the buffer is empty, `false` otherwise |

---

## Line Range Object

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Tag name |
| `path` | string | Source file path |
| `start` | number | Start line of the tag (1-based, inclusive) |
| `end` | number | End line of the tag (1-based, inclusive); `0` if the end could not be determined |

---

## Snippet Object

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Tag name |
| `path` | string | Source file path |
| `start` | number | Start line of the snippet (1-based, inclusive) |
| `end` | number | End line of the snippet (1-based, inclusive); `0` if the end could not be determined, in which case `code` contains only the single line at `start` |
| `code` | string | Extracted source code lines joined with newlines |

---

## Tag Object

Fields follow the [Universal Ctags JSON output format](https://docs.ctags.io/en/latest/man/ctags-client-tools.7.html). Optional fields are omitted when empty.

| Field | Type | Description |
|-------|------|-------------|
| `_type` | string | Always `"tag"` |
| `name` | string | Tag name |
| `path` | string | Source file path |
| `pattern` | string | Search pattern from the tags file (optional) |
| `language` | string | Programming language (optional) |
| `kind` | string | Tag kind, e.g. `function`, `type`, `method` (optional) |
| `line` | number | Line number in the source file (optional) |
| *(extension fields)* | string | Additional ctags extension fields (e.g. `typeref`, `scope`) are inlined at the top level |

## Tag File Resolution

The server resolves tags files relative to the working directory at startup:

| `context` query param | Tags file path |
|-----------------------|----------------|
| *(not set)* | `./tags` |
| `sub/project` | `./sub/project/tags` |

If `readtags` is available on `$PATH`, it is used for individual tag lookups (`GET /tags/{name}`, `GET /snippets/{name}`, and `GET /lines/{name}`); otherwise the tags file is parsed in-memory.
