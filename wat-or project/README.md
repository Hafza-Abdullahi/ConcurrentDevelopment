# Wa-Tor Simulation (Go) 

A concurrent implementation of A.K. Dewdney's Wa-Tor simulation using the Go language. This project utilizes the Fork-Join pattern and Stencil pattern to simulate an ecological war between Sharks and Fish on a toroidal grid.

## How to Build
To build the executable:
```bash
go build -o wator.exe main.go

## To run the simulation with visualization
.\wator.exe -NumFish=200 -NumShark=20 -GridSize=30 -Threads=1

## Performace benchmarking, bigger grid
.\wator.exe -GridSize=200 -NumFish=2000 -NumShark=500 -Threads=4
.\wator.exe -GridSize=200 -NumFish=2000 -NumShark=500 -Threads=8