// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package firewalld

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type Zone struct {
	XMLName            xml.Name      `xml:"zone"`
	Target             string        `xml:"target,attr"`
	Masquerade         *struct{}     `xml:"masquerade"`
	IcmpBlockInversion *struct{}     `xml:"icmp-block-inversion"`
	Interfaces         []Interface   `xml:"interface"`
	Sources            []Source      `xml:"source"`
	Services           []Name        `xml:"service"`
	Ports              []Port        `xml:"port"`
	Protocols          []Protocol    `xml:"protocol"`
	ForwardPorts       []ForwardPort `xml:"forward-port"`
	SourcePorts        []Port        `xml:"source-port"`
	IcmpBlocks         []Name        `xml:"icmp-block"`
	Rules              []Rule        `xml:"rule"`
	Raw                string        `xml:",innerxml"`
}

type Interface struct {
	Name string `xml:"name,attr"`
}

type Source struct {
	Address string `xml:"address,attr"`
}

type Name struct {
	Name string `xml:"name,attr"`
}

type Port struct {
	Port     string `xml:"port,attr"`
	Protocol string `xml:"protocol,attr"`
}

type Protocol struct {
	Value string `xml:"value,attr"`
}

type ForwardPort struct {
	Port     string `xml:"port,attr"`
	Protocol string `xml:"protocol,attr"`
	ToPort   string `xml:"to-port,attr"`
	ToAddr   string `xml:"to-addr,attr"`
}

type Rule struct {
	Family   string               `xml:"family,attr"`
	Priority string               `xml:"priority,attr"`
	Source   *RuleEndpoint        `xml:"source"`
	Dest     *RuleEndpoint        `xml:"destination"`
	Service  *Name                `xml:"service"`
	Port     *Port                `xml:"port"`
	Log      *firewalldRuleLog    `xml:"log"`
	Accept   *struct{}            `xml:"accept"`
	Reject   *firewalldRuleReject `xml:"reject"`
	Drop     *struct{}            `xml:"drop"`
	Masq     *struct{}            `xml:"masquerade"`
	Mark     *firewalldRuleMark   `xml:"mark"`
	InnerXML string               `xml:",innerxml"`
}

func (rule *Rule) ToTokens() []string {
	tokens := []string{"rule"}

	if v := strings.TrimSpace(rule.Family); v != "" {
		tokens = append(tokens, fmt.Sprintf("family=\"%s\"", v))
	}
	if v := strings.TrimSpace(rule.Priority); v != "" {
		tokens = append(tokens, fmt.Sprintf("priority=\"%s\"", v))
	}

	if rule.Source != nil {
		if srcTokens := rule.Source.ToTokens(); len(srcTokens) > 0 {
			tokens = append(tokens, "source")
			tokens = append(tokens, srcTokens...)
		}
	}
	if rule.Dest != nil {
		if dstTokens := rule.Dest.ToTokens(); len(dstTokens) > 0 {
			tokens = append(tokens, "destination")
			tokens = append(tokens, dstTokens...)
		}
	}
	if rule.Service != nil {
		if v := strings.TrimSpace(rule.Service.Name); v != "" {
			tokens = append(tokens, "service", fmt.Sprintf("name=\"%s\"", v))
		}
	}
	if rule.Port != nil {
		portTokens := []string{}
		if v := strings.TrimSpace(rule.Port.Port); v != "" {
			portTokens = append(portTokens, fmt.Sprintf("port=\"%s\"", v))
		}
		if v := strings.TrimSpace(rule.Port.Protocol); v != "" {
			portTokens = append(portTokens, fmt.Sprintf("protocol=\"%s\"", v))
		}
		if len(portTokens) > 0 {
			tokens = append(tokens, "port")
			tokens = append(tokens, portTokens...)
		}
	}
	if rule.Log != nil {
		logTokens := []string{}
		if v := strings.TrimSpace(rule.Log.Prefix); v != "" {
			logTokens = append(logTokens, fmt.Sprintf("prefix=\"%s\"", v))
		}
		if v := strings.TrimSpace(rule.Log.Level); v != "" {
			logTokens = append(logTokens, fmt.Sprintf("level=\"%s\"", v))
		}
		if len(logTokens) > 0 {
			tokens = append(tokens, "log")
			tokens = append(tokens, logTokens...)
		}
	}

	switch {
	case rule.Accept != nil:
		tokens = append(tokens, "accept")
	case rule.Drop != nil:
		tokens = append(tokens, "drop")
	case rule.Reject != nil:
		tokens = append(tokens, "reject")
		if v := strings.TrimSpace(rule.Reject.Type); v != "" {
			tokens = append(tokens, fmt.Sprintf("type=\"%s\"", v))
		}
	case rule.Masq != nil:
		tokens = append(tokens, "masquerade")
	case rule.Mark != nil:
		tokens = append(tokens, "mark")
		if v := strings.TrimSpace(rule.Mark.Set); v != "" {
			tokens = append(tokens, fmt.Sprintf("set=\"%s\"", v))
		}
	default:
		return nil
	}

	return tokens
}

type RuleEndpoint struct {
	Address string `xml:"address,attr"`
	Ipset   string `xml:"ipset,attr"`
	Mac     string `xml:"mac,attr"`
	Invert  string `xml:"invert,attr"`
}

func (e *RuleEndpoint) ToTokens() []string {
	if e == nil {
		return nil
	}
	tokens := []string{}
	if ParseBool(e.Invert) {
		tokens = append(tokens, "not")
	}
	if v := strings.TrimSpace(e.Address); v != "" {
		tokens = append(tokens, fmt.Sprintf("address=\"%s\"", v))
	}
	if v := strings.TrimSpace(e.Ipset); v != "" {
		tokens = append(tokens, fmt.Sprintf("ipset=\"%s\"", v))
	}
	if v := strings.TrimSpace(e.Mac); v != "" {
		tokens = append(tokens, fmt.Sprintf("mac=\"%s\"", v))
	}
	return tokens
}

type firewalldRuleLog struct {
	Prefix string `xml:"prefix,attr"`
	Level  string `xml:"level,attr"`
}

type firewalldRuleReject struct {
	Type string `xml:"type,attr"`
}

type firewalldRuleMark struct {
	Set string `xml:"set,attr"`
}
