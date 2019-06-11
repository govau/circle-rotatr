package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/govau/torque/config"
)

var (
	configFile = flag.String("config.file", "config.yaml", "Path to configuration file.")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	settings   = &config.Settings{}
	cfInfos    = map[string]*CfInfo{}
)

func getEnvVar(key string) string {
	value, present := os.LookupEnv(key)
	if !present {
		log.Fatalf("ERROR: Must set %s environment variable", key)
	}
	return value
}

func initCfInfos() {
	for _, cf := range settings.Cfs {
		newCfInfo, err := NewCfInfo(cf.ID, cf.APIHref, settings.UaaOrigin)
		if err != nil {
			log.Fatalln(err)
		}
		cfInfos[cf.ID] = newCfInfo
	}
}

// ensureStaticCircleEnvVarsSet ensure the circle project has all the static environment variables
// a project needs to deploy to cf. This is all the env vars except the password
// Since we cannot read env vars from CircleCI, if they already exist we do not touch them.
func ensureStaticCircleEnvVarsSet(circle *Circle, cfOrg string, cfSpace string, skipIDs []string, repo string) error {
	if *verbose {
		log.Printf("Ensuring static circle env vars exist for %s", repo)
	}
	desiredEnvVars := map[string]string{
		"CF_ORG":      cfOrg,
		"CF_SPACE":    cfSpace,
		"CF_USERNAME": cfUserName(cfOrg, cfSpace),
	}

	// Add the CF_API_* env vars for each CF this repo will deploy to
	for id, cfInfo := range cfInfos {
		skip := false
		for _, skipID := range skipIDs {
			if skipID == id {
				skip = true
				break
			}
		}
		if !skip {
			desiredEnvVars[fmt.Sprintf("CF_API_%s", id)] = cfInfo.APIHref
		}
	}

	err := circle.AddEnvVarIfNotAlreadySet(repo, desiredEnvVars)
	if err != nil {
		return err
	}

	return nil
}

func main() {

	flag.Parse()

	if *verbose {
		log.Println("started")
	}

	if err := config.LoadFile(*configFile, settings); err != nil {
		log.Fatalf("Problem loading config: %s\n", err)
	}

	if *verbose {
		log.Printf("Using config: %+v", settings)
	}

	initCfInfos()

	circleToken := getEnvVar("CIRCLE_TOKEN")
	circle, err := NewCircle(circleToken)
	if err != nil {
		log.Fatalln(err)
	}
	passwordsRotated := 0
	numberOfCircleCIRepos := 0

	for _, cfOrg := range settings.Orgs {
		for _, cfSpace := range cfOrg.Spaces {
			for _, repo := range cfSpace.Repos {
				if err := circle.EnsureProjectEnabled(repo); err != nil {
					log.Fatalf("Problem ensuring %s was being built in Circle: %v", repo, err)
				}

				if err := ensureStaticCircleEnvVarsSet(circle, cfOrg.Name, cfSpace.Name, cfSpace.SkipIDs, repo); err != nil {
					log.Fatalf("Problem ensuring static circle env vars were set in %s: %v", repo, err)
				}

				numberOfCircleCIRepos++
			}

			for id, cfInfo := range cfInfos {
				skip := false
				for _, skipID := range cfSpace.SkipIDs {
					if id == skipID {
						skip = true
						break
					}
				}
				if skip {
					if *verbose {
						log.Printf("Skipping %s", id)
					}
					continue
				}

				newPassword, err := cfInfo.RotateCIUserPassword(cfOrg.Name, cfSpace.Name)
				if err != nil {
					log.Fatalf("Problem rotating ci user password %s %s: %v", cfOrg.Name, cfSpace.Name, err)
				}
				envVarName := fmt.Sprintf("CF_PASSWORD_%s", id)

				// Set the new password for each of the repos in circleci
				for _, repo := range cfSpace.Repos {
					err := circle.SetEnvVar(repo, envVarName, newPassword)
					if err != nil {
						log.Fatalf("Error setting new password in circle for %s\n", repo)
					}
				}

				passwordsRotated++
				if *verbose {
					log.Printf("Successfully rotated ci user %s %s %s", cfInfo.UaaAPI.TargetURL, cfOrg.Name, cfSpace.Name)
				}
			}
		}
	}

	fmt.Printf("Rotated %d ci user passwords in %d circleci repos\n", passwordsRotated, numberOfCircleCIRepos)
	if *verbose {
		log.Println("finished")
	}
}
