package main

// Simple AMI query tool: uses basic loops

import (
	"flag"
	"fmt"
	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"os"
	"strings"
)

func main() {

	type config struct {
		keys_raw        string
		secret_keys_raw string
		region          string
		keys            []string
		secret_keys     []string
		ami             string
	}

	// Get arguments
	c := &config{}

	flag.StringVar(&c.region, "region", "", "Region")
	flag.StringVar(&c.region, "r", "", "Region")
	flag.StringVar(&c.ami, "ami", "", "AMI to find")
	flag.StringVar(&c.ami, "a", "", "AMI to find")
	flag.StringVar(&c.secret_keys_raw, "secret_key", "", "Secret Access key")
	flag.StringVar(&c.secret_keys_raw, "s", "", "Secret Access key")
	flag.StringVar(&c.keys_raw, "key", "", "Access key")
	flag.StringVar(&c.keys_raw, "k", "", "Access key")
	flag.Parse()

	if c.region == "" || c.ami == "" || c.secret_keys_raw == "" || c.keys_raw == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Extract into slices
	c.secret_keys = strings.Split(c.secret_keys_raw, ",")
	c.keys = strings.Split(c.keys_raw, ",")
	fmt.Println()

	done := make(chan bool)

	// Loop through all of the accounts, search for instance in parallel
	for i, k := range c.keys {
		go func() {
			log.Println("Querying account ", k)
			svc := ec2.New(&aws.Config{
				Region:      aws.String(c.region),
				Credentials: credentials.NewStaticCredentials(k, c.secret_keys[i], ""),
			})
			if queryAmi(svc, c.ami) {
				done <- true
			}
		}()
	}

	<-done
	log.Printf("Exiting")

	// Profit
}

// Return true if AMI exists
func queryAmi(service *ec2.EC2, ami string) bool {
	input := ec2.DescribeImagesInput{
		ImageIds: []*string{&ami},
	}
	output, err := service.DescribeImages(&input)
	if len(output.Images) > 0 {
		checkError(err)
		image := output.Images[0]
		log.Printf("Found image in account: %s, with name: %s\n", *image.OwnerId, *image.Name)
		log.Printf("Tags: %v", image.Tags)
		return true
	}
	return false
}

func checkError(err error) {
	if err != nil {
		log.Fatalf("Error: ", err)
	}
}
