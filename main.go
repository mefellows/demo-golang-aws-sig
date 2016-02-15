package main

import (
	"encoding/json"
	"flag"
	"fmt"
	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/vaughan0/go-ini"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

var verbose bool

func main() {

	type config struct {
		profiles []string
		region   string
		action   string
		query    string
		timeout  time.Duration
	}

	// Flags go in here
	c := &config{}

	flag.StringVar(&c.action, "action", "", "Action to perform (ami)")
	flag.StringVar(&c.action, "a", "", "Action to perform (ami)")
	flag.StringVar(&c.query, "query", "", "Query value (e.g. ami-1234)")
	flag.StringVar(&c.query, "q", "", "Query value (e.g. ami-1234)")
	flag.DurationVar(&c.timeout, "timeout", 5*time.Second, "Timeout e.g. 5s")
	flag.BoolVar(&verbose, "verbose", false, "Verbose logging?")
	flag.BoolVar(&verbose, "v", false, "Verbose logging?")
	flag.Parse()

	if c.action == "" && c.query == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Log verbosity
	if !verbose {
		log.SetOutput(ioutil.Discard)
	}

	c.profiles = listProfiles()

	// If we find our thing earlier
	doneChan := make(chan bool, 1)
	failChan := make(chan bool, 1)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	// Notify all async when complete
	go func() {
		var done sync.WaitGroup
		done.Add(len(c.profiles))

		// Loop through all of the accounts, search for instance in parallel
		for _, k := range c.profiles {
			go func(key string) {
				config := &aws.Config{
					Credentials: credentials.NewSharedCredentials("", key),
				}
				sess := session.New(config)
				svc := ec2.New(sess)

				var r interface{}
				switch strings.ToLower(c.action) {
				case "ami":
					r = queryAmi(svc, c.query)
				default:
					log.Fatalf("Action '%s' is not a valid action", c.action)
				}

				if r != nil {
					v, err := json.Marshal(r)
					checkError(err)
					fmt.Printf("%s", v)
					doneChan <- true
				}
				done.Done()
			}(k)
		}
		done.Wait()
		failChan <- true
	}()

	// Wait up to timeout, or when first result comes back
	select {
	case <-sigChan:
		log.Fatalf("Interrupt")
	case <-time.After(c.timeout):
		log.Fatalf("Timeout waiting for result")
	case <-failChan:
		log.Fatalf("No results returned")
	case <-doneChan:
		os.Exit(0)
	}
}

// Return true if AMI exists
func queryAmi(service *ec2.EC2, ami string) interface{} {
	input := ec2.DescribeImagesInput{
		ImageIds: []*string{&ami},
	}
	output, err := service.DescribeImages(&input)
	if len(output.Images) > 0 {
		checkError(err)
		image := output.Images[0]
		log.Printf("Found image in account: %s, with name: %s\n", *image.OwnerId, *image.Name)
		log.Printf("Tags: %v", image.Tags)
		return image
	}
	return nil
}

func listProfiles() []string {
	// Make sure the config file exists
	config := os.Getenv("HOME") + "/.aws/credentials"

	if _, err := os.Stat(config); os.IsNotExist(err) {
		fmt.Println("No credentials file found at: %s", config)
		os.Exit(1)
	}

	file, _ := ini.LoadFile(config)
	profiles := make([]string, 0)

	for key, _ := range file {
		profiles = append(profiles, key)
	}

	return profiles
}

func checkError(err error) {
	if err != nil {
		log.Fatalf("Error: ", err)
	}
}
