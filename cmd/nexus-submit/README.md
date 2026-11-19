# nexus-submit

`nexus-submit` is a fast, lightweight CLI tool for one-shot AI task submission from a Markdown task specification file.

---

## Usage

```sh
nexus-submit /path/to/TASK-123.md
```

`nexus-submit` parses frontmatter attributes (`project`, `model`, `role`, `verify`), packages context files, and submits the task directly to the running `nexus-daemon`.
