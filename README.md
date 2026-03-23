# codesnippetd

A REST API server that exposes [Universal Ctags](https://ctags.io/) tag databases over HTTP.

## Usage

```
codesnippetd [-addr <listen-address>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | TCP address to listen on |

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
The snippet spans from the tag's start line to the end line recorded in the ctags `end` extension field.
If the `end` field is absent, the snippet extends to the end of the file.

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
The end line is taken from the ctags `end` extension field; if absent, it defaults to the total line count of the file.

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

## Line Range Object

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Tag name |
| `path` | string | Source file path |
| `start` | number | Start line of the tag (1-based, inclusive) |
| `end` | number | End line of the tag (1-based, inclusive) |

---

## Snippet Object

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Tag name |
| `path` | string | Source file path |
| `start` | number | Start line of the snippet (1-based, inclusive) |
| `end` | number | End line of the snippet (1-based, inclusive) |
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
