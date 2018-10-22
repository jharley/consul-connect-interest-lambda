package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-redis/redis"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
)

const myConnectServiceName = "balance-interest-lambda"

const maxInterestIncrease = 40

var (
	myAccountIDs     = [...]int{1, 23, 383, 82}
	redisServiceName = os.Getenv("REDIS_SERVICE_NAME")
	consulServer     = os.Getenv("CONSUL_SERVER")
)

func getConsulServer() string {
	// TODO: update this to use Tag-based discovery with the AWS Go SDK
	//   n.b. go-discover didn't seem to love running in a Lambda environment
	if consulServer == "" {
		consulServer = "localhost"
	}
	return fmt.Sprintf("%s:8500", consulServer)
}

func updateAccountInterest(accountID int, client *redis.Client) {
	log.Println("[DEBUG] Updating account interest for ", accountID)
	accountKey := fmt.Sprintf("balance-%03d", accountID)
	randomIncrement := calculateInterest(accountID)

	cmd := client.IncrBy(accountKey, randomIncrement)
	if cmd.Err() != nil {
		log.Printf("[ERROR] Unable to update account %d: %v", accountID, cmd.Err())
	}
	log.Printf("[INFO] Account %d now has a balance of %d\n", accountID, cmd.Val())
}

func calculateInterest(accountID int) int64 {
	interest := int64(rand.Intn(maxInterestIncrease))
	log.Printf("[INFO] Account %d earns %d in interest", accountID, interest)
	return interest
}

/*
Handler contacts a redis service via Consul Connect's native integration

All provided accounts have their balance updated with random interest
*/
func Handler() (string, error) {
	log.Println("[INFO] starting ", myConnectServiceName)
	if redisServiceName == "" {
		redisServiceName = "redis"
	}

	config := api.DefaultConfig()
	config.Address = fmt.Sprintf("http://%s", getConsulServer())
	log.Println("[DEBUG] Determined Consul address to be ", config.Address)

	consulClient, err := api.NewClient(config)
	if err != nil {
		return "", fmt.Errorf("Unable to instantiate Consul API client: %v", err)
	}
	log.Println("[INFO] Configured Consul API client against ", config.Address)

	connectSvc, err := connect.NewService(myConnectServiceName, consulClient)
	if err != nil {
		return "", fmt.Errorf("Unable to instantiate Consul Connect Service: %v", err)
	}
	defer connectSvc.Close()
	log.Println("[DEBUG] Successfully registered Consul Connect Service \"", myConnectServiceName, "\"")

	// Create a new redis client, but use Consul Connect as the transport
	conn := redis.NewClient(&redis.Options{
		Dialer: func() (net.Conn, error) {
			return connectSvc.Dial(context.Background(), &connect.ConsulResolver{
				Client: consulClient,
				Name:   redisServiceName,
			})
		},
	})
	defer conn.Close()

	// Ensure we're connected to Redis via Consul Connect
	_, err = conn.Ping().Result()
	if err != nil {
		return "", fmt.Errorf("Unable to ping Redis service: %v", err)
	}

	// Update the interest for each provided account
	for _, account := range myAccountIDs {
		log.Println("[INFO] Updating interest balance for account ", account)
		updateAccountInterest(account, conn)
	}

	return "Account balances updated successfully", nil
}

func main() {
	if _, lambdaEnv := os.LookupEnv("LAMBDA_TASK_ROOT"); lambdaEnv == true {
		// We're properly running in AWS Lambda
		lambda.Start(Handler)
	} else {
		// let's assume we're running interactively
		output, err := Handler()
		if err != nil {
			fmt.Printf("[ERROR] %v", err)
		}

		fmt.Println(output)
	}
}
