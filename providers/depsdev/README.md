# deps.dev Provider

The deps.dev provider queries the [deps.dev API](https://deps.dev/) for dependency information about Go modules. It analyzes `go.mod` files and retrieves version history, project metadata, OpenSSF Scorecards, and GitHub repository status for each direct dependency.

## Prerequisites

- A `go.mod` file to analyze
- Optional: `GITHUB_TOKEN` environment variable for the `archived` field (avoids GitHub API rate limiting at 60 requests/hour unauthenticated)

```shell
export GITHUB_TOKEN="$(gh auth token)"
```

## Usage

Point the provider at a Go module directory or `go.mod` file:

```shell
mql shell depsdev ./path/to/project
```

Or query individual packages without a `go.mod`:

```shell
mql shell depsdev
```

## Examples

**List all direct dependencies with their versions**

```shell
mql run depsdev . -c 'depsdev.packages { name currentVersion latestVersion }'
```

**Find dependencies not updated in over a year**

```shell
mql run depsdev . -c 'depsdev.packages.where(latestPublished < time.now - 365 * time.day) { name latestVersion latestPublished }'
```

**Find archived dependencies**

```shell
mql run depsdev . -c 'depsdev.packages.where(project.archived == true) { name project { id } }'
```

**Combine filters: find stale and archived dependencies**

Chain `where` clauses to first filter by age (fewer deps.dev API calls), then check archived status (fewer GitHub API calls):

```shell
mql run depsdev . -c 'depsdev.packages.where(latestPublished < time.now - 365 * time.day).where(project.archived == true) { name latestVersion latestPublished project { id } }'
```

**Check OpenSSF Scorecard for a specific package**

```shell
mql run depsdev -c 'depsdev.package("github.com/rs/zerolog").project.scorecard { overallScore date checks { name score reason } }'
```

**View project metadata (stars, forks, license)**

```shell
mql run depsdev -c 'depsdev.package("github.com/rs/zerolog").project { id starsCount forksCount license description }'
```

## Future Language Support

The provider currently supports Go modules. The [deps.dev API](https://docs.deps.dev/api/v3/) also supports these package ecosystems, which could be added in the future:

- **npm** (Node.js)
- **PyPI** (Python)
- **Maven** (Java)
- **Cargo** (Rust)
- **NuGet** (.NET)
- **RubyGems** (Ruby)

To add support for another ecosystem, the provider would need to parse the corresponding manifest file (e.g., `package.json`, `requirements.txt`, `Cargo.toml`) and use the appropriate system identifier when calling the deps.dev API.
