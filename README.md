# Cloudflare DDNS Updater 
 
This program updates a Cloudflare DNS entry with the current *IPv6* address of the machine it is running on. It is useful for keeping DNS records up-to-date with dynamic IP addresses. 
 
## Getting Started 
 
### Prerequisites 
 
- Go programming language (to build and run the program) 
- A Cloudflare account with a domain and corresponding API key 
 
### Installation 
 
1. Clone the repository to your local machine:
git clone https://github.com/yourusername/go-cf-ddns.git
2. Build the binary file:
cd go-cf-ddns
   go build -o go-cf-ddns main.go
## Usage 
 
To run the program, use the following command:
```
./go-cf-ddns -k "YOUR_API_KEY" -d "DDNS_RECORD_NAME"
```
Replace  YOUR_API_KEY  with your Cloudflare API key and  DDNS_RECORD_NAME  with the desired DNS record name (e.g.,  home.example.com ). 
 
## Example
```
./go-cf-ddns -k "1234567890abcdefghijklm" -d "home.example.com"
```
This command will update the DNS record  home.example.com  with the current IPv6 address of the machine it is running on. 
 
## Contributing 
 
Feel free to open an issue or submit a pull request if you find any issue, bug, or have a feature request for this project. 
 
## License 
 
This project is open-source and available under the MIT License. See [LICENSE](LICENSE) for more details.
