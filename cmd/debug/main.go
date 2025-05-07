package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const chain_name = "lava"

func main() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	urls := []string{
		fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/%s/chain.json", chain_name),
		fmt.Sprintf("https://raw.githubusercontent.com/cosmos/chain-registry/master/testnets/%s/chain.json", chain_name),
	}

	for _, url := range urls {
		fmt.Printf("\nTesting URL: %s\n", url)

		resp, err := client.Head(url)
		if err != nil {
			fmt.Printf("HEAD request error: %v\n", err)
			continue
		}
		fmt.Printf("HEAD Status: %d\n", resp.StatusCode)
		resp.Body.Close()

		resp, err = client.Get(url)
		if err != nil {
			fmt.Printf("GET request error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		fmt.Printf("GET Status: %d\n", resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Read body error: %v\n", err)
			continue
		}
		fmt.Printf("Response body length: %d bytes\n", len(body))
		if len(body) > 100 {
			fmt.Printf("First 100 bytes: %s\n", body[:100])
		}
	}
}
