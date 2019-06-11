package main

import (
	"fmt"
	"log"
	"strings"

	circleci "github.com/jszwedko/go-circleci"
)

//Circle client instance
type Circle struct {
	Client circleci.Client
}

//NewCircle Create new Circle instance. The circleci token is tested and any error is returned.
func NewCircle(circleToken string) (*Circle, error) {
	circle := &Circle{}

	circle.Client = circleci.Client{Token: circleToken}

	user, err := circle.Client.Me()
	if err != nil {
		return nil, fmt.Errorf("Bad circle token: %v", err)
	}

	if *verbose {
		log.Printf("Circle Token belongs to user %s", user.Login)
	}

	// test the token
	if _, err := circle.Client.ListProjects(); err != nil {
		return nil, fmt.Errorf("Problem testing circle token: %v", err)
	}

	return circle, nil
}

// EnsureProjectEnabled ensure this project is being built in CircleCI
func (c *Circle) EnsureProjectEnabled(orgAndRepo string) error {
	if *verbose {
		log.Printf("Ensuring circleci is building this repo: %s", orgAndRepo)
	}
	org, repo, err := SplitOrgAndRepo(orgAndRepo)
	if err != nil {
		return err
	}
	return c.Client.EnableProject(org, repo)
}

//SetEnvVar on the given project. If it already exists, it is recreated with the new value.
func (c *Circle) SetEnvVar(orgAndRepo string, name string, value string) error {
	org, repo, err := SplitOrgAndRepo(orgAndRepo)
	if err != nil {
		return err
	}

	envVars, err := c.Client.ListEnvVars(org, repo)
	if err != nil {
		return err
	}
	for _, envVar := range envVars {
		if envVar.Name == name {
			err := c.Client.DeleteEnvVar(org, repo, envVar.Name)
			if err != nil {
				return err
			}
		}
	}
	_, err = c.Client.AddEnvVar(org, repo, name, value)
	return err
}

// AddEnvVarIfNotAlreadySet set environment variables if not already set in CircleCI for this repo
func (c *Circle) AddEnvVarIfNotAlreadySet(orgAndRepo string, desiredEnvVars map[string]string) error {
	org, repo, err := SplitOrgAndRepo(orgAndRepo)

	if err != nil {
		return err
	}

	envVars, err := c.Client.ListEnvVars(org, repo)
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
			_, err = c.Client.AddEnvVar(org, repo, key, value)
			if err != nil {
				return fmt.Errorf("Problem adding environment variable to %s: %v", orgAndRepo, err)
			}
		}
	}

	return nil
}

// SplitOrgAndRepo split a single string with org and repo to separate strings.
// e.g. govau/torque will return govau,torque
func SplitOrgAndRepo(s string) (string, string, error) {
	stringSlice := strings.Split(s, "/")
	if len(stringSlice) != 2 {
		return "", "", fmt.Errorf("bad repo string: %s. Must be like 'org/foo'", s)
	}
	return stringSlice[0], stringSlice[1], nil
}
