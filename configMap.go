package main

import "strings"

// This function help convert the k8s configmap to a Golang map object (map[string]string[])
func parseConfigMapData(data string) map[string]string {
	parsedData := make(map[string]string)

	// Split the data by newline and create key-value pairs
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			parsedData[key] = value
		}
	}
	return parsedData
}