package main

import (
	"bytes"
	"encoding/json"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type nodeInfo struct {
	Name       string `json:"name"`
	ProviderID string `json:"providerid"`
	Hostname   string `json:"hostname"`
	InternalIP string `json:"internalip"`
	CloudName  string `json:"cloud"`
	Zone       string `json:"zone"`
}

func newNodeInfo(node v1.Node) *nodeInfo {
	n := nodeInfo{
		Name:       node.Name,
		ProviderID: node.Spec.ProviderID,
		CloudName:  "unknown",
		Zone:       "unknown",
	}

	if zone, ok := node.GetLabels()["failure-domain.beta.kubernetes.io/zone"]; ok {
		n.Zone = zone
	}

	for _, a := range node.Status.Addresses {
		if a.Type == v1.NodeHostName {
			n.Hostname = a.Address
		} else if a.Type == v1.NodeInternalIP {
			n.InternalIP = a.Address
		}
	}

	n.parseProviderID()
	return &n
}

func (n *nodeInfo) parseProviderID() {
	if strings.HasPrefix(n.ProviderID, "aws:///") {
		n.CloudName = "AWS"
		whole := n.ProviderID[len("aws:///"):]
		parts := strings.Split(whole, "/")
		if len(parts) > 0 && n.Zone == "unknown" {
			n.Zone = parts[0]
		}
		return
	}
	if strings.HasPrefix(n.ProviderID, "gce://") {
		n.CloudName = "GCE"
		whole := n.ProviderID[len("gce://"):]
		parts := strings.Split(whole, "/")
		if len(parts) > 1 && n.Zone == "unknown" {
			n.Zone = parts[1]
		}
		return
	}
	if strings.HasPrefix(n.ProviderID, "azure:///") {
		n.CloudName = "Azure"
		whole := n.ProviderID[len("azure:///"):]
		parts := strings.Split(whole, "/")
		if len(parts) > 3 && n.Zone == "unknown" {
			// the azure providerID doesn't seem to contain the availability
			// zone - we'll use the resource group instead
			n.Zone = parts[3]
		}
		return
	}
	if strings.HasPrefix(n.ProviderID, "openstack:///") {
		n.CloudName = "OpenStack"
		return
	}
	if i := strings.Index(n.ProviderID, "://"); i != -1 {
		n.CloudName = n.ProviderID[:i]
	}
}

func (n nodeInfo) String() string {
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(&n)
	return b.String()
}
