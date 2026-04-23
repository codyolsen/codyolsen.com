package main

import (
	"fmt"
	"math"
)

const boomThreshold = 75000.0

func power(distPx float64) float64 {
	norm := distPx / 1000.0
	return distPx*0.05 + math.Pow(norm, 15)*100000
}

func main() {
	fmt.Println("dist(px)  power        boom?")
	fmt.Println("--------  -----------  -----")

	distances := []float64{
		10, 20, 30, 50, 75, 100, 150, 200, 300, 400, 500, 750, 1000, 1500,
	}

	for _, dist := range distances {
		p := power(dist)
		boom := ""
		if p > boomThreshold {
			boom = "BOOM"
		}
		fmt.Printf("%8.0f  %11.2f  %s\n", dist, p, boom)
	}
}
