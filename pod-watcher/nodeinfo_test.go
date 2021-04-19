package main

import "testing"

func TestParseProviderID(t *testing.T) {
	tables := []struct {
		providerID string
		cloudName  string
		zone       string
	}{
		{"gce://striped-torus-760/us-central1-b/kubernetes-node-861h", "GCE", "us-central1-b"}, // taken from https://kubernetes.io/docs/tasks/debug-application-cluster/debug-application-introspection/
		{"aws:///ap-southeast-1a/i-06fbbd699deb4abcd", "AWS", "ap-southeast-1a"},
		{"azure:///subscriptions/subscriptionId/resourceGroups/kubernetes/providers/Microsoft.Compute/virtualMachines/kubernetes-master", "Azure", "kubernetes"}, // taken from https://sourcegraph.com/github.com/kubernetes/autoscaler@902d2414b7f58f58cdb6218bb1b60b3c75ef7283/-/blob/cluster-autoscaler/cloudprovider/azure/azure_cloud_provider_test.go#L184
	}

	for _, table := range tables {
		n := nodeInfo{
			ProviderID: table.providerID,
			Zone:       "unknown",
		}
		n.parseProviderID()
		if n.CloudName != table.cloudName {
			t.Errorf("expected cloudName of %s from providerID %s - got %s instead", table.cloudName, n.ProviderID, n.CloudName)
		}
		if n.Zone != table.zone {
			t.Errorf("expected zone of %s from providerID %s - got %s instead", table.zone, n.ProviderID, n.Zone)
		}
	}
}
