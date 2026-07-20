# ForkTool

`forktool` is a Go-based utility for creating efficient Git repository forks using the **alternates** mechanism. It allows you to create multiple forks of a single repository on the same filesystem while sharing the same underlying object database, significantly reducing disk space and speeding up fork creation.

This is similar to how large Git hosting platforms like GitHub and git.kernel.org manage forks and object deduplication.

## How it Works

Instead of performing a full `git clone --bare`, which copies every object (commit, tree, blob) from the upstream repository, `forktool` performs the following steps:

1.  **Initializes a Bare Repository:** Creates a new empty Git repository at the target path.
2.  **Configures Alternates:** Creates an `objects/info/alternates` file in the fork that points to the `objects` directory of the upstream repository.
3.  **Syncs Refs:** Fetches the branch and tag references from the upstream so the fork is immediately populated with the same history.
4.  **Sets HEAD:** Synchronizes the symbolic `HEAD` (default branch) to match the upstream repository.

Because of the `alternates` file, Git knows to look in the upstream's object store if it can't find an object locally. This means the fork only stores *new* objects created within the fork itself.

## Installation

Ensure you have [Go](https://golang.org/doc/install) installed, then clone this repository and build:

```bash
go build -o forktool main.go
```

## Usage

To create a new fork:

```bash
./forktool fork <upstream_path> <fork_path>
```

Example:

```bash
./forktool fork /srv/git/linux.git /srv/git/user-forks/linux.git
```

### Detaching a Fork

If you need to make a fork independent (so you can safely delete the upstream repository), use the `detach` command:

```bash
./forktool detach <fork_path>
```

This will copy all borrowed objects from the upstream into the fork's local object database and remove the `alternates` file.

## Caveats

*   **Same Filesystem:** The upstream and the fork must reside on the same filesystem (or at least be accessible via absolute paths on the same machine).
*   **Upstream Dependency:** If the upstream repository is deleted or its object database is corrupted, all forks depending on it via alternates will also break.
*   **Maintenance:** Standard Git maintenance tasks (like `git gc`) on the upstream can affect the forks. It is generally recommended to keep the upstream "pristine" and perform writes only to the forks.

## Implementation Details

The utility is built using the [go-git](https://github.com/go-git/go-git) library, providing a pure-Go implementation that does not depend on the system's `git` binary.
