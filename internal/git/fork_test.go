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
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestForkinator(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "forkinator-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	upstreamPath := filepath.Join(tempDir, "upstream")
	forkPath := filepath.Join(tempDir, "fork")

	// 1. Create upstream repo
	repo, err := git.PlainInit(upstreamPath, false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(upstreamPath, "hello.txt")
	err = os.WriteFile(filename, []byte("hello world"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Add("hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. Create fork
	err = CreateFork(upstreamPath, forkPath)
	if err != nil {
		t.Fatalf("CreateFork failed: %v", err)
	}

	// 3. Verify alternates file
	alternatesFile := filepath.Join(forkPath, "objects", "info", "alternates")
	content, err := os.ReadFile(alternatesFile)
	if err != nil {
		t.Fatalf("failed to read alternates file: %v", err)
	}

	absUpstream, _ := filepath.Abs(upstreamPath)
	expectedPath := filepath.Join(absUpstream, ".git", "objects")
	if string(content) != expectedPath+"\n" {
		t.Errorf("expected alternates content %q, got %q", expectedPath+"\n", string(content))
	}

	// 4. Verify fork has the same refs and HEAD
	forkRepo, err := git.PlainOpen(forkPath)
	if err != nil {
		t.Fatal(err)
	}

	head, err := forkRepo.Reference(plumbing.HEAD, false)
	if err != nil {
		t.Errorf("failed to get HEAD from fork: %v", err)
	}

	if head.Target().String() != "refs/heads/master" {
		t.Errorf("expected fork HEAD to be refs/heads/master, got %s", head.Target())
	}

	// 5. Verify objects are borrowed (not present in fork)
	upstreamHead, _ := repo.Head()
	commitHash := upstreamHead.Hash()

	objectFile := filepath.Join(forkPath, "objects", commitHash.String()[:2], commitHash.String()[2:])
	if _, err := os.Stat(objectFile); err == nil {
		t.Errorf("object %s should not exist in fork (it should be borrowed)", commitHash)
	}

	// 6. Detach fork
	err = DetachFork(forkPath)
	if err != nil {
		t.Fatalf("DetachFork failed: %v", err)
	}

	// 7. Verify alternates file is gone
	if _, err := os.Stat(alternatesFile); !os.IsNotExist(err) {
		t.Errorf("alternates file should have been removed")
	}

	// 8. Verify object now exists in fork (it will be in a packfile)
	_, err = forkRepo.Object(plumbing.CommitObject, commitHash)
	if err != nil {
		t.Errorf("object %s should now exist in fork after detach: %v", commitHash, err)
	}

	// 9. Verify fork still works after upstream deletion
	os.RemoveAll(upstreamPath)
	_, err = forkRepo.CommitObject(commitHash)
	if err != nil {
		t.Errorf("fork should still be able to access objects after upstream deletion: %v", err)
	}
}

func TestDetachWithLocalBranch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "forkinator-test-local-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	upstreamPath := filepath.Join(tempDir, "upstream")
	forkPath := filepath.Join(tempDir, "fork")

	// 1. Create upstream repo
	repo, err := git.PlainInit(upstreamPath, false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(upstreamPath, "hello.txt")
	err = os.WriteFile(filename, []byte("hello world"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Add("hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. Create fork
	err = CreateFork(upstreamPath, forkPath)
	if err != nil {
		t.Fatalf("CreateFork failed: %v", err)
	}

	// 3. Create a local branch in fork that has a new commit
	clonePath := filepath.Join(tempDir, "clone")
	cmd := exec.Command("git", "clone", forkPath, clonePath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone fork: %v", err)
	}

	cmd = exec.Command("git", "-C", clonePath, "checkout", "-b", "local-branch")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to checkout local-branch: %v", err)
	}

	err = os.WriteFile(filepath.Join(clonePath, "local.txt"), []byte("local change"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("git", "-C", clonePath, "add", "local.txt")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add local.txt: %v", err)
	}

	cmd = exec.Command("git", "-C", clonePath, "-c", "user.name=Test", "-c", "user.email=t@test.com", "commit", "-m", "Local commit")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit local: %v", err)
	}

	cmd = exec.Command("git", "-C", clonePath, "push", "origin", "local-branch")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to push local-branch: %v", err)
	}

	// Now verify the commit exists in the fork (via alternates)
	forkRepo, err := git.PlainOpen(forkPath)
	if err != nil {
		t.Fatal(err)
	}

	localRef, err := forkRepo.Reference(plumbing.ReferenceName("refs/heads/local-branch"), true)
	if err != nil {
		t.Fatalf("fork doesn't have local-branch: %v", err)
	}
	localHash := localRef.Hash()

	// 4. Detach fork
	err = DetachFork(forkPath)
	if err != nil {
		t.Fatalf("DetachFork failed: %v", err)
	}

	// 5. Verify the fork is detached, and the local-branch ref and its commit still exist and work
	// Even after deleting upstream!
	os.RemoveAll(upstreamPath)

	_, err = forkRepo.CommitObject(localHash)
	if err != nil {
		t.Errorf("fork lost local-branch commit after detach & upstream delete: %v", err)
	}
}

func TestDetachForkWithLocalCommitOnMaster(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "forkinator-test-master-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	upstreamPath := filepath.Join(tempDir, "upstream")
	forkPath := filepath.Join(tempDir, "fork")

	// 1. Create upstream repo
	repo, err := git.PlainInit(upstreamPath, false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(upstreamPath, "hello.txt")
	err = os.WriteFile(filename, []byte("hello world"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Add("hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. Create fork
	err = CreateFork(upstreamPath, forkPath)
	if err != nil {
		t.Fatalf("CreateFork failed: %v", err)
	}

	// 3. Create a local commit on master in fork
	clonePath := filepath.Join(tempDir, "clone")
	cmd := exec.Command("git", "clone", forkPath, clonePath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone fork: %v", err)
	}

	err = os.WriteFile(filepath.Join(clonePath, "local.txt"), []byte("local change"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("git", "-C", clonePath, "add", "local.txt")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add local.txt: %v", err)
	}

	cmd = exec.Command("git", "-C", clonePath, "-c", "user.name=Test", "-c", "user.email=t@test.com", "commit", "-m", "Local commit on master")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit local: %v", err)
	}

	cmd = exec.Command("git", "-C", clonePath, "push", "origin", "master")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to push master: %v", err)
	}

	// Now verify the commit exists in the fork
	forkRepo, err := git.PlainOpen(forkPath)
	if err != nil {
		t.Fatal(err)
	}

	masterRef, err := forkRepo.Reference(plumbing.ReferenceName("refs/heads/master"), true)
	if err != nil {
		t.Fatalf("fork doesn't have master: %v", err)
	}
	localHash := masterRef.Hash()

	// 4. Detach fork
	err = DetachFork(forkPath)
	if err != nil {
		t.Fatalf("DetachFork failed: %v", err)
	}

	// 5. Verify the fork is detached, and the master ref and its commit still exist and work
	// Even after deleting upstream!
	os.RemoveAll(upstreamPath)

	masterRefPost, err := forkRepo.Reference(plumbing.ReferenceName("refs/heads/master"), true)
	if err != nil {
		t.Fatalf("fork lost master reference post-detach: %v", err)
	}

	if masterRefPost.Hash() != localHash {
		t.Errorf("expected master ref hash to be %s, but got %s (overwritten by upstream!)", localHash, masterRefPost.Hash())
	}

	_, err = forkRepo.CommitObject(localHash)
	if err != nil {
		t.Errorf("fork lost local master commit after detach & upstream delete: %v", err)
	}
}

func TestSyncFork(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "forkinator-test-sync-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	upstreamPath := filepath.Join(tempDir, "upstream")
	forkPath := filepath.Join(tempDir, "fork")

	// 1. Create upstream repo
	repo, err := git.PlainInit(upstreamPath, false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	filename := filepath.Join(upstreamPath, "hello.txt")
	err = os.WriteFile(filename, []byte("hello world"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Add("hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. Create fork
	err = CreateFork(upstreamPath, forkPath)
	if err != nil {
		t.Fatalf("CreateFork failed: %v", err)
	}

	// 3. Add a new commit to upstream master branch
	err = os.WriteFile(filepath.Join(upstreamPath, "hello.txt"), []byte("hello world updated"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Add("hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	newCommit, err := w.Commit("Second commit on upstream", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 4. Sync the fork
	err = SyncFork(forkPath)
	if err != nil {
		t.Fatalf("SyncFork failed: %v", err)
	}

	// 5. Verify the fork master branch has been fast-forwarded to newCommit
	forkRepo, err := git.PlainOpen(forkPath)
	if err != nil {
		t.Fatal(err)
	}

	masterRef, err := forkRepo.Reference(plumbing.ReferenceName("refs/heads/master"), true)
	if err != nil {
		t.Fatalf("failed to get master ref post-sync: %v", err)
	}

	if masterRef.Hash() != newCommit {
		t.Errorf("expected master ref to be fast-forwarded to %s, but got %s", newCommit, masterRef.Hash())
	}
}
