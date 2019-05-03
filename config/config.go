package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Settings The application settings
type Settings struct {
	Cfs  []Cf
	Orgs []CfOrg
}

// Cf CloudFoundry API settings
type Cf struct {
	UaaHref string `yaml:"uaa_href"`
	Suffix  string
}

// CfOrg CloudFoundry Organisation settings
type CfOrg struct {
	Name   string
	Spaces []CfSpace
}

// CfSpace CloudFoundry Space settings
type CfSpace struct {
	Name  string
	Repos []string
}

// Load Populate the given Settings from the given file
func Load(filename string, settings *Settings) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(content, settings)
	if err != nil {
		return err
	}

	//todo add some validation?
	return nil
}
