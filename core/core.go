package core

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/url"
	"os"
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/store"
	"github.com/geekgonecrazy/rfd-tool/store/boltstore"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var _dataStore store.Store
var _githubOAuth *oauth2.Config
var _oidcOAuth *oauth2.Config
var _oidcVerifier *oidc.IDTokenVerifier

var _jwtPrivateKey *rsa.PrivateKey
var _jwtPublicKey *rsa.PublicKey

var _validId *regexp.Regexp

var _gitPublicKeys *ssh.PublicKeys
var _gitCloneOptions *git.CloneOptions

func Setup() error {
	_validId, _ = regexp.Compile(`^\d{1,4}$`)

	// Initialize datastore
	store, err := boltstore.New()
	if err != nil {
		return err
	}

	_dataStore = store

	if err := _dataStore.EnsureUpdateLatestRFDID(); err != nil {
		return err
	}

	_githubOAuth = &oauth2.Config{
		ClientID:     config.Config.Github.ClientID,
		ClientSecret: config.Config.Github.ClientSecret,
		Scopes:       []string{"read:user"},
		RedirectURL:  fmt.Sprintf("%s/github/callback", config.Config.Site.URL),
		Endpoint:     github.Endpoint,
	}

	_oidcOAuth = &oauth2.Config{
		ClientID:     config.Config.OIDC.ClientID,
		ClientSecret: config.Config.OIDC.ClientSecret,
		Scopes:       []string{"openid", "profile", "email"},
		RedirectURL:  fmt.Sprintf("%s/oidc/callback", config.Config.Site.URL),
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.Config.OIDC.AuthURL,
			TokenURL: config.Config.OIDC.TokenURL,
		},
	}

	provider, err := oidc.NewProvider(context.TODO(), config.Config.OIDC.IssuerURL)
	if err != nil {
		return err
	}

	_oidcVerifier = provider.Verifier(&oidc.Config{ClientID: config.Config.OIDC.ClientID})

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(config.Config.JWT.PrivateKey))
	if err != nil {
		return err
	}

	_jwtPrivateKey = privateKey

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(config.Config.JWT.PublicKey))
	if err != nil {
		return err
	}

	_jwtPublicKey = publicKey

	publicKeys, err := ssh.NewPublicKeys(config.Config.Repo.Username, []byte(config.Config.Repo.PrivateDeployKey), "")
	if err != nil {
		return err
	}

	// Setup the git stuff
	_gitPublicKeys = publicKeys

	u, err := url.Parse(config.Config.Repo.URL)
	if err != nil {
		return err
	}

	sshGithubURL := fmt.Sprintf("%s:%s.git", u.Host, u.Path)

	_gitCloneOptions = &git.CloneOptions{
		URL:           sshGithubURL,
		Progress:      os.Stdout,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", config.Config.Repo.MainBranch)),
		Auth:          publicKeys,
	}

	return nil
}
