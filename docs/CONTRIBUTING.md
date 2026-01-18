# Contributing to cnquery Documentation

This folder contains the user-facing documentation for cnquery. The content is synced to the [Mondoo docs site](https://mondoo.com/docs/cnquery/).

## How the sync works

The cnquery docs live in this open source repo but are published on mondoo.com/docs. A GitHub Action in the docs repo syncs the content:

1. **Source**: This folder (`cnquery/docs/`)
2. **Destination**: `docs/content/cnquery/` in the private docs repo
3. **Trigger**: Daily at 6 AM UTC or manual dispatch

### What gets synced

- All `.mdx` files (documentation content)
- CLI reference files in `cli/` folder (auto-generated)

### What stays in the docs repo only

- `meta.json` files (fumadocs navigation configuration)
- Images in `public/img/cnquery/`

The sync workflow backs up and restores `meta.json` files to preserve navigation structure.

## Writing documentation

### File format

- Use `.mdx` extension for all documentation files
- CLI reference files in `cli/` use `.md` (auto-generated)

### Frontmatter

Each file needs frontmatter with at least:

```yaml
---
title: Page Title
sidebar_label: Short Label
displayed_sidebar: cnquery
description: Brief description for SEO
image: /img/cnquery/mondoo-feature.jpg
---
```

### Images

Images are stored in the docs repo at `public/img/cnquery/`. Reference them with absolute paths:

```markdown
![Alt text](/img/cnquery/folder/image.png)
```

Available image folders:
- `/img/cnquery/` - Featured images, banners
- `/img/cnquery/github/` - GitHub app setup screenshots
- `/img/cnquery/gw/` - Google Workspace screenshots
- `/img/cnquery/m365/` - Microsoft 365 screenshots

### Links

Use absolute paths for links:
```markdown
[MQL Reference](/mql/resources/)
[Other cnquery page](/cnquery/cnquery-about/)
```

## Adding new images

If you need new images:

1. Add the image to the docs repo at `public/img/cnquery/`
2. Reference it in your markdown with the absolute path
3. The image will be available after the docs repo change is merged

## Folder structure

```
docs/
├── cli/                 # Auto-generated CLI reference (do not edit)
├── cloud/               # Cloud provider docs (AWS, Azure, GCP, etc.)
│   ├── aws/
│   ├── k8s/
│   └── ...
├── os/                  # Operating system docs
├── saas/                # SaaS integration docs (GitHub, M365, etc.)
├── index.mdx            # cnquery docs home page
├── cnquery-about.mdx    # What is cnquery
├── providers.mdx        # Provider installation
└── CONTRIBUTING.md      # This file
```
