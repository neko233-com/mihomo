package route

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestConcurrentRecreateAndShutdown(t *testing.T) {
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reservation.Addr().String()
	_ = reservation.Close()
	uiDir := t.TempDir()
	t.Cleanup(func() { ReCreateServer(&Config{}); SetEmbedMode(false) })
	ReCreateServer(&Config{Addr: address, Secret: "test-secret"})
	client := &http.Client{Timeout: time.Second}
	req, _ := http.NewRequest(http.MethodGet, "http://"+address+"/version", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("status %d", response.StatusCode)
	}
	SetUIPath(uiDir)
	SetEmbedMode(true)
	ReCreateServer(&Config{Addr: address, Secret: "test-secret"})
	uiResponse, err := client.Get("http://" + address + "/ui/")
	if err != nil {
		t.Fatal(err)
	}
	_ = uiResponse.Body.Close()
	if uiResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("embedded server exposed external UI: %d", uiResponse.StatusCode)
	}
	upgradeReq, _ := http.NewRequest(http.MethodPost, "http://"+address+"/upgrade/ui", nil)
	upgradeReq.Header.Set("Authorization", "Bearer test-secret")
	upgradeResponse, err := client.Do(upgradeReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = upgradeResponse.Body.Close()
	if upgradeResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("embedded server exposed UI updater: %d", upgradeResponse.StatusCode)
	}
	var wg sync.WaitGroup
	for worker := 0; worker < 3; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				if worker == 0 {
					ReCreateServer(&Config{Addr: address, Secret: "test-secret"})
				} else if worker == 1 {
					ReCreateServer(&Config{})
				} else {
					SetEmbedMode(i%2 == 0)
					SetUIPath(uiDir)
				}
			}
		}(worker)
	}
	wg.Wait()
	ReCreateServer(&Config{})
	connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
	if err == nil {
		_ = connection.Close()
		t.Fatal(fmt.Sprintf("controller %s remained open after shutdown", address))
	}
}
