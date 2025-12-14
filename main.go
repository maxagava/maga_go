package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL  = "http://srv.msk01.gigacorp.local/_stats"
	maxErrors  = 3
	pollPeriod = 5 * time.Second // период опроса
)

func main() {
	ticker := time.NewTicker(pollPeriod)
	defer ticker.Stop()

	errorCount := 0

	for range ticker.C {
		stats, err := fetchStats()
		if err != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
				errorCount = 0
			}
			continue
		}

		errorCount = 0
		checkThresholds(stats)
	}
}

// fetchStats отправляет HTTP‑запрос и парсит 7 чисел из ответа.
func fetchStats() ([7]float64, error) {
	var empty [7]float64

	resp, err := http.Get(serverURL)
	if err != nil {
		return empty, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return empty, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, err
	}

	fields := strings.Split(strings.TrimSpace(string(body)), ",")
	if len(fields) != 7 {
		return empty, fmt.Errorf("invalid data format")
	}

	var stats [7]float64
	for i, f := range fields {
		value, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return empty, fmt.Errorf("parse error: %w", err)
		}
		stats[i] = value
	}

	return stats, nil
}

// checkThresholds проверяет пороги и печатает сообщения.
func checkThresholds(stats [7]float64) {
	loadAvg := stats[0]
	totalMem := stats[1]
	usedMem := stats[2]
	totalDisk := stats[3]
	usedDisk := stats[4]
	totalNet := stats[5]
	usedNet := stats[6]

	// 1. Load Average
	if loadAvg > 30 {
		fmt.Printf("Load Average is too high: %.0f\n", loadAvg)
	}

	// 2. Память: > 80%
	if totalMem > 0 {
		memUsage := (usedMem / totalMem) * 100
		if memUsage > 80 {
			fmt.Printf("Memory usage too high: %.0f%%\n", memUsage)
		}
	}

	// 3. Диск: занято > 90%, вывести свободное место в МБ
	if totalDisk > 0 {
		usedRatio := usedDisk / totalDisk
		freeDiskMb := (totalDisk - usedDisk) / 1024 / 1024
		if usedRatio > 0.9 {
			fmt.Printf("Free disk space is too low: %.0f Mb left\n", freeDiskMb)
		}
	}

	// 4. Сеть: занято > 90%, вывести свободную полосу в Мбит/с
	if totalNet > 0 {
		usedRatio := usedNet / totalNet
		freeNetMbit := (totalNet - usedNet) * 8 / 1024 / 1024
		if usedRatio > 0.9 {
			fmt.Printf("Network bandwidth usage high: %.0f Mbit/s available\n", freeNetMbit)
		}
	}
}
