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
func main() {
	api_key := flag.String("k", "", "Cloudflare API key")
	ddns_record_name := flag.String("d", "", "DDNS record name")
	flag.Parse()
	ipv6_address, err := getIP()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *api_key == "" || *ddns_record_name == "" {
		fmt.Fprintln(os.Stderr, "API key or ddns record name not provided")
		os.Exit(1)
	}
	zone_name := fmt.Sprintf("%s.%s", strings.Split(*ddns_record_name, ".")[1], strings.Split(*ddns_record_name, ".")[2])
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", zone_name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *api_key))
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to make request: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		fmt.Fprintln(os.Stderr, "Unable to get zone ID for domain", zone_name)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)
	zone_id := result["result"].([]interface{})[0].(map[string]interface{})["id"]
	zone_idStr, ok := zone_id.(string)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unable to get zone ID\n")
		os.Exit(1)
	}
	url = fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=AAAA&name=%s", zone_idStr, *ddns_record_name)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *api_key))
	req.Header.Add("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to make request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ = ioutil.ReadAll(resp.Body)
	json.Unmarshal([]byte(body), &result)
	resultArray, ok := result["result"].([]interface{})
	if !ok || len(resultArray) == 0 {
		// No record found, create a new record
		url = fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zone_idStr)
		jsonStr := fmt.Sprintf(`{"type":"AAAA","name":"%s","content":"%s","ttl":120}`, *ddns_record_name, ipv6_address)
		reqBody := []byte(jsonStr)
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *api_key))
		req.Header.Add("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to make request: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			fmt.Fprintf(os.Stderr, "Failed to create DNS record: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("DNS record created successfully")
	} else {
		// Record found, update the existing record with the new IPv6 address
		record_id := resultArray[0].(map[string]interface{})["id"]
		record_idStr, ok := record_id.(string)
		if !ok {
			fmt.Fprintf(os.Stderr, "Unable to get record ID\n")
			os.Exit(1)
		}
		url = fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zone_idStr, record_idStr)
		jsonStr := fmt.Sprintf(`{"type":"AAAA","name":"%s","content":"%s","ttl":120}`, *ddns_record_name, ipv6_address)
		reqBody := []byte(jsonStr)
		req, err = http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create request: %v\n", err)
			os.Exit(1)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *api_key))
		req.Header.Add("Content-Type", "application/json")
		resp, err = client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to make request: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			fmt.Fprintf(os.Stderr, "Failed to update DNS record: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("DNS record updated successfully")
	}
}
