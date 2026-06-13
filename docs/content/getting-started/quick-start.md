---
title: "Quick start"
description: "Run your first cf command."
weight: 30
---

Once `cf` is on your `PATH`:

```bash
cf --help       # see the command tree
cf version      # build info
```

This is a fresh scaffold, so the command tree is just `version` for now. Add
your first real command in `cli/`, build on the `codeforces` library package,
and document it here.

A good first command usually fetches one thing and prints it as JSON, so the
output pipes straight into `jq` and the rest of your tools.
