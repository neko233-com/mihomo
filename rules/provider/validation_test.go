package provider

import (
	"errors"
	"path/filepath"
	"testing"

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
