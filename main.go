package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/erikswanson/enable-docker-credential-ecr-login/dockerconfig"
)

func getEcrEndpoints(regionID, accountID string) []string {
	partitions := endpoints.DefaultPartitions()
	partition, ok := endpoints.PartitionForRegion(partitions, regionID)
	if !ok {
		return nil
	}

	regions, ok := endpoints.RegionsForService(partitions, partition.ID(), endpoints.EcrServiceID)
	if !ok {
		return nil
	}

	var result = make([]string, 0, len(regions))
	for _, region := range regions {
		re, err := region.ResolveEndpoint(endpoints.EcrServiceID)
		if err != nil {
			panic(err)
		}
		if suffix := strings.TrimPrefix(re.URL, "https://"); suffix != re.URL {
			result = append(result, fmt.Sprintf("%s.dkr.%s", accountID, suffix))
		}
	}
	return result
}

func getRegionAndAccount() (regionID, accountID string, err error) {
	awsSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return
	}
	client := sts.New(awsSession)
	regionID = client.SigningRegion
	output, err := client.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return
	}
	if output != nil && output.Account != nil {
		accountID = *output.Account
	}
	return
}

func main() {
	updater, err := dockerconfig.ForCurrentUser()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to determine AWS identity: %s\n", err.Error())
		os.Exit(1)
	}

	regionID, accountID, err := getRegionAndAccount()
	if err != nil {
		os.Exit(1)
	}
	if regionID == "" {
		println("ERROR: Failed to determine AWS region")
		os.Exit(1)
	}
	if accountID == "" {
		println("ERROR: Failed to determine AWS account")
		os.Exit(1)
	}
	registries := getEcrEndpoints(regionID, accountID)

	if len(registries) == 0 {
		println("WARNING: No ECR endpoints found for the current AWS partition")
		os.Exit(0)
	}

	updated, err := updater.EnsureCredHelpers("ecr-login", registries)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	if updated {
		fmt.Fprintf(os.Stdout, "Updated Docker config file: %s\n", updater.Path)
	}
}
