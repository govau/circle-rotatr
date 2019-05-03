package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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

func initUaaAPIs() {
	uaaAPIs = make(map[string]*uaa.API)

	for _, cf := range settings.Cfs {
		suffix := cf.Suffix
		target := cf.UaaHref
		clientID := getEnvVar(fmt.Sprintf("UAA_CLIENT_ID_%s", suffix))
		clientSecret := getEnvVar(fmt.Sprintf("UAA_CLIENT_SECRET_%s", suffix))
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
		uaaAPIs[suffix] = uaaAPI
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
