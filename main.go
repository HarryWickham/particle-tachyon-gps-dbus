package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/godbus/dbus/v5"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}
}

func getEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Missing required environment variable: %s", key)
	}
	return val
}

func main() {
	mqttBrokerPort := getEnv("MQTT_BROKER_PORT")
	mqttBrokerURL := getEnv("MQTT_BROKER_URL")
	mqttTopic := getEnv("MQTT_TOPIC")
	mqttUsername := getEnv("MQTT_USERNAME")
	mqttPassword := getEnv("MQTT_PASSWORD")

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

	for {
		data, err := getGnssData(conn)
		if err != nil {
			log.Printf("Failed to get GNSS data: %v", err)
		}
		if data != nil {
			payload, err := json.Marshal(data)
			if err != nil {
				log.Printf("Failed to marshal GNSS data: %v", err)
			}
			if err == nil {
				token := client.Publish(fmt.Sprintf("%s/gnss", mqttTopic), 0, false, payload)
				token.Wait()
				if token.Error() != nil {
					log.Printf("Failed to publish GNSS data: %v", token.Error())
				} else {
					log.Printf("Published full GNSS data to MQTT %s", time.Now().UTC())
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
}

type NmeaSatelliteMsg struct {
	Num    int8
	Eledeg int8
	Azideg int32
	SN     int8
}

type BeidouNmeaSatelliteMsg struct {
	BeidouNum    int8
	BeidouEledeg int8
	BeidouAzideg int32
	BeidouSN     int8
}

type NmeaUtcTime struct {
	Year  int32
	Month int8
	Date  int8
	Hour  int8
	Min   int8
	Sec   int8
}

type GnssFullData struct {
	Valid          int32
	LastLockTimeMs uint64
	Svnum          uint8
	BeidouSvnum    uint8
	NSHemi         string
	EWHemi         string
	Latitude       float64
	Longitude      float64
	Gpssta         uint8
	Posslnum       uint8
	Fixmode        uint8
	Pdop           float64
	Hdop           float64
	Vdop           float64
	Altitude       float64
	Speed          float64
	Utc            NmeaUtcTime
	Slmsg          [12]NmeaSatelliteMsg
	BeidouSlmsg    [12]BeidouNmeaSatelliteMsg
	Possl          [12]uint8
}

type GnssData struct {
	Latitude       float64
	Longitude      float64
	Speed          float64
	Valid          int32
	LastLockTimeMs uint64
	Svnum          uint8
	BeidouSvnum    uint8
	NSHemi         string
	EWHemi         string
	Altitude       float64
	Utc            NmeaUtcTime
	Slmsg          [12]NmeaSatelliteMsg
	BeidouSlmsg    [12]BeidouNmeaSatelliteMsg
	Possl          [12]uint8
}

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
			for i := 0; i < len(arr) && i < 12; i++ {
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
			for i := 0; i < len(arr) && i < 12; i++ {
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
			for i := 0; i < len(arr) && i < 12; i++ {
				data.Possl[i] = ToUint8(arr[i])
			}
		}
	}
	return &data, nil
}
