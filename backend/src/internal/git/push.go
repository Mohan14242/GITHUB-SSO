package git

import (
	"fmt"
	"log"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// PushRepo initialises a local repo, commits all files, and pushes
// to the organisation remote under the specified branch
func PushRepo(token, repoName, localPath, branch string) error {
	log.Printf("[GIT][PUSH-REPO] starting repoName=%s localPath=%s branch=%s",
		repoName, localPath, branch)

	// ── Resolve org ───────────────────────────────────────────────
	org, err := getOrgName()
	if err != nil {
		log.Printf("[GIT][PUSH-REPO][ERROR] failed to get org name: %v", err)
		return err
	}
	log.Printf("[GIT][PUSH-REPO] org resolved org=%s", org)

	// ── 1. Git init ───────────────────────────────────────────────
	log.Printf("[GIT][PUSH-REPO] initialising local git repo path=%s", localPath)
	repo, err := git.PlainInit(localPath, false)
	if err != nil {
		log.Printf("[GIT][PUSH-REPO][ERROR] git init failed path=%s: %v", localPath, err)
		return fmt.Errorf("git init failed: %w", err)
	}
	log.Printf("[GIT][PUSH-REPO] ✅ git init complete path=%s", localPath)

	// ── 2. Worktree ───────────────────────────────────────────────
	log.Printf("[GIT][PUSH-REPO] getting worktree")
	worktree, err := repo.Worktree()
	if err != nil {
		log.Printf("[GIT][PUSH-REPO][ERROR] failed to get worktree: %v", err)
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// ── 3. Stage all files ────────────────────────────────────────
	log.Printf("[GIT][PUSH-REPO] staging all files path=%s", localPath)
	if _, err := worktree.Add("."); err != nil {
		log.Printf("[GIT][PUSH-REPO][ERROR] git add failed: %v", err)
		return fmt.Errorf("git add failed: %w", err)
	}
	log.Printf("[GIT][PUSH-REPO] ✅ files staged")

	// ── 4. Initial commit ─────────────────────────────────────────
	log.Printf("[GIT][PUSH-REPO] creating initial commit")
	commitHash, err := worktree.Commit("Initial commit from platform", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Platform Bot",
			Email: "platform@company.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Printf("[GIT][PUSH-REPO][ERROR] commit failed: %v", err)
		return fmt.Errorf("git commit failed: %w", err)
	}
	log.Printf("[GIT][PUSH-REPO] ✅ initial commit created hash=%s", commitHash.String())

	// ── 5. Create & checkout target branch ───────────────────────
	log.Printf("[GIT][PUSH-REPO] creating and checking out branch=%s", branch)
	refName := plumbing.NewBranchReferenceName(branch)
	if err := worktree.Checkout(&git.CheckoutOptions{
		Branch: refName,
		Create: true,
	}); err != nil {
		log.Printf("[GIT][PUSH-REPO][ERROR] checkout failed branch=%s: %v", branch, err)
		return fmt.Errorf("create/checkout branch %s failed: %w", branch, err)
	}
	log.Printf("[GIT][PUSH-REPO] ✅ checked out branch=%s", branch)

	// ── 6. Add remote pointing to org ─────────────────────────────
	remoteURL := fmt.Sprintf("https://github.com/%s/%s.git", org, repoName)
	log.Printf("[GIT][PUSH-REPO] adding remote origin url=%s", remoteURL)

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL},
	})
	if err != nil {
		log.Printf("[GIT][PUSH-REPO][ERROR] failed to add remote url=%s: %v", remoteURL, err)
		return fmt.Errorf("failed to add remote origin %s: %w", remoteURL, err)
	}
	log.Printf("[GIT][PUSH-REPO] ✅ remote added origin=%s", remoteURL)

	// ── 7. Push branch to org remote ─────────────────────────────
	refSpec := config.RefSpec(
		fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch),
	)
	log.Printf("[GIT][PUSH-REPO] pushing branch=%s refSpec=%s org=%s repo=%s",
		branch, refSpec, org, repoName)

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{refSpec},
		Auth: &http.BasicAuth{
			Username: "x-access-token",
			Password: token,
		},
		Progress: nil,
	})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			log.Printf("[GIT][PUSH-REPO] ⚠️ already up to date org=%s repo=%s branch=%s",
				org, repoName, branch)
			return nil
		}
		log.Printf("[GIT][PUSH-REPO][ERROR] push failed org=%s repo=%s branch=%s: %v",
			org, repoName, branch, err)
		return fmt.Errorf("git push failed org=%s repo=%s branch=%s: %w",
			org, repoName, branch, err)
	}

	log.Printf("[GIT][PUSH-REPO] ✅ push complete org=%s repo=%s branch=%s url=%s",
		org, repoName, branch, remoteURL)
	return nil
}
