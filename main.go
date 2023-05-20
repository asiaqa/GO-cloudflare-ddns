package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

func getIP() (string, error) {
	resp, err := http.Get("https://api6.ipify.org")
	if err != nil {
		return "", fmt.Errorf("Unable to get IP address: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Unable to read response body: %v", err)
	}
	return string(body), nil
}
func writeToLog(result string) {
	f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		return
	}
	defer f.Close()
	t := time.Now()
	logLine := fmt.Sprintf("%s - %s\n", t.Format("2006-01-02 15:04:05"), result)
	if _, err := f.WriteString(logLine); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing log file: %v\n", err)
		return
	}
}
func main() {
	api_key := flag.String("k", "", "Cloudflare API key")
	ddns_record_name := flag.String("d", "", "DDNS record name")
	flag.Parse()
	ipv6_address, err := getIP()
	if err != nil {
		result := fmt.Sprintf("Error getting IP address: %v", err)
		fmt.Fprintln(os.Stderr, result)
		writeToLog(result)
		os.Exit(1)
	}
	if *api_key == "" || *ddns_record_name == "" {
		result := "API key or ddns record name not provided"
		fmt.Fprintln(os.Stderr, result)
		writeToLog(result)
		os.Exit(1)
	}
	zone_name := fmt.Sprintf("%s.%s", strings.Split(*ddns_record_name, ".")[1], strings.Split(*ddns_record_name, ".")[2])
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", zone_name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result := fmt.Sprintf("Failed to create request: %v\n", err)
		fmt.Fprintf(os.Stderr, result)
		writeToLog(result)
		os.Exit(1)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *api_key))
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		result := fmt.Sprintf("Failed to make request: %v\n", err)
		fmt.Fprintf(os.Stderr, result)
		writeToLog(result)
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		result := fmt.Sprintf("Unable to get zone ID for domain %s", zone_name)
		fmt.Fprintln(os.Stderr, result)
		writeToLog(result)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(body), &resultData)
	zone_id := resultData["result"].([]interface{})[0].(map[string]interface{})["id"]
	zone_idStr, ok := zone_id.(string)
	if !ok {
		result := "Unable to get zone ID"
		fmt.Fprintf(os.Stderr, result)
		writeToLog(result)
		os.Exit(1)
	}
	url = fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=AAAA&name=%s", zone_idStr, *ddns_record_name)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		result := fmt.Sprintf("Failed to create request: %v\n", err)
		fmt.Fprintf(os.Stderr, result)
		writeToLog(result)
		os.Exit(1)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *api_key))
	req.Header.Add("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		result := fmt.Sprintf("Failed to make request: %v\n", err)
		fmt.Fprintf(os.Stderr, result)
		writeToLog(result)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ = ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(body), &resultData)
	resultArray, ok := resultData["result"].([]interface{})
	if !ok || len(resultArray) == 0 {
		// No record found, create a new record
		url = fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zone_idStr)
		jsonStr := fmt.Sprintf(`{"type":"AAAA","name":"%s","content":"%s","ttl":120}`, *ddns_record_name, ipv6_address)
		reqBody := []byte(jsonStr)
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			result := fmt.Sprintf("Failed to create request: %v\n", err)
			fmt.Fprintf(os.Stderr, result)
			writeToLog(result)
			os.Exit(1)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *api_key))
		req.Header.Add("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			result := fmt.Sprintf("Failed to make request: %v\n", err)
			fmt.Fprintf(os.Stderr, result)
			writeToLog(result)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			result := "Failed to create DNS record"
			fmt.Fprintf(os.Stderr, result)
			writeToLog(result)
			os.Exit(1)
		}
		result := "DNS record created successfully"
		fmt.Println(result)
		writeToLog(result)
	} else {
		// Record found, update the existing record with the new IPv6 address
		record_id := resultArray[0].(map[string]interface{})["id"]
		record_idStr, ok := record_id.(string)
		if !ok {
			result := "Unable to get record ID"
			fmt.Fprintf(os.Stderr, result)
			writeToLog(result)
			os.Exit(1)
		}
		url = fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone_idStr, record_idStr)
		jsonStr := fmt.Sprintf(`{"type":"AAAA","name":"%s","content":"%s","ttl":120}`, *ddns_record_name, ipv6_address)
		reqBody := []byte(jsonStr)
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
		if err != nil {
			result := fmt.Sprintf("Failed to create request: %v\n", err)
			fmt.Fprintf(os.Stderr, result)
			writeToLog(result)
			os.Exit(1)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *api_key))
		req.Header.Add("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			result := fmt.Sprintf("Failed to make request: %v\n", err)
			fmt.Fprintf(os.Stderr, result)
			writeToLog(result)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			result := "Failed to update DNS record"
			fmt.Fprintf(os.Stderr, result)
			writeToLog(result)
			os.Exit(1)
		}
		result := "DNS record updated successfully"
		fmt.Println(result)
		writeToLog(result)
	}
}
