package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

	uaa "github.com/cloudfoundry-community/go-uaa"
)

// Space CloudFoundry space mapped to circleci projects
type Space struct {
	Name      string
	Org       string
	Repos     []string
	UaaOrigin string
}

func generateNewPassword() string {
	buf := make([]byte, 40)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

func repoToAccountAndProject(s string) (string, string, error) {
	stringSlice := strings.Split(s, "/")
	if len(stringSlice) != 2 {
		return "", "", fmt.Errorf("bad repo string: %s. Must be like 'org/foo'", s)
	}
	return stringSlice[0], stringSlice[1], nil
}

func (s *Space) cfUserName() string {
	return fmt.Sprintf("ci-%s-%s", s.Org, s.Name)
}

// EnsureCircleEnvVarsSet ensure the circle project has all environment variable
// a project needs to deploy to cf
func (s *Space) EnsureCircleEnvVarsSet(circle *Circle) error {

	desiredEnvVars := map[string]string{
		"CF_ORG":      s.Org,
		"CF_SPACE":    s.Name,
		"CF_USERNAME": s.cfUserName(),
	}
	for _, cf := range settings.Cfs {
		desiredEnvVars[fmt.Sprintf("CF_API_%s", cf.ID)] = cf.APIHref
	}

	for _, repo := range s.Repos {
		account, project, err := repoToAccountAndProject(repo)
		if err != nil {
			return err
		}

		err = circle.EnsureEnvVarsExist(account, project, desiredEnvVars)
		if err != nil {
			return err
		}
	}

	return nil
}

//Rotate the password of the ci user in this space, and write it to each
//circleci project
func (s *Space) Rotate(uaaAPI *uaa.API, circle *Circle, envVarName string) error {
	username := s.cfUserName()
	attributes := ""

	user, err := uaaAPI.GetUserByUsername(username, s.UaaOrigin, attributes)
	if err != nil {
		log.Printf("Error getting user %s\n", username)
		return err
	}

	// todo confirm this behaviour when user doesnt exist
	if user == nil {
		return fmt.Errorf("user %s not found in UAA", username)
	}

	newPassword := generateNewPassword()

	err = uaaAPI.SetPassword(newPassword, "", user.ID)
	if err != nil {
		log.Println("Error changing password")
		return err
	}

	for _, repo := range s.Repos {
		account, project, err := repoToAccountAndProject(repo)
		if err != nil {
			return err
		}
		err = circle.SetEnvVar(account, project, envVarName, newPassword)
		if err != nil {
			log.Printf("Error setting new password in circle for %s\n", repo)
			return err
		}
	}

	log.Printf("Successfully rotated %s - %s", uaaAPI.TargetURL, username)
	return nil
}
