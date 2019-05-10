package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"time"

	uaa "github.com/cloudfoundry-community/go-uaa"

	"github.com/govau/torque/config"
)

var (
	configFile = flag.String("config.file", "config.yaml", "Path to configuration file.")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	settings   = &config.Settings{}
	uaaAPIs    = map[string]*uaa.API{}
)

func getEnvVar(key string) string {
	value, present := os.LookupEnv(key)
	if !present {
		log.Fatalf("Must set %s environment variable", key)
	}
	return value
}

// CfAPIResponse is the expected response from querying a CloudFoundry API endpoint
// type CfAPIResponse struct {
// 	Links Uaa Uaa
// }

// type Uaa struct {
// 	href string
// }

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

func initUaaAPIs() {
	uaaAPIs = make(map[string]*uaa.API)

	for _, cf := range settings.Cfs {
		id := cf.ID
		target, err := apiToUaaHref(cf.APIHref)
		if err != nil {
			log.Fatalln(err)
		}
		clientID := getEnvVar(fmt.Sprintf("UAA_CLIENT_ID_%s", id))
		clientSecret := getEnvVar(fmt.Sprintf("UAA_CLIENT_SECRET_%s", id))
		zoneID := ""
		uaaAPI, err := uaa.NewWithClientCredentials(target, zoneID, clientID, clientSecret, uaa.JSONWebToken, false)
		if err != nil {
			log.Fatalln(err)
		}

		value, present := os.LookupEnv("UAA_VERBOSE")
		if (present && value != "0") || *verbose {
			uaaAPI.Verbose = true
		}

		// test the connection
		users, err := uaaAPI.ListAllUsers("", "", "", uaa.SortAscending)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("Found %d users in uaa", len(users))
		if len(users) == 0 {
			log.Println("Found 0 users in uaa, something may be wrong. Continuing")
		}
		uaaAPIs[id] = uaaAPI
	}
}

func main() {

	flag.Parse()

	log.Println("started")

	if err := config.Load(*configFile, settings); err != nil {
		log.Fatalln("Problem loading config", err)
	}

	log.Printf("Using config: %+v", settings)

	initUaaAPIs()

	circleToken := getEnvVar("CIRCLE_TOKEN")
	circle, err := NewCircle(circleToken)
	if err != nil {
		log.Fatalln("Problem using Circle", err)
	}

	for _, cfOrg := range settings.Orgs {
		for _, cfSpace := range cfOrg.Spaces {

			space := Space{
				Name:  cfSpace.Name,
				Org:   cfOrg.Name,
				Repos: cfSpace.Repos,
			}

			for suffix, api := range uaaAPIs {
				envName := fmt.Sprintf("CF_PASSWORD_%s", suffix)
				err := space.Rotate(api, circle, envName)
				if err != nil {
					log.Fatalln(err)
				}
			}
		}
	}

	log.Println("finished")
}
