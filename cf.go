package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	uaa "github.com/cloudfoundry-community/go-uaa"
)

// CfInfo CloudFoundry instance
type CfInfo struct {
	ID        string
	APIHref   string
	UaaOrigin string
	UaaAPI    *uaa.API
}

//NewCfInfo Create new CfInfo instance. A UaaAPI client is created and tested, and any error is returned.
func NewCfInfo(ID string, APIHref string, UaaOrigin string) (*CfInfo, error) {
	newCfInfo := &CfInfo{
		ID:        ID,
		APIHref:   APIHref,
		UaaOrigin: settings.UaaOrigin,
	}

	target, err := apiToUaaHref(APIHref)
	if err != nil {
		log.Fatalln(err)
	}
	clientID := getEnvVar(fmt.Sprintf("UAA_CLIENT_ID_%s", ID))
	clientSecret := getEnvVar(fmt.Sprintf("UAA_CLIENT_SECRET_%s", ID))
	zoneID := ""
	uaaAPI, err := uaa.NewWithClientCredentials(target, zoneID, clientID, clientSecret, uaa.JSONWebToken, false)
	if err != nil {
		log.Fatalln(err)
	}
	newCfInfo.UaaAPI = uaaAPI

	value, present := os.LookupEnv("UAA_VERBOSE")
	if present && value != "0" {
		newCfInfo.UaaAPI.Verbose = true
	}

	// test the connection
	users, err := newCfInfo.UaaAPI.ListAllUsers("", "", "", uaa.SortAscending)
	if err != nil {
		log.Fatalln(err)
	}
	if *verbose {
		log.Printf("Found %d users in uaa", len(users))
	}
	if len(users) == 0 {
		log.Println("Found 0 users in uaa, something may be wrong. Continuing")
	}

	return newCfInfo, nil
}

// Query a cf api endpoint for its uaa href
func apiToUaaHref(apiHref string) (string, error) {
	if *verbose {
		log.Printf("Getting uaa href from %s", apiHref)
	}
	type APILink struct {
		// HREF is the fully qualified URL for the link.
		HREF string `json:"href"`
	}

	type InfoLinks struct {
		// UAA is the link to the UAA API.
		UAA APILink `json:"uaa"`
	}

	var ccResourceLinks struct {
		Links map[string]APILink `json:"links"`
	}

	client := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, apiHref, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, getErr := client.Do(req)
	if getErr != nil {
		return "", err
	}
	defer res.Body.Close()

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return "", err
	}

	err = json.Unmarshal(body, &ccResourceLinks)
	if err != nil {
		return "", err
	}
	uaaHref := ccResourceLinks.Links["uaa"].HREF

	if *verbose {
		log.Printf("Got uaa href %s", uaaHref)
	}

	return uaaHref, nil
}

// RotateCIUserPassword changes the password of the CI user for this cf org and space
func (cf *CfInfo) RotateCIUserPassword(cfOrg string, cfSpace string) (string, error) {
	username := cfUserName(cfOrg, cfSpace)
	if *verbose {
		log.Printf("Rotating password for %s", username)
	}

	attributes := ""
	user, err := cf.UaaAPI.GetUserByUsername(username, cf.UaaOrigin, attributes)
	if err != nil {
		// log.Printf("Error getting user %s\n", username)
		return "", fmt.Errorf("Error getting user %s: %v", username, err)
	}

	// todo confirm this behaviour when user doesnt exist
	if user == nil {
		return "", fmt.Errorf("Unable to fetch user %s, maybe it does not exist in UAA: %s", username, cf.UaaAPI.TargetURL)
	}

	newPassword := generateNewPassword()

	err = cf.UaaAPI.SetPassword(newPassword, "", user.ID)
	if err != nil {
		log.Println("Error changing password")
		return "", err
	}

	if *verbose {
		log.Printf("Set password succeeded")
	}

	return newPassword, nil
}

func cfUserName(org string, space string) string {
	return fmt.Sprintf("ci-%s-%s", org, space)
}

func generateNewPassword() string {
	buf := make([]byte, 40)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}
