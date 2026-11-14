package provider

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/metacubex/mihomo/common/utils"
	"github.com/metacubex/mihomo/component/resource"
	P "github.com/metacubex/mihomo/constant/provider"
)

func TestValidateBytesRejectsMissingYAMLRulePayload(t *testing.T) {
	p := NewRuleSetProvider("test", P.Domain, P.YamlRule, 0, resource.NewFileVehicle(filepath.Join(t.TempDir(), "rules")), nil, nil, nil).(*RuleSetProvider)
	defer p.Close()
	for _, valid := range []string{"payload:\n  - '+.example.test'\n", "payload: []\n"} {
		if err := p.ValidateBytes([]byte(valid)); err != nil {
			t.Fatalf("valid payload rejected: %v", err)
		}
	}
	if err := p.ValidateBytes([]byte("unrelated: true\n")); !errors.Is(err, ErrNoPayload) {
		t.Fatalf("missing payload: %v", err)
	}
	if err := p.ValidateBytes([]byte("payload: [broken\n")); err == nil {
		t.Fatal("malformed YAML accepted")
	}
}

func TestYAMLRuleFormatsAreParsedInsteadOfSilentlyEmpty(t *testing.T) {
	for _, tc := range []struct {
		source string
		count  int
	}{
		{"payload: ['+.a.test', '+.b.test']", 2},
		{"payload: [\n  '+.a.test',\n  '+.b.test'\n]\n", 2},
		{"payload:\n  - '+.a.test'\n  - '+.b.test'\n", 2},
		{"payload: []\n", 0},
	} {
		strategy, err := rulesParse([]byte(tc.source), newStrategy(P.Domain, nil), P.YamlRule)
		if err != nil || strategy.Count() != tc.count {
			t.Fatalf("%q: count=%v err=%v", tc.source, strategy, err)
		}
	}
	for _, invalid := range []string{"unrelated: true\n", "payload: [broken\n"} {
		if _, err := rulesParse([]byte(invalid), newStrategy(P.Domain, nil), P.YamlRule); err == nil {
			t.Fatalf("invalid YAML accepted: %q", invalid)
		}
	}
}

type validationTunnel struct {
	cb utils.Callback[P.RuleProvider]
}

func (*validationTunnel) Providers() map[string]P.ProxyProvider                 { return nil }
func (*validationTunnel) RuleProviders() map[string]P.RuleProvider              { return nil }
func (t *validationTunnel) RuleUpdateCallback() *utils.Callback[P.RuleProvider] { return &t.cb }

func TestConcurrentRuleRefreshAndInspection(t *testing.T) {
	oldTunnel := tunnel
	tunnel = &validationTunnel{}
	defer func() { tunnel = oldTunnel }()
	p := NewRuleSetProvider("test", P.Domain, P.YamlRule, 0, resource.NewFileVehicle(filepath.Join(t.TempDir(), "rules")), nil, nil, nil).(*RuleSetProvider)
	defer p.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 80; i++ {
			data := []byte("payload: ['+.a.test']\n")
			if i%2 == 0 {
				data = []byte("payload: ['+.b.test', '+.c.test']\n")
			}
			if _, _, err := p.Fetcher.SideUpdate(data); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 300; i++ {
		_ = p.Count()
		_ = p.Strategy()
		if _, err := json.Marshal(p); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
}
