# 5G Core Microservice Demo

This project implements a simplified 5G core network function mesh in Go using a microservice design with an NRF registration and discovery anchor.

## Included network functions

- NRF: Network Repository Function
- AMF: Access and Mobility Management Function
- SMF: Session Management Function
- AUSF: Authentication Server Function
- UPF: User Plane Function
- UDM: Unified Data Management

## Realistic 5G flow

The AMF orchestrates a realistic attach / registration flow:

1. UE sends an attach request to the AMF.
2. AMF discovers the AUSF through the NRF.
3. AUSF authenticates the subscriber and returns an auth token.
4. AMF fetches subscriber profile data from the UDM.
5. AMF calls the SMF to create a PDU session.
6. SMF returns session context and UPF anchor details.

This is intentionally simplified, but it mirrors the expected service interaction pattern used in a real 5G core.

## Run with Go

```bash
cd /home/felicity/learn/5gcore

go run ./cmd/nrf

go run ./cmd/amf

go run ./cmd/smf

go run ./cmd/ausf

go run ./cmd/upf

go run ./cmd/udm
```

## Run with Docker Compose

```bash
docker compose up --build
```

Then test the attach procedure:

```bash
curl -X POST http://localhost:8001/procedure/attach \
  -H 'Content-Type: application/json' \
  -d '{"imsi":"310150123456789"}'
```

## Default ports

- NRF: 8000
- AMF: 8001
- SMF: 8002
- AUSF: 8003
- UPF: 8004
- UDM: 8005
