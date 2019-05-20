package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// Settings The application settings
type Settings struct {
	UaaOrigin string `yaml:"uaa_origin"`
  Cfs  []Cf
	Orgs []CfOrg
}

// Cf CloudFoundry instance settings
type Cf struct {
	APIHref string `yaml:"api_href"`
	ID      string
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
	SkipIDs []string `yaml:"skip_ids"`
}

func validate(s *Settings) error {
	for _, cfOrg := range s.Orgs {
		for _, cfSpace := range cfOrg.Spaces {
			// Check SkipIDs exist in Cfs
			for _, skipID := range cfSpace.SkipIDs {
				found := false
				for _, cf := range s.Cfs {
					if skipID == cf.ID {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("Config: skipped ID not found in CFs: %s", skipID)
				}
			}
		}
	}
	return nil
}

// Load settings from the given io.Reader
func Load(reader io.Reader, settings *Settings) error {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	err = yaml.UnmarshalStrict(bytes, settings)
	if err != nil {
		return err
	}

	return validate(settings)
}

// LoadFile load settings from the given file
func LoadFile(file string, settings *Settings) error {
	handle, err := os.Open(file)
	if err != nil {
		return err
	}
	defer handle.Close()
	return Load(handle, settings)
}
