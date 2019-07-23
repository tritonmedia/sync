package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Config is a configuration object for sync
type Config struct {
	// Version of this config
	Version string `yaml:"version"`

	// S3 is a S3 object
	S3 struct {
		// S3 compatible storage endpoint
		Endpoint string `yaml:"endpoint"`

		// AccessKey is the S3 access key
		AccessKey string `yaml:"accessKey"`

		// SecretAccessKey is the S3 secret access key
		SecretAccessKey string `yaml:"secretAccessKey"`

		// Bucket is the S3 bucket
		Bucket string `yaml:"bucket"`
	} `yaml:"s3"`

	// SaveDir is an absolute path to sync media into. If empty assumes workdir
	SaveDir string `yaml:"saveDir"`
}

// Load loads a config from `config.yaml`
func Load() (*Config, error) {
	d, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("failed to read working directory: %v", err)
		return nil, err
	}

	b, err := ioutil.ReadFile(filepath.Join(d, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read conf: %v", err)
	}

	var conf *Config
	err = yaml.Unmarshal(b, &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal conf: %v", err)
	}

	return conf, nil
}
