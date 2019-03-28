package config

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

// Config is the main config object
type Config struct {
	data map[string]interface{}
}

func New() *Config {
	return &Config{
		data: make(map[string]interface{}),
	}
}

func Load(filename string) (*Config, error) {

	// find the real name of the config file
	abspath, err := homedir.Expand(filename)
	if err != nil {
		return nil, errors.Wrap(err, "Can't expand homedir?")
	}

	filedata, err := ioutil.ReadFile(abspath)
	if err != nil {
		return nil, errors.Wrap(err, "Read error")
	}
	if filedata == nil {
		return nil, fmt.Errorf("No input found")
	}

	// Marshall to Yaml
	var data map[string]interface{}
	if err = yaml.Unmarshal(filedata, &data); err != nil {
		return nil, errors.Wrap(err, "YAML check error")
	}
	return &Config{data: data}, nil
}

// Get gets you a single datum.
//
// Args:
//   1: the key, as a string
//   2: optional default value, if key is not in the dictionary
func (c *Config) Get(args ...interface{}) interface{} {
	key := args[0].(string)

	// if there's valid data, return it
	if val, ok := c.data[key]; ok {
		return val
	}

	// if not, but we have a default, return that.
	// otherwise, return nil
	if len(args) > 1 {
		return args[1]
	} else {
		return nil
	}
}

// GetString is the same as Get but returns a value as a string
func (c *Config) GetString(args ...interface{}) string {
	value := c.Get(args...)

	// We might still have an empty value here, so
	// return the default if set
	if value == nil {
		if len(args) > 1 {
			return args[1].(string)
		} else {
			return ""
		}
	}

	return value.(string)
}

//
// Set will set a key to a given value.
//
// Args:
//    key: the key to set
//    value: the value to set it to
//
func (c *Config) Set(key string, value interface{}) {
	c.data[key] = value
}
