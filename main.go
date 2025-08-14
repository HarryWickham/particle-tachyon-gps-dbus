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
	"github.com/godbus/dbus/v5"
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

	conn, err := dbus.SystemBus()
	if err != nil {
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
			data, err := getGnssData(conn)
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

// NmeaSatelliteMsg represents NMEA satellite message data
type NmeaSatelliteMsg struct {
	Num    int8  // Satellite number
	Eledeg int8  // Elevation in degrees
	Azideg int32 // Azimuth in degrees
	SN     int8  // Signal-to-noise ratio
}

// BeidouNmeaSatelliteMsg represents Beidou NMEA satellite message data
type BeidouNmeaSatelliteMsg struct {
	BeidouNum    int8  // Beidou satellite number
	BeidouEledeg int8  // Beidou elevation in degrees
	BeidouAzideg int32 // Beidou azimuth in degrees
	BeidouSN     int8  // Beidou signal-to-noise ratio
}

// NmeaUtcTime represents UTC time information from NMEA data
type NmeaUtcTime struct {
	Year  int32 // Year
	Month int8  // Month (1-12)
	Date  int8  // Day of month (1-31)
	Hour  int8  // Hour (0-23)
	Min   int8  // Minutes (0-59)
	Sec   int8  // Seconds (0-59)
}

// GnssFullData represents complete GNSS data retrieved from the D-Bus interface
type GnssFullData struct {
	Valid          int32                                     // Validity flag for GPS data
	LastLockTimeMs uint64                                    // Last GPS lock time in milliseconds
	Svnum          uint8                                     // Number of satellites in view
	BeidouSvnum    uint8                                     // Number of Beidou satellites in view
	NSHemi         string                                    // North/South hemisphere indicator
	EWHemi         string                                    // East/West hemisphere indicator
	Latitude       float64                                   // Latitude coordinate
	Longitude      float64                                   // Longitude coordinate
	Gpssta         uint8                                     // GPS status
	Posslnum       uint8                                     // Position solution number
	Fixmode        uint8                                     // GPS fix mode
	Pdop           float64                                   // Position dilution of precision
	Hdop           float64                                   // Horizontal dilution of precision
	Vdop           float64                                   // Vertical dilution of precision
	Altitude       float64                                   // Altitude above sea level
	Speed          float64                                   // Ground speed
	Utc            NmeaUtcTime                               // UTC time information
	Slmsg          [MaxSatelliteCount]NmeaSatelliteMsg       // Satellite message data
	BeidouSlmsg    [MaxSatelliteCount]BeidouNmeaSatelliteMsg // Beidou satellite message data
	Possl          [MaxSatelliteCount]uint8                  // Position solution levels
}

// GnssData represents simplified GNSS data for publishing
type GnssData struct {
	Latitude       float64                                   // Latitude coordinate
	Longitude      float64                                   // Longitude coordinate
	Speed          float64                                   // Ground speed
	Valid          int32                                     // Validity flag for GPS data
	LastLockTimeMs uint64                                    // Last GPS lock time in milliseconds
	Svnum          uint8                                     // Number of satellites in view
	BeidouSvnum    uint8                                     // Number of Beidou satellites in view
	NSHemi         string                                    // North/South hemisphere indicator
	EWHemi         string                                    // East/West hemisphere indicator
	Altitude       float64                                   // Altitude above sea level
	Utc            NmeaUtcTime                               // UTC time information
	Slmsg          [MaxSatelliteCount]NmeaSatelliteMsg       // Satellite message data
	BeidouSlmsg    [MaxSatelliteCount]BeidouNmeaSatelliteMsg // Beidou satellite message data
	Possl          [MaxSatelliteCount]uint8                  // Position solution levels
}

// getGnssData retrieves GNSS data from the D-Bus interface and returns it as GnssFullData
func getGnssData(conn *dbus.Conn) (*GnssFullData, error) {
	obj := conn.Object("io.particle.tachyon.GNSS", "/io/particle/tachyon/GNSS/Modem")
	var result map[string]dbus.Variant
	if err := obj.Call("io.particle.tachyon.GNSS.Modem.GetGnss", 0).Store(&result); err != nil {
		return nil, err
	}
	data := GnssFullData{}
	// Scalar fields
	if v, ok := result["valid"]; ok {
		data.Valid, _ = v.Value().(int32)
	}
	if v, ok := result["last_lock_time_ms"]; ok {
		data.LastLockTimeMs, _ = v.Value().(uint64)
	}
	if v, ok := result["svnum"]; ok {
		data.Svnum, _ = v.Value().(uint8)
	}
	if v, ok := result["beidou_svnum"]; ok {
		data.BeidouSvnum, _ = v.Value().(uint8)
	}
	if v, ok := result["nshemi"]; ok {
		data.NSHemi, _ = v.Value().(string)
	}
	if v, ok := result["ewhemi"]; ok {
		data.EWHemi, _ = v.Value().(string)
	}
	if v, ok := result["latitude"]; ok {
		data.Latitude, _ = ParseFloatVariant(v)
	}
	if v, ok := result["longitude"]; ok {
		data.Longitude, _ = ParseFloatVariant(v)
	}
	if v, ok := result["gpssta"]; ok {
		data.Gpssta, _ = v.Value().(uint8)
	}
	if v, ok := result["posslnum"]; ok {
		data.Posslnum, _ = v.Value().(uint8)
	}
	if v, ok := result["fixmode"]; ok {
		data.Fixmode, _ = v.Value().(uint8)
	}
	if v, ok := result["pdop"]; ok {
		data.Pdop, _ = ParseFloatVariant(v)
	}
	if v, ok := result["hdop"]; ok {
		data.Hdop, _ = ParseFloatVariant(v)
	}
	if v, ok := result["vdop"]; ok {
		data.Vdop, _ = ParseFloatVariant(v)
	}
	if v, ok := result["altitude"]; ok {
		data.Altitude, _ = ParseFloatVariant(v)
	}
	if v, ok := result["speed"]; ok {
		data.Speed, _ = ParseFloatVariant(v)
	}
	// UTC time
	if v, ok := result["utc"]; ok {
		if utcArr, ok := v.Value().([]any); ok && len(utcArr) == 6 {
			data.Utc.Year = ToInt32(utcArr[0])
			data.Utc.Month = ToInt8(utcArr[1])
			data.Utc.Date = ToInt8(utcArr[2])
			data.Utc.Hour = ToInt8(utcArr[3])
			data.Utc.Min = ToInt8(utcArr[4])
			data.Utc.Sec = ToInt8(utcArr[5])
		}
	}
	// Satellite arrays
	if v, ok := result["slmsg"]; ok {
		if arr, ok := v.Value().([][]any); ok {
			for i := 0; i < len(arr) && i < MaxSatelliteCount; i++ {
				if len(arr[i]) == 4 {
					data.Slmsg[i].Num = ToInt8(arr[i][0])
					data.Slmsg[i].Eledeg = ToInt8(arr[i][1])
					data.Slmsg[i].Azideg = ToInt32(arr[i][2])
					data.Slmsg[i].SN = ToInt8(arr[i][3])
				}
			}
		}
	}
	if v, ok := result["beidou_slmsg"]; ok {
		if arr, ok := v.Value().([][]any); ok {
			for i := 0; i < len(arr) && i < MaxSatelliteCount; i++ {
				if len(arr[i]) == 4 {
					data.BeidouSlmsg[i].BeidouNum = ToInt8(arr[i][0])
					data.BeidouSlmsg[i].BeidouEledeg = ToInt8(arr[i][1])
					data.BeidouSlmsg[i].BeidouAzideg = ToInt32(arr[i][2])
					data.BeidouSlmsg[i].BeidouSN = ToInt8(arr[i][3])
				}
			}
		}
	}
	if v, ok := result["possl"]; ok {
		if arr, ok := v.Value().([]any); ok {
			for i := 0; i < len(arr) && i < MaxSatelliteCount; i++ {
				data.Possl[i] = ToUint8(arr[i])
			}
		}
	}
	return &data, nil
}
