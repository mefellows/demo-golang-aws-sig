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
	"sync"
)

var total int

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

	countChan := make(chan int, len(c.secret_keys))
	finish := make(chan bool)
	var done sync.WaitGroup
	done.Add(len(c.keys))
	go countStuff(countChan, done)

	// Loop through all of the accounts, search for instance in parallel
	for i, k := range c.keys {
		go func(i int, key string) {
			log.Println("Querying account ", key)
			svc := ec2.New(&aws.Config{
				Region:      aws.String(c.region),
				Credentials: credentials.NewStaticCredentials(key, c.secret_keys[i], ""),
			})

			countChan <- getInstanceCount(svc)
			done.Done()
		}(i, k)
	}
	close(finish)
	done.Wait()
	log.Printf("Total: %d", total)

	// Profit
}

func countStuff(ch chan int, done sync.WaitGroup) {

	for {
		select {
		case v := <-ch:
			total = total + v
		}
	}

}

func getInstanceCount(service *ec2.EC2) int {
	params := &ec2.DescribeInstancesInput{
		MaxResults: aws.Int64(1024),
	}
	resp, err := service.DescribeInstances(params)
	checkError(err)
	count := 0
	for _, res := range resp.Reservations {
		count += len(res.Instances)
	}
	return count
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
