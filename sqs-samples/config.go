package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
)

type AppConfig struct {
	client *appconfigdata.Client
	token  *string
}

func NewAppConfig() *AppConfig {
	return &AppConfig{}
}

func (c *AppConfig) getConfig() *appconfigdata.GetLatestConfigurationInput {
	return &appconfigdata.GetLatestConfigurationInput{ConfigurationToken: c.token}
}

func (c *AppConfig) fetchAppConfig(ctx context.Context, awsCfg aws.Config) error {

	// time.Sleep(500 * time.Millisecond)
	// time.Sleep(5 * time.Second)

	c.client = appconfigdata.NewFromConfig(awsCfg)

	// Define your AppConfig details
	applicationID := "meolhvk"
	environmentID := "testing"
	configurationProfileID := "m0sw0id"

	sessionConf := appconfigdata.StartConfigurationSessionInput{
		ApplicationIdentifier:          aws.String(applicationID),
		ConfigurationProfileIdentifier: aws.String(configurationProfileID),
		EnvironmentIdentifier:          aws.String(environmentID),
	}

	// Start the configuration session
	configSession, err := c.client.StartConfigurationSession(ctx, &sessionConf)
	if err != nil {
		fmt.Printf("failed to get start session %v", err)
		logger(fmt.Sprintf("failed to get start session %v", err))
		return err
	}

	// Save the initial token
	c.token = configSession.InitialConfigurationToken

	// Fetch the configuration
	latestConf, err := c.client.GetLatestConfiguration(ctx, c.getConfig())
	if err != nil {
		logger(fmt.Sprintf("failed to get initial config %v", err))
		return err
	}

	// Print the configuration
	logger(fmt.Sprintf("Configuration: %v", string(latestConf.Configuration)))

	// Update token for the next fetch
	err = c.updateConfigAndNextToken(latestConf)
	if err != nil {
		logger(fmt.Sprintf("failed to update token %v", err))
		return err
	}

	// After initial config is fetched, start periodic fetching
	go c.startPeriodicFetch(ctx)

	return nil
}

func (c *AppConfig) updateConfigAndNextToken(latestConf *appconfigdata.GetLatestConfigurationOutput) error {
	c.token = latestConf.NextPollConfigurationToken
	return nil
}

// This function will fetch the configuration every 10 seconds after the first configuration
func (c *AppConfig) startPeriodicFetch(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Directly fetch the latest configuration
			latestConf, err := c.client.GetLatestConfiguration(ctx, c.getConfig())
			if err != nil {
				logger(fmt.Sprintf("Error fetching config: %v\n", err))
				return
			}

			// Print the configuration
			logger(fmt.Sprintf("Latest Configuration: %v", string(latestConf.Configuration)))

			// Update token for the next fetch
			err = c.updateConfigAndNextToken(latestConf)
			if err != nil {
				logger(fmt.Sprintf("failed to update token %v", err))
				return
			}
		case <-ctx.Done():
			return
		}
	} // comment
}
