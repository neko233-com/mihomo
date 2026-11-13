package geodata_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/metacubex/mihomo/component/geodata"
	_ "github.com/metacubex/mihomo/component/geodata/standard"
	C "github.com/metacubex/mihomo/constant"
)

func TestOfflineValidationDoesNotDownloadOrRepair(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); _, _ = w.Write([]byte("remote")) }))
	defer server.Close()
	oldHome := C.Path.HomeDir()
	dir := t.TempDir()
	C.SetHomeDir(dir)
	defer C.SetHomeDir(oldHome)
	geodata.SetLoader("standard")
	geodata.SetGeoSiteUrl(server.URL)
	geodata.SetGeoIpUrl(server.URL)
	geodata.SetMmdbUrl(server.URL)
	geodata.SetASNUrl(server.URL)
	geodata.SetOfflineValidation(true)
	defer geodata.SetOfflineValidation(false)
	for _, tc := range []struct {
		name string
		path string
		run  func() error
	}{
		{"site", C.Path.GeoSite(), geodata.InitGeoSite},
		{"mmdb", C.Path.MMDB(), geodata.InitGeoIP},
		{"asn", C.Path.ASN(), geodata.InitASN},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err == nil {
				t.Fatal("missing database accepted")
			}
			if _, err := os.Stat(tc.path); !os.IsNotExist(err) {
				t.Fatalf("missing database changed: %v", err)
			}
			if err := os.WriteFile(tc.path, []byte("invalid"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := tc.run(); err == nil {
				t.Fatal("invalid database accepted")
			}
			data, err := os.ReadFile(tc.path)
			if err != nil || string(data) != "invalid" {
				t.Fatalf("invalid database changed: %q %v", data, err)
			}
		})
	}
	geodata.SetGeodataMode(true)
	defer geodata.SetGeodataMode(false)
	geoPath := filepath.Join(dir, C.GeoipName)
	if err := os.WriteFile(geoPath, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := geodata.InitGeoIP(); err == nil {
		t.Fatal("invalid GeoIP accepted")
	}
	if data, err := os.ReadFile(geoPath); err != nil || string(data) != "invalid" {
		t.Fatal("invalid GeoIP changed")
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("made %d requests", got)
	}
}
