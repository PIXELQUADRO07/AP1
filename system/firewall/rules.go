package firewall

import "strings"

// This package provides helpers to generate firewall rules.

type FirewallRule struct {
    Chain string
    Rule  string
}

func NewAcceptRule(interfaceName string) FirewallRule {
    return FirewallRule{
        Chain: "FORWARD",
        Rule:  "-i " + interfaceName + " -j ACCEPT",
    }
}

func NewDropRule(interfaceName string) FirewallRule {
    return FirewallRule{
        Chain: "FORWARD",
        Rule:  "-i " + interfaceName + " -j DROP",
    }
}

func (r FirewallRule) ToIptablesArgs() []string {
    return []string{"-A", r.Chain, strings.Fields(r.Rule)...}
}
