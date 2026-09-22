# 5G Core Microservice Demo

This project implements a simplified 5G core network function mesh in Go using a microservice design with an NRF registration and discovery anchor.

## Included network functions

- NRF: Network Repository Function
- AMF: Access and Mobility Management Function
- SMF: Session Management Function
- AUSF: Authentication Server Function
- UPF: User Plane Function
- UDM: Unified Data Management
- UE Simulator: lightweight subscriber emulator

## Realistic 5G flow

The AMF orchestrates a realistic attach / registration flow:

1. UE simulator sends an attach request to the AMF.
2. AMF discovers the AUSF through the NRF.
3. AUSF authenticates the subscriber and returns an auth token/security context.
4. AMF fetches subscriber profile data from the UDM.
5. AMF calls the SMF to create a PDU session.
6. SMF returns session context and UPF anchor details.
7. UPF receives a forwarding instruction and the session becomes active.
8. UE can later be released or deactivated, returning to idle/detached state.

This is intentionally simplified, but it mirrors the expected service interaction pattern used in a real 5G core.

## UE lifecycle model

The demo models these lifecycle states:

- `connected`
- `registered`
- `idle`
- `detached`
- PDU session `active` and `inactive`

The UE simulator exposes endpoints for lifecycle actions and status checks.

## Run with Go

```bash
cd /home/felicity/learn/5gcore

go run ./cmd/nrf

go run ./cmd/amf

go run ./cmd/smf

go run ./cmd/ausf

go run ./cmd/upf

go run ./cmd/udm

go run ./cmd/ue-sim
```

## Run with Docker Compose

```bash
docker compose up --build
```

Then test the attach procedure directly through the UE simulator:

```bash
curl -X POST http://localhost:8006/attach \
  -H 'Content-Type: application/json' \
  -d '{"imsi":"310150123456789"}'
```

Check session state:

```bash
curl http://localhost:8006/session-status
```

Release the session:

```bash
curl -X POST http://localhost:8006/release \
  -H 'Content-Type: application/json' \
  -d '{"imsi":"310150123456789"}'
```

## Quick script helper

A helper script is included to run the attach/status/release sequence:

```bash
cd /home/felicity/learn/5gcore
./ue-flow.sh
```

## Default ports

- NRF: 8000
- AMF: 8001
- SMF: 8002
- AUSF: 8003
- UPF: 8004
- UDM: 8005
- UE Simulator: 8006

## Project files

- `5gcore-ue-lifecycle.mmd`: Mermaid sequence diagram for attach and release flow
- `ue-flow.sh`: one-shot attach/status/release script
- `docker-compose.yml`: services for the full mock 5G core stack
- `Dockerfile`: multi-service Go image build
