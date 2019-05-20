package main

import (
	"log"
  "fmt"

	circleci "github.com/jszwedko/go-circleci"
)

//Circle client instance
type Circle struct {
	Client circleci.Client
}

//SetEnvVar on the given project. If it already exists, it is recreated with the new value.
func (c *Circle) SetEnvVar(account string, repoName string, name string, value string) error {
	envVars, err := c.Client.ListEnvVars(account, repoName)
	if err != nil {
		return err
	}
	for _, envVar := range envVars {
		if envVar.Name == name {
			err := c.Client.DeleteEnvVar(account, repoName, envVar.Name)
			if err != nil {
				return err
			}
		}
	}
	_, err = c.Client.AddEnvVar(account, repoName, name, value)
	return err
}

//NewCircle Create new Circle instance
func NewCircle(circleToken string) (*Circle, error) {
	circle := &Circle{}

	circle.Client = circleci.Client{Token: circleToken}

	// todo find a better way to test our circle-token?
	_, err := circle.Client.ListRecentBuildsForProject("govau", "cga-docs", "", "", -1, 0)
	if err != nil {
		log.Fatalln("Problem with circle token", err)
	}
	return circle, nil
}

// EnsureEnvVarsExist make sure that all env vars are set on the given circle project
func (c *Circle) EnsureEnvVarsExist(account string, repoName string, desiredEnvVars map[string]string) error {
  envVars, err := c.Client.ListEnvVars(account, repoName)
	if err != nil {
		return err
  }

  for key, value := range desiredEnvVars {
    found := false
    for _, envVar := range envVars {
      if envVar.Name == key {
        found = true
        break
      }
    }
    if !found {
      _, err = c.Client.AddEnvVar(account, repoName, key, value)
      if err != nil {
        return fmt.Errorf("Problem adding environment variable: %v", err)
      }
    }
  }
	
	return nil
}
