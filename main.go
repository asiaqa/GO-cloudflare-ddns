package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func getIPv4() (string, error) {
	resp, err := http.Get("https://api.ipify.org/")
	if err != nil {
		return "", fmt.Errorf("Unable to get IPv4 address: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Unable to read response body: %v", err)
	}
	return string(body), nil
}

func getIPv6() (string, error) {
	resp, err := http.Get("https://ipv6.duiadns.net/")
	if err != nil {
		return "", fmt.Errorf("Unable to get IPv6 address: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Unable to read response body: %v", err)
	}
	return string(body), nil
}

func writeToLog(result string) {
	f, err := os.OpenFile("ddns-log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

func handleError(errMsg string) {
	fmt.Fprintln(os.Stderr, errMsg)
	writeToLog(errMsg)
}

func getZoneName(ddnsRecordName string) string {
	splitDomain := strings.Split(ddnsRecordName, ".")
	length := len(splitDomain)
	if length == 3 {
		return fmt.Sprintf("%s.%s", splitDomain[1], splitDomain[2])
	} else {
		return fmt.Sprintf("%s.%s.%s", splitDomain[1], splitDomain[2], splitDomain[3])
	}
}

func getZoneID(apiKey, zoneName string) string {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", zoneName)
	respBody := makeRequest("GET", url, apiKey, nil)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(respBody), &resultData)
	zoneID := resultData["result"].([]interface{})[0].(map[string]interface{})["id"].(string)
	if zoneID == "" {
		handleError(fmt.Sprintf("Unable to get zone ID for domain %s", zoneName))
	}
	return zoneID
}

func getRecordID(apiKey, zoneID, ddnsRecordName string) string {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=AAAA&name=%s", zoneID, ddnsRecordName)
	respBody := makeRequest("GET", url, apiKey, nil)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(respBody), &resultData)
	resultArray, ok := resultData["result"].([]interface{})
	if ok && len(resultArray) != 0 {
		recordID := resultArray[0].(map[string]interface{})["id"].(string)
		if recordID == "" {
			handleError("Unable to get record ID")
		}
		return recordID
	}
	return ""
}

func getARecord(apiKey, zoneID, ddnsRecordName string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=A&name=%s", zoneID, ddnsRecordName)
	respBody := makeRequest("GET", url, apiKey, nil)
	var resultData map[string]interface{}
	json.Unmarshal([]byte(respBody), &resultData)
	resultArray, ok := resultData["result"].([]interface{})
	if ok && len(resultArray) != 0 {
		recordContent := resultArray[0].(map[string]interface{})["content"].(string)
		if recordContent == "" {
			return "", fmt.Errorf("Unable to get A record content")
		}
		return recordContent, nil
	}
	return "", fmt.Errorf("A record not found")
}

func updateDNSRecord(apiKey, zoneID, recordID, ddnsRecordName, ipv6Address string) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)
	jsonStr := fmt.Sprintf(`{"type":"AAAA","name":"%s","content":"%s","ttl":120}`, ddnsRecordName, ipv6Address)
	reqBody := []byte(jsonStr)
	makeRequest("PUT", url, apiKey, reqBody)
}

func updateDNSRecordIPv4(apiKey, zoneID, recordID, ddnsRecordName, ipv4Address string) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)
	jsonStr := fmt.Sprintf(`{"type":"A","name":"%s","content":"%s","ttl":120}`, ddnsRecordName, ipv4Address)
	reqBody := []byte(jsonStr)
	makeRequest("PUT", url, apiKey, reqBody)
}

func createDNSRecord(apiKey, zoneID, ddnsRecordName, ipv6Address string) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
	jsonStr := fmt.Sprintf(`{"type":"AAAA","name":"%s","content":"%s","ttl":120}`, ddnsRecordName, ipv6Address)
	reqBody := []byte(jsonStr)
	makeRequest("POST", url, apiKey, reqBody)
}

func createDNSRecordIPv4(apiKey, zoneID, ddnsRecordName, ipv4Address string) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)
	jsonStr := fmt.Sprintf(`{"type":"A","name":"%s","content":"%s","ttl":120}`, ddnsRecordName, ipv4Address)
	reqBody := []byte(jsonStr)
	makeRequest("POST", url, apiKey, reqBody)
}

func makeRequest(method, url, apiKey string, requestBody []byte) []byte {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(requestBody))
	if err != nil {
		handleError(fmt.Sprintf("Failed to create request: %v", err))
		return nil
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		handleError(fmt.Sprintf("Failed to make request: %v", err))
		return nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		handleError(fmt.Sprintf("Failed to read response body: %v", err))
		return nil
	}
	if resp.StatusCode != 200 {
		handleError(fmt.Sprintf("Request failed with status code %d\nResponse Body: %s", resp.StatusCode, respBody))
		return nil
	}
	return respBody
}

var lastIPv6Address string
var lastIPv4Address string

func main() {
	apiKey := flag.String("k", "", "Cloudflare API key")
	ddnsRecordName := flag.String("d", "", "DDNS record name")
	monitorInterval := flag.Int("m", 60, "Monitoring interval in minutes")
	flag.Parse()
	lastIPv6Address = ""
	lastIPv4Address = ""
	for {
		ipv6Address, err := getIPv6()
		if err != nil {
			handleError(fmt.Sprintf("Error getting IPv6 address: %v", err))
		}
		ipv4Address, err := getIPv4()
		if err != nil {
			handleError(fmt.Sprintf("Error getting IPv4 address: %v", err))
		}
		if *apiKey == "" || *ddnsRecordName == "" {
			handleError("API key or DDNS record name not provided")
		}
		if ipv4Address != lastIPv4Address {
			zoneName := getZoneName(*ddnsRecordName)
			zoneID := getZoneID(*apiKey, zoneName)
			recordID := getRecordID(*apiKey, zoneID, *ddnsRecordName)
			if err != nil {
				handleError(fmt.Sprintf("Error retrieving A record: %v", err))
			}
			if recordID != "" {
				recordContent, _ := getARecord(*apiKey, zoneID, *ddnsRecordName)
				if recordContent != ipv4Address {
					updateDNSRecordIPv4(*apiKey, zoneID, recordID, *ddnsRecordName, ipv4Address)
					fmt.Println("IPv4 DNS record updated successfully, ip:", ipv4Address)
				} else {
					fmt.Println("IPv4 DNS record has not updated, ip:", ipv4Address)
				}
			} else {
				createDNSRecordIPv4(*apiKey, zoneID, *ddnsRecordName, ipv4Address)
				fmt.Println("IPv4 DNS record created successfully, ip:", ipv4Address)
			}
			lastIPv4Address = ipv4Address
		}
		if ipv6Address != lastIPv6Address {
			zoneName := getZoneName(*ddnsRecordName)
			zoneID := getZoneID(*apiKey, zoneName)
			recordID := getRecordID(*apiKey, zoneID, *ddnsRecordName)
			if recordID != "" {
				updateDNSRecord(*apiKey, zoneID, recordID, *ddnsRecordName, ipv6Address)
				fmt.Println("IPv6 DNS record updated successfully, ip:", ipv6Address)
			} else {
				createDNSRecord(*apiKey, zoneID, *ddnsRecordName, ipv6Address)
				fmt.Println("IPv6 DNS record created successfully, ip:", ipv6Address)
			}
			lastIPv6Address = ipv6Address
		}
		writeToLog("Current: IPV4: " + lastIPv4Address + " Current: IPV6: " + lastIPv6Address)
		fmt.Println("Check IPv6 and IPv4 again in", time.Duration(*monitorInterval)*time.Minute)
		time.Sleep(time.Duration(*monitorInterval) * time.Minute)
	}
}
