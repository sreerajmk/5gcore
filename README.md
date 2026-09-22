# 5G Core Microservice Skeleton

This project implements a minimal 5G core microservice architecture in Python using FastAPI.

## Network functions

- AMF: Access and Mobility Management Function
- SMF: Session Management Function
- AUSF: Authentication Server Function
- UPF: User Plane Function
- UDM: Unified Data Management
- NRF: Network Repository Function / service discovery anchor

## How it works

- Each network function exposes a FastAPI app with `/health`, `/status`, and registration endpoints.
- The NRF keeps an in-memory registry of all active network functions.
- Network functions can register with the NRF at startup and query the NRF for other NFs when needed.

## Run locally

```bash
python -m pip install -r requirements.txt
python run_all.py
```

The default ports are:

- NRF: 8000
- AMF: 8001
- SMF: 8002
- AUSF: 8003
- UPF: 8004
- UDM: 8005
