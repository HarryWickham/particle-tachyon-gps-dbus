# Particle Tachyon GNSS Reader

Main GNSS reader logic is implemented in [`gnss_dbus.go`](./gnss_dbus.go). This requies access to `/var/run/dbus/system_bus_socket` on the Tachyon.

Use the docker-compose for simple startup, demo [here](./docker-compose.yml)

## Environment variables:

- `MQTT_BROKER_PORT` 
- `MQTT_BROKER_URL` I'm using a secure broker so this must be a domain to allow certificates to be verified.
- `MQTT_TOPIC`
- `MQTT_USERNAME`
- `MQTT_PASSWORD`

## Docker image:

[ghcr.io/harrywickham/particle-tachyon-gps-dbus](https://github.com/HarryWickham/particle-tachyon-gps-dbus/pkgs/container/particle-tachyon-gps-dbus)

For arm64. Demo compose file [here](./production.docker-compose.yml)