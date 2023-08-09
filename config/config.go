package config

import (
	"errors"
	"io/ioutil"
	"log"
	"strings"

	"github.com/gin-gonic/gin"

	yaml "gopkg.in/yaml.v2"
)

// Config contains a reference to the configuration
var Config *config

type config struct {
	Site              siteConfig   `yaml:"site" json:"site"`
	DataPath          string       `yaml:"dataPath" json:"dataPath"`
	APISecret         string       `yaml:"apiSecret" json:"apiSecret"`
	OIDC              oidcConfig   `yaml:"oidc" json:"oidc"`
	Github            githubConfig `yaml:"github" json:"github"`
	JWT               jwtConfig    `yaml:"jwt" json:"jwt"`
	RocketChatWebhook string       `yaml:"rocketchatWebhook" json:"rocketchatWebhook"`
}

type siteConfig struct {
	Name    string `yaml:"name" json:"name"`
	URL     string `yaml:"url" json:"url"`
	LogoSVG string `yaml:"logo.svg" json:"logo.svg"`
}

type jwtConfig struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`
	PublicKey  string `yaml:"publicKey" json:"publicKey"`
}

type oidcConfig struct {
	ClientID     string `yaml:"clientId" json:"clientId"`
	ClientSecret string `yaml:"clientSecret" json:"clientSecret"`
	AuthURL      string `yaml:"authUrl" json:"authUrl"`
	TokenURL     string `yaml:"tokenUrl" json:"tokenUrl"`
	IssuerURL    string `yaml:"issuerUrl" json:"issuerUrl"`
}

type githubConfig struct {
	Repo                string `yaml:"repo" json:"repo"`
	Folder              string `yaml:"folder" json:"folder"`
	MainBranch          string `yaml:"mainBranch" json:"mainBranch"`
	PersonalAccessToken string `yaml:"personalAccessToken"`
	ClientID            string `yaml:"clientId" json:"clientId"`
	ClientSecret        string `yaml:"clientSecret" json:"clientSecret"`
}

func (c *config) Load(filePath string) error {
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		return err
	}

	if err = yaml.Unmarshal(yamlFile, c); err != nil {
		log.Fatalf("Unmarshal: %v", err)
		return err
	}

	return nil
}

func (c *config) VerifySettings() error {
	if c.DataPath == "" {
		return errors.New("invalid dataPath, can not be empty")
	}

	if !strings.HasSuffix(c.DataPath, "/") {
		return errors.New("dataPath must end with '/'")
	}

	return nil
}

func (c *config) HttpHandler(gc *gin.Context) {
	gc.JSON(200, Config)
}

// Load tries to load the configuration file
func Load(filePath string) error {
	Config = new(config)

	if err := Config.Load(filePath); err != nil {
		return err
	}

	return Config.VerifySettings()
}
