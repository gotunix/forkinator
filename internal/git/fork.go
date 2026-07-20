// SPDX-License-Identifier: GPL-3.0-or-later
// SPDX-FileCopyrightText: 2026 The Forkinator authors
// =============================================================================================== //
//    /$$$$$$$$                  /$$       /$$                       /$$                           //
//   | $$_____/                 | $$      |__/                      | $$                           //
//   | $$     /$$$$$$   /$$$$$$ | $$   /$$ /$$ /$$$$$$$   /$$$$$$  /$$$$$$    /$$$$$$   /$$$$$$    //
//   | $$$$$ /$$__  $$ /$$__  $$| $$  /$$/| $$| $$__  $$ |____  $$|_  $$_/   /$$__  $$ /$$__  $$   //
//   | $$__/| $$  \ $$| $$  \__/| $$$$$$/ | $$| $$  \ $$  /$$$$$$$  | $$    | $$  \ $$| $$  \__/   //
//   | $$   | $$  | $$| $$      | $$_  $$ | $$| $$  | $$ /$$__  $$  | $$ /$$| $$  | $$| $$         //
//   | $$   |  $$$$$$/| $$      | $$ \  $$| $$| $$  | $$|  $$$$$$$  |  $$$$/|  $$$$$$/| $$         //
//   |__/    \______/ |__/      |__/  \__/|__/|__/  |__/ \_______/   \___/   \______/ |__/         //
//                                                                                                 //
// =============================================================================================== //
// This program is free software: you can redistribute it and/or modify                            //
// it under the terms of the GNU Affero General Public License as                                  //
// published by the Free Software Foundation, either version 3 of the                              //
// License, or (at your option) any later version.                                                 //
//                                                                                                 //
// This program is distributed in the hope that it will be useful,                                 //
// but WITHOUT ANY WARRANTY; without even the implied warranty of                                  //
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                   //
// GNU Affero General Public License for more details.                                             //
//                                                                                                 //
// You should have received a copy of the GNU Affero General Public License                        //
// along with this program.  If not, see <https://www.gnu.org/licenses/>.                          //
// =============================================================================================== //

package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// CreateFork creates a new bare repository at forkPath that uses upstreamPath as an alternate object store.
func CreateFork(upstreamPath, forkPath string) error {
	// 1. Validate upstream exists and is a git repo
	upstreamPath, err := filepath.Abs(upstreamPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for upstream: %w", err)
	}

	if _, err := os.Stat(filepath.Join(upstreamPath, "objects")); os.IsNotExist(err) {
		// Check if it's a non-bare repo (objects are in .git/objects)
		if _, err := os.Stat(filepath.Join(upstreamPath, ".git", "objects")); err == nil {
			upstreamPath = filepath.Join(upstreamPath, ".git")
		} else {
			return fmt.Errorf("upstream path %s does not appear to be a git repository (missing objects directory)", upstreamPath)
		}
	}

	// 2. Initialize bare repo at forkPath
	if _, err = git.PlainInit(forkPath, true); err != nil {
		return fmt.Errorf("failed to initialize bare repository at %s: %w", forkPath, err)
	}

	// 3. Configure alternates
	// The alternates file should contain the absolute path to the upstream's objects directory.
	upstreamObjectsPath := filepath.Join(upstreamPath, "objects")
	alternatesDir := filepath.Join(forkPath, "objects", "info")
	if err := os.MkdirAll(alternatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create objects/info directory: %w", err)
	}

	alternatesFile := filepath.Join(alternatesDir, "alternates")
	if err := os.WriteFile(alternatesFile, []byte(upstreamObjectsPath+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write alternates file: %w", err)
	}

	// Re-open the repository so it picks up the alternates
	repo, err := git.PlainOpen(forkPath)
	if err != nil {
		return fmt.Errorf("failed to re-open repository: %w", err)
	}

	// 4. Fetch refs from upstream
	// We add the upstream as a remote and fetch from it.
	remote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "upstream",
		URLs: []string{upstreamPath},
	})
	if err != nil {
		return fmt.Errorf("failed to create upstream remote: %w", err)
	}

	err = remote.Fetch(&git.FetchOptions{
		RemoteName: "upstream",
		RefSpecs:   []config.RefSpec{"+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"},
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch refs from upstream: %w", err)
	}

	// 5. Set HEAD to match upstream's HEAD
	upstreamRepo, err := git.PlainOpen(upstreamPath)
	if err != nil {
		return fmt.Errorf("failed to open upstream repo to check HEAD: %w", err)
	}
	// Get the symbolic HEAD of the upstream
	upstreamHead, err := upstreamRepo.Reference(plumbing.HEAD, false)
	if err != nil {
		return fmt.Errorf("failed to get upstream HEAD: %w", err)
	}

	if upstreamHead.Type() == plumbing.SymbolicReference {
		// Set fork's HEAD to the same symbolic target
		head := plumbing.NewSymbolicReference(plumbing.HEAD, upstreamHead.Target())
		if err := repo.Storer.SetReference(head); err != nil {
			return fmt.Errorf("failed to set fork HEAD (symbolic): %w", err)
		}
	} else {
		// Set fork's HEAD to the same hash (detached)
		head := plumbing.NewHashReference(plumbing.HEAD, upstreamHead.Hash())
		if err := repo.Storer.SetReference(head); err != nil {
			return fmt.Errorf("failed to set fork HEAD (hash): %w", err)
		}
	}

	return nil
}

// DetachFork makes a fork independent by repacking all borrowed objects locally and removing the alternates file.
func DetachFork(forkPath string) error {
	repo, err := git.PlainOpen(forkPath)
	if err != nil {
		return fmt.Errorf("failed to open fork: %w", err)
	}

	// 1. Check if alternates exists
	alternatesFile := filepath.Join(forkPath, "objects", "info", "alternates")
	if _, err := os.Stat(alternatesFile); os.IsNotExist(err) {
		return fmt.Errorf("repository at %s is not a fork (no alternates file found)", forkPath)
	}

	// Read the upstream path from the remote config
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repo config: %w", err)
	}

	upstreamRemote, ok := cfg.Remotes["upstream"]
	if !ok || len(upstreamRemote.URLs) == 0 {
		return fmt.Errorf("could not find 'upstream' remote to fetch objects from")
	}
	upstreamURL := upstreamRemote.URLs[0]

	// Read alternates content so we can backup/restore it on failure
	alternatesContent, err := os.ReadFile(alternatesFile)
	if err != nil {
		return fmt.Errorf("failed to read alternates file: %w", err)
	}

	// Delete/rename the alternates file. We will restore it on failure.
	if err := os.Remove(alternatesFile); err != nil {
		return fmt.Errorf("failed to remove alternates file: %w", err)
	}

	restoreAlternates := func() {
		_ = os.WriteFile(alternatesFile, alternatesContent, 0644)
	}

	// Create temporary remote for detaching
	remote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "detach-temp",
		URLs: []string{upstreamURL},
	})
	if err != nil {
		restoreAlternates()
		return fmt.Errorf("failed to create temp remote for detaching: %w", err)
	}

	// Fetch all objects from upstream mapping them to a temporary namespace (refs/detach-temp/*)
	// to avoid overwriting the fork's own branch heads/tags (especially master/main).
	err = remote.Fetch(&git.FetchOptions{
		RemoteName: "detach-temp",
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/detach-temp/heads/*",
			"+refs/tags/*:refs/detach-temp/tags/*",
		},
		Tags: git.NoTags, // Do not fetch tags automatically to standard refs/tags/*
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		_ = repo.DeleteRemote("detach-temp")
		restoreAlternates()
		return fmt.Errorf("failed to fetch objects during detach: %w", err)
	}

	// Clean up the temp remote
	_ = repo.DeleteRemote("detach-temp")

	// Clean up all the temporary refs we created
	refs, err := repo.References()
	if err == nil {
		_ = refs.ForEach(func(ref *plumbing.Reference) error {
			if strings.HasPrefix(ref.Name().String(), "refs/detach-temp/") {
				_ = repo.Storer.RemoveReference(ref.Name())
			}
			return nil
		})
	}

	return nil
}

// SyncFork fetches updates from the upstream remote and updates local references.
func SyncFork(forkPath string) error {
	repo, err := git.PlainOpen(forkPath)
	if err != nil {
		return fmt.Errorf("failed to open fork: %w", err)
	}

	// Retrieve the upstream remote
	remote, err := repo.Remote("upstream")
	if err != nil {
		return fmt.Errorf("could not find 'upstream' remote in fork config: %w", err)
	}

	// Fetch references from upstream to refs/remotes/upstream/*
	fetchOpts := &git.FetchOptions{
		RemoteName: "upstream",
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/remotes/upstream/*",
			"+refs/tags/*:refs/tags/*",
		},
		Tags: git.NoTags,
	}

	err = remote.Fetch(fetchOpts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch from upstream: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("failed to get references: %w", err)
	}

	const remotePrefix = "refs/remotes/upstream/"
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refName := ref.Name().String()
		if !strings.HasPrefix(refName, remotePrefix) {
			return nil
		}

		branchName := strings.TrimPrefix(refName, remotePrefix)
		localRefName := plumbing.ReferenceName("refs/heads/" + branchName)

		localRef, err := repo.Reference(localRefName, true)
		if err == plumbing.ErrReferenceNotFound {
			// Local branch doesn't exist yet, create it pointing to the remote commit
			newRef := plumbing.NewHashReference(localRefName, ref.Hash())
			if err := repo.Storer.SetReference(newRef); err != nil {
				fmt.Printf("• Warning: failed to create branch %s: %v\n", branchName, err)
			} else {
				fmt.Printf("• Created new branch: %s\n", branchName)
			}
			return nil
		} else if err != nil {
			return err
		}

		// Local branch exists. Check if it's a fast-forward.
		if localRef.Hash() == ref.Hash() {
			// Already up to date
			return nil
		}

		// Check if local branch can be fast-forwarded to remote
		isAncestor, err := isCommitAncestor(repo, localRef.Hash(), ref.Hash())
		if err != nil {
			fmt.Printf("• Warning: could not determine relationship for %s: %v\n", branchName, err)
			return nil
		}

		if isAncestor {
			// Fast-forward local branch to remote branch hash
			newRef := plumbing.NewHashReference(localRefName, ref.Hash())
			if err := repo.Storer.SetReference(newRef); err != nil {
				fmt.Printf("• Warning: failed to update branch %s: %v\n", branchName, err)
			} else {
				fmt.Printf("• Fast-forwarded branch: %s\n", branchName)
			}
		} else {
			// Check if local is ahead
			isLocalAncestor, err := isCommitAncestor(repo, ref.Hash(), localRef.Hash())
			if err == nil && isLocalAncestor {
				fmt.Printf("• Branch %s is ahead of upstream (local commits present)\n", branchName)
			} else {
				fmt.Printf("• Warning: branch %s has diverged from upstream, skipping automatic update\n", branchName)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to process branch updates: %w", err)
	}

	return nil
}

// isCommitAncestor returns true if 'ancestor' is an ancestor of 'commit'.
func isCommitAncestor(repo *git.Repository, ancestor, commit plumbing.Hash) (bool, error) {
	if ancestor == commit {
		return true, nil
	}

	cObj, err := repo.CommitObject(commit)
	if err != nil {
		return false, err
	}

	aObj, err := repo.CommitObject(ancestor)
	if err != nil {
		return false, err
	}

	return aObj.IsAncestor(cObj)
}
