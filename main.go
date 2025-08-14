package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	// MaxSatelliteCount defines the maximum number of satellites that can be tracked
	MaxSatelliteCount = 12
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}
}

// getEnv retrieves an environment variable value and returns an error if it's missing
func getEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	return val, nil
}

func main() {
	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()

	// Load environment variables with error handling
	mqttBrokerPort, err := getEnv("MQTT_BROKER_PORT")
	if err != nil {
		log.Fatalf("Environment setup failed: %v", err)
	}
	mqttBrokerURL, err := getEnv("MQTT_BROKER_URL")
	if err != nil {
		log.Fatalf("Environment setup failed: %v", err)
	}
	mqttTopic, err := getEnv("MQTT_TOPIC")
	if err != nil {
		log.Fatalf("Environment setup failed: %v", err)
	}
	mqttUsername, err := getEnv("MQTT_USERNAME")
	if err != nil {
		log.Fatalf("Environment setup failed: %v", err)
	}
	mqttPassword, err := getEnv("MQTT_PASSWORD")
	if err != nil {
		log.Fatalf("Environment setup failed: %v", err)
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("Failed to load system cert pool: %v", err)
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%s", mqttBrokerURL, mqttBrokerPort))
	opts.SetUsername(mqttUsername)
	opts.SetPassword(mqttPassword)
	opts.SetTLSConfig(&tls.Config{RootCAs: rootCAs})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connection error: %v", token.Error())
	}
	log.Println("Connected to MQTT broker")

	gnss := GNSSDbus{}

	if err := gnss.Connect(); err != nil {
		log.Fatalf("Failed to connect to D-Bus: %v", err)
	}

	// Main processing loop with graceful shutdown support
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully...")
			client.Disconnect(250) // Wait up to 250ms for clean disconnect
			return
		case <-ticker.C:
			data, err := gnss.GetData()
			if err != nil {
				log.Printf("Failed to get GNSS data: %v", err)
				continue
			}
			if data != nil {
				payload, err := json.Marshal(data)
				if err != nil {
					log.Printf("Failed to marshal GNSS data: %v", err)
					continue
				}
				token := client.Publish(fmt.Sprintf("%s/gnss", mqttTopic), 0, false, payload)
				token.Wait()
				if token.Error() != nil {
					log.Printf("Failed to publish GNSS data: %v", token.Error())
				} else {
					log.Printf("Published full GNSS data to MQTT %s", time.Now().UTC())
				}
			}
		}
	}
}
