package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/pier-oliviert/sequencer/api/v1alpha1/builds/config"
	"github.com/pier-oliviert/sequencer/internal/builder/secrets"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	cryptoSSH "golang.org/x/crypto/ssh"
)

type Repository struct {
	path  string
	ref   string
	url   string
	depth int
	auth  transport.AuthMethod

	*git.Repository
}

func Git(ctx context.Context, opts ...RepositoryOption) (*Repository, error) {
	var err error

	logger := log.FromContext(ctx)

	repo := &Repository{}

	for _, opt := range opts {
		opt(repo)
	}

	logger.Info("Cloning Git Repository", "URL", repo.url, "Ref", repo.ref)

	if err := os.MkdirAll(repo.path, os.ModePerm); err != nil {
		return nil, err
	}

	cloneOpts := &git.CloneOptions{
		Depth:      repo.depth,
		URL:        repo.url,
		NoCheckout: true,
	}

	if repo.auth != nil {
		logger.Info("Cloning repo using credential", "URL", repo.url)
		cloneOpts.Auth = repo.auth
	}

	repo.Repository, err = git.PlainClone(repo.path, false, cloneOpts)
	if err != nil {
		return nil, err
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	hash, err := repo.Repository.ResolveRevision(plumbing.Revision(repo.ref))
	if err != nil {
		return nil, err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	})

	if err != nil {
		return nil, fmt.Errorf("E#1015: Git error during checkout: %w", err)
	}

	return repo, nil
}

// Path returns the temporary path where
// the repository was cloned to.
func (r *Repository) Path() string {
	return r.path
}

// Ref returns the git reference(https://git-scm.com/book/en/v2/Git-Internals-Git-References)
// that was used to clone this repository. It is unlikely that a valid cloned
// repo returns an error here as it's asking for the Head, which will point at the reference
// used, but if it does happen, it means the repository is in an unknown state and can't be used
// to generate an image
func (r *Repository) Ref() (string, error) {
	ref, err := r.Head()
	if err != nil {
		return "", err
	}

	return ref.Hash().String(), nil
}

type RepositoryOption func(*Repository) error

type RepositoryOpts struct {
	Path string
	Host string
	Ref  string

	Credential *secrets.Credentials
}

func WithPath(path string) RepositoryOption {
	return func(r *Repository) error {
		r.path = path
		return nil
	}
}

func WithAuth(path string, credentials *config.Credentials) RepositoryOption {
	return func(r *Repository) error {
		if credentials.AuthScheme != config.SingleToken {
			return fmt.Errorf("E#1016: Only auth scheme supported for Git is SingleToken (privateKey): %s", credentials.AuthScheme)
		}

		file := filepath.Join(path, *credentials.Name, "privateKey")

		auth, err := ssh.NewPublicKeysFromFile("git", file, "")
		if err != nil {
			return err
		}
		auth.HostKeyCallback = cryptoSSH.InsecureIgnoreHostKey()
		r.auth = auth
		return nil
	}
}

func WithURL(url string) RepositoryOption {
	return func(r *Repository) error {
		r.url = url
		return nil
	}
}

func WithRef(ref string) RepositoryOption {
	return func(r *Repository) error {
		r.ref = ref
		return nil
	}
}

func WithDepth(depth *int) RepositoryOption {
	return func(r *Repository) error {
		if depth == nil {
			r.depth = 1
		} else {
			r.depth = *depth
		}
		return nil
	}
}
