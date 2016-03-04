package main

// Simple AMI query tool: uses basic loops

import (
	"flag"
	"fmt"
	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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

	flag.StringVar(&c.keys_raw, "key", "", "Access key")
	flag.StringVar(&c.region, "region", "", "Region")
	flag.StringVar(&c.secret_keys_raw, "secret_key", "", "Secret Access key")
	flag.StringVar(&c.ami, "ami", "", "AMI to find")
	flag.Parse()

	// Extract into slices
	c.secret_keys = strings.Split(c.secret_keys_raw, ",")
	c.keys = strings.Split(c.keys_raw, ",")
	fmt.Println()

	// Loop through all of the accounts, search for instance
	for i, k := range c.keys {
		log.Println("Checking another account ", k)
		config := &aws.Config{
			Region:      aws.String(c.region),
			Credentials: credentials.NewStaticCredentials(k, c.secret_keys[i], ""),
		}
		sess := session.New(config)
		svc := ec2.New(sess)

		if queryAmi(svc, c.ami) {
			os.Exit(0)
		}
	}
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
		return true
	}
	return false
}

func checkError(err error) {
	if err != nil {
		log.Fatalf("Error: ", err)
	}
}
