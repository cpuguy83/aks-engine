// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package api

import (
	"path"
	"strconv"
	"testing"

	"github.com/Azure/aks-engine/pkg/api/common"
	"github.com/Azure/go-autorest/autorest/to"
)

func TestKubeletConfigDefaults(t *testing.T) {
	cs := CreateMockContainerService("testcluster", common.RationalizeReleaseAndVersion(Kubernetes, common.KubernetesDefaultRelease, "", false, false), 3, 2, false)
	winProfile := &AgentPoolProfile{}
	winProfile.Count = 1
	winProfile.Name = "agentpool2"
	winProfile.VMSize = "Standard_D2_v2"
	winProfile.OSType = Windows
	cs.Properties.AgentPoolProfiles = append(cs.Properties.AgentPoolProfiles, winProfile)
	cs.Properties.OrchestratorProfile.KubernetesConfig.KubernetesImageBase = "foo.com"
	cs.setKubeletConfig()
	kubeletConfig := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig

	expected := map[string]string{
		"--address":                           "0.0.0.0",
		"--allow-privileged":                  "true",
		"--anonymous-auth":                    "false",
		"--authorization-mode":                "Webhook",
		"--azure-container-registry-config":   "/etc/kubernetes/azure.json",
		"--cadvisor-port":                     "", // Validate that we delete this key for >= 1.12 clusters
		"--cgroups-per-qos":                   "true",
		"--client-ca-file":                    "/etc/kubernetes/certs/ca.crt",
		"--cloud-provider":                    "azure",
		"--cloud-config":                      "/etc/kubernetes/azure.json",
		"--cluster-dns":                       DefaultKubernetesDNSServiceIP,
		"--cluster-domain":                    "cluster.local",
		"--enforce-node-allocatable":          "pods",
		"--event-qps":                         DefaultKubeletEventQPS,
		"--eviction-hard":                     DefaultKubernetesHardEvictionThreshold,
		"--image-gc-high-threshold":           strconv.Itoa(DefaultKubernetesGCHighThreshold),
		"--image-gc-low-threshold":            strconv.Itoa(DefaultKubernetesGCLowThreshold),
		"--image-pull-progress-deadline":      "30m",
		"--keep-terminated-pod-volumes":       "false",
		"--kubeconfig":                        "/var/lib/kubelet/kubeconfig",
		"--max-pods":                          strconv.Itoa(DefaultKubernetesMaxPods),
		"--network-plugin":                    NetworkPluginKubenet,
		"--node-status-update-frequency":      K8sComponentsByVersionMap[cs.Properties.OrchestratorProfile.OrchestratorVersion]["nodestatusfreq"],
		"--non-masquerade-cidr":               DefaultNonMasqueradeCIDR,
		"--pod-manifest-path":                 "/etc/kubernetes/manifests",
		"--pod-infra-container-image":         path.Join(cs.Properties.OrchestratorProfile.KubernetesConfig.KubernetesImageBase, K8sComponentsByVersionMap[cs.Properties.OrchestratorProfile.OrchestratorVersion]["pause"]),
		"--pod-max-pids":                      strconv.Itoa(DefaultKubeletPodMaxPIDs),
		"--protect-kernel-defaults":           "true",
		"--rotate-certificates":               "true",
		"--streaming-connection-idle-timeout": "5m",
		"--feature-gates":                     "PodPriority=true,RotateKubeletServerCertificate=true",
	}
	for key, val := range kubeletConfig {
		if expected[key] != val {
			t.Fatalf("got unexpected kubelet config value for %s: %s, expected %s",
				key, val, expected[key])
		}
	}
	masterKubeletConfig := cs.Properties.MasterProfile.KubernetesConfig.KubeletConfig
	for key, val := range masterKubeletConfig {
		if expected[key] != val {
			t.Fatalf("got unexpected masterProfile kubelet config value for %s: %s, expected %s",
				key, val, expected[key])
		}
	}
	linuxProfileKubeletConfig := cs.Properties.AgentPoolProfiles[0].KubernetesConfig.KubeletConfig
	for key, val := range linuxProfileKubeletConfig {
		if expected[key] != val {
			t.Fatalf("got unexpected Linux agent profile kubelet config value for %s: %s, expected %s",
				key, val, expected[key])
		}
	}
	windowsProfileKubeletConfig := cs.Properties.AgentPoolProfiles[1].KubernetesConfig.KubeletConfig
	expected["--azure-container-registry-config"] = "c:\\k\\azure.json"
	expected["--pod-infra-container-image"] = "kubletwin/pause"
	expected["--kubeconfig"] = "c:\\k\\config"
	expected["--cloud-config"] = "c:\\k\\azure.json"
	expected["--cgroups-per-qos"] = "false"
	expected["--enforce-node-allocatable"] = "\"\"\"\""
	expected["--system-reserved"] = "memory=2Gi"
	expected["--client-ca-file"] = "c:\\k\\ca.crt"
	expected["--hairpin-mode"] = "promiscuous-bridge"
	expected["--image-pull-progress-deadline"] = "20m"
	expected["--resolv-conf"] = "\"\"\"\""
	expected["--eviction-hard"] = "\"\"\"\""
	delete(expected, "--pod-manifest-path")
	delete(expected, "--protect-kernel-defaults")
	for key, val := range windowsProfileKubeletConfig {
		if expected[key] != val {
			t.Fatalf("got unexpected Windows agent profile kubelet config value for %s: %s, expected %s",
				key, val, expected[key])
		}
	}

	cs = CreateMockContainerService("testcluster", "1.8.6", 3, 2, false)
	// TODO test all default overrides
	overrideVal := "/etc/override"
	cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig = map[string]string{
		"--azure-container-registry-config": overrideVal,
	}
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	for key, val := range map[string]string{"--azure-container-registry-config": overrideVal} {
		if k[key] != val {
			t.Fatalf("got unexpected kubelet config value for %s: %s, expected %s",
				key, k[key], val)
		}
	}
}

func TestKubeletConfigUseCloudControllerManager(t *testing.T) {
	// Test UseCloudControllerManager = true
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.UseCloudControllerManager = to.BoolPtr(true)
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--cloud-provider"] != "external" {
		t.Fatalf("got unexpected '--cloud-provider' kubelet config value for UseCloudControllerManager=true: %s",
			k["--cloud-provider"])
	}

	// Test UseCloudControllerManager = false
	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.UseCloudControllerManager = to.BoolPtr(false)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--cloud-provider"] != "azure" {
		t.Fatalf("got unexpected '--cloud-provider' kubelet config value for UseCloudControllerManager=false: %s",
			k["--cloud-provider"])
	}

}

func TestKubeletConfigCloudConfig(t *testing.T) {
	// Test default value and custom value for --cloud-config
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--cloud-config"] != "/etc/kubernetes/azure.json" {
		t.Fatalf("got unexpected '--cloud-config' kubelet config default value: %s",
			k["--cloud-config"])
	}

	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig["--cloud-config"] = "custom.json"
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--cloud-config"] != "custom.json" {
		t.Fatalf("got unexpected '--cloud-config' kubelet config default value: %s",
			k["--cloud-config"])
	}
}

func TestKubeletConfigAzureContainerRegistryCofig(t *testing.T) {
	// Test default value and custom value for --azure-container-registry-config
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--azure-container-registry-config"] != "/etc/kubernetes/azure.json" {
		t.Fatalf("got unexpected '--azure-container-registry-config' kubelet config default value: %s",
			k["--azure-container-registry-config"])
	}

	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig["--azure-container-registry-config"] = "custom.json"
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--azure-container-registry-config"] != "custom.json" {
		t.Fatalf("got unexpected '--azure-container-registry-config' kubelet config default value: %s",
			k["--azure-container-registry-config"])
	}
}

func TestKubeletConfigNetworkPlugin(t *testing.T) {
	// Test NetworkPlugin = "kubenet"
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.NetworkPlugin = NetworkPluginKubenet
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--network-plugin"] != NetworkPluginKubenet {
		t.Fatalf("got unexpected '--network-plugin' kubelet config value for NetworkPlugin=kubenet: %s",
			k["--network-plugin"])
	}

	// Test NetworkPlugin = "azure"
	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.NetworkPlugin = NetworkPluginAzure
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--network-plugin"] != "cni" {
		t.Fatalf("got unexpected '--network-plugin' kubelet config value for NetworkPlugin=azure: %s",
			k["--network-plugin"])
	}

}

func TestKubeletConfigEnableSecureKubelet(t *testing.T) {
	// Test EnableSecureKubelet = true
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.EnableSecureKubelet = to.BoolPtr(true)
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--anonymous-auth"] != "false" {
		t.Fatalf("got unexpected '--anonymous-auth' kubelet config value for EnableSecureKubelet=true: %s",
			k["--anonymous-auth"])
	}
	if k["--authorization-mode"] != "Webhook" {
		t.Fatalf("got unexpected '--authorization-mode' kubelet config value for EnableSecureKubelet=true: %s",
			k["--authorization-mode"])
	}
	if k["--client-ca-file"] != "/etc/kubernetes/certs/ca.crt" {
		t.Fatalf("got unexpected '--client-ca-file' kubelet config value for EnableSecureKubelet=true: %s",
			k["--client-ca-file"])
	}

	// Test EnableSecureKubelet = false
	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.EnableSecureKubelet = to.BoolPtr(false)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	for _, key := range []string{"--anonymous-auth", "--client-ca-file"} {
		if _, ok := k[key]; ok {
			t.Fatalf("got unexpected '%s' kubelet config value for EnableSecureKubelet=false: %s",
				key, k[key])
		}
	}

	// Test default (EnableSecureKubelet = false) for Windows
	cs = CreateMockContainerService("testcluster", "1.10.13", 3, 1, false)
	p := GetK8sDefaultProperties(true)
	cs.Properties = p
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	for _, key := range []string{"--anonymous-auth", "--client-ca-file"} {
		if _, ok := k[key]; ok {
			t.Fatalf("got unexpected '%s' kubelet config value for EnableSecureKubelet=false: %s",
				key, k[key])
		}
	}

	// Test explicit EnableSecureKubelet = false for Windows
	cs = CreateMockContainerService("testcluster", "1.10.13", 3, 1, false)
	p = GetK8sDefaultProperties(true)
	cs.Properties = p
	cs.Properties.OrchestratorProfile.KubernetesConfig.EnableSecureKubelet = to.BoolPtr(false)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	for _, key := range []string{"--anonymous-auth", "--client-ca-file"} {
		if _, ok := k[key]; ok {
			t.Fatalf("got unexpected '%s' kubelet config value for EnableSecureKubelet=false: %s",
				key, k[key])
		}
	}

	// Test EnableSecureKubelet = true for Windows
	cs = CreateMockContainerService("testcluster", "1.10.13", 3, 1, false)
	p = GetK8sDefaultProperties(true)
	cs.Properties = p
	cs.Properties.OrchestratorProfile.KubernetesConfig.EnableSecureKubelet = to.BoolPtr(true)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--anonymous-auth"] != "false" {
		t.Fatalf("got unexpected '--anonymous-auth' kubelet config value for EnableSecureKubelet=true: %s",
			k["--anonymous-auth"])
	}
	if k["--client-ca-file"] != "/etc/kubernetes/certs/ca.crt" {
		t.Fatalf("got unexpected '--client-ca-file' kubelet config value for EnableSecureKubelet=true: %s",
			k["--client-ca-file"])
	}

}

func TestKubeletMaxPods(t *testing.T) {
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.NetworkPlugin = NetworkPluginAzure
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--max-pods"] != strconv.Itoa(DefaultKubernetesMaxPodsVNETIntegrated) {
		t.Fatalf("got unexpected '--max-pods' kubelet config value for NetworkPolicy=%s: %s",
			NetworkPluginAzure, k["--max-pods"])
	}

	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.NetworkPlugin = NetworkPluginKubenet
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--max-pods"] != strconv.Itoa(DefaultKubernetesMaxPods) {
		t.Fatalf("got unexpected '--max-pods' kubelet config value for NetworkPolicy=%s: %s",
			NetworkPluginKubenet, k["--max-pods"])
	}

	// Test that user-overrides for --max-pods work as intended
	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.NetworkPlugin = NetworkPluginKubenet
	cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig["--max-pods"] = "99"
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--max-pods"] != "99" {
		t.Fatalf("got unexpected '--max-pods' kubelet config value for NetworkPolicy=%s: %s",
			NetworkPluginKubenet, k["--max-pods"])
	}

	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.NetworkPlugin = NetworkPluginAzure
	cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig["--max-pods"] = "99"
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--max-pods"] != "99" {
		t.Fatalf("got unexpected '--max-pods' kubelet config value for NetworkPolicy=%s: %s",
			NetworkPluginKubenet, k["--max-pods"])
	}
}

func TestKubeletCalico(t *testing.T) {
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.NetworkPolicy = NetworkPolicyCalico
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--network-plugin"] != "cni" {
		t.Fatalf("got unexpected '--network-plugin' kubelet config value for NetworkPolicy=%s: %s",
			NetworkPolicyCalico, k["--network-plugin"])
	}
}

func TestKubeletHostedMasterIPMasqAgentDisabled(t *testing.T) {
	subnet := "172.16.0.0/16"
	// MasterIPMasqAgent disabled, --non-masquerade-cidr should be subnet
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.HostedMasterProfile = &HostedMasterProfile{
		IPMasqAgent: false,
	}
	cs.Properties.OrchestratorProfile.KubernetesConfig.ClusterSubnet = subnet
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--non-masquerade-cidr"] != subnet {
		t.Fatalf("got unexpected '--non-masquerade-cidr' kubelet config value %s, the expected value is %s",
			k["--non-masquerade-cidr"], subnet)
	}

	// MasterIPMasqAgent enabled, --non-masquerade-cidr should be 0.0.0.0/0
	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.HostedMasterProfile = &HostedMasterProfile{
		IPMasqAgent: true,
	}
	cs.Properties.OrchestratorProfile.KubernetesConfig.ClusterSubnet = subnet
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--non-masquerade-cidr"] != DefaultNonMasqueradeCIDR {
		t.Fatalf("got unexpected '--non-masquerade-cidr' kubelet config value %s, the expected value is %s",
			k["--non-masquerade-cidr"], DefaultNonMasqueradeCIDR)
	}

	// no HostedMasterProfile, --non-masquerade-cidr should be 0.0.0.0/0
	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig.ClusterSubnet = subnet
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--non-masquerade-cidr"] != DefaultNonMasqueradeCIDR {
		t.Fatalf("got unexpected '--non-masquerade-cidr' kubelet config value %s, the expected value is %s",
			k["--non-masquerade-cidr"], DefaultNonMasqueradeCIDR)
	}
}

func TestKubeletIPMasqAgentEnabledOrDisabled(t *testing.T) {
	subnet := "172.16.0.0/16"
	// MasterIPMasqAgent disabled, --non-masquerade-cidr should be subnet
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	b := false
	cs.Properties.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{
		Addons: []KubernetesAddon{
			{
				Name:    IPMASQAgentAddonName,
				Enabled: &b,
			},
		},
	}
	cs.Properties.OrchestratorProfile.KubernetesConfig.ClusterSubnet = subnet
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--non-masquerade-cidr"] != subnet {
		t.Fatalf("got unexpected '--non-masquerade-cidr' kubelet config value %s, the expected value is %s",
			k["--non-masquerade-cidr"], subnet)
	}

	// MasterIPMasqAgent enabled, --non-masquerade-cidr should be 0.0.0.0/0
	cs = CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	b = true
	cs.Properties.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{
		Addons: []KubernetesAddon{
			{
				Name:    IPMASQAgentAddonName,
				Enabled: &b,
			},
		},
	}
	cs.Properties.OrchestratorProfile.KubernetesConfig.ClusterSubnet = subnet
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--non-masquerade-cidr"] != DefaultNonMasqueradeCIDR {
		t.Fatalf("got unexpected '--non-masquerade-cidr' kubelet config value %s, the expected value is %s",
			k["--non-masquerade-cidr"], DefaultNonMasqueradeCIDR)
	}
}

func TestEnforceNodeAllocatable(t *testing.T) {
	// Validate default
	cs := CreateMockContainerService("testcluster", "1.10.13", 3, 2, false)
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--enforce-node-allocatable"] != "pods" {
		t.Fatalf("got unexpected '--enforce-node-allocatable' kubelet config value %s, the expected value is %s",
			k["--enforce-node-allocatable"], "pods")
	}

	// Validate that --enforce-node-allocatable is overridable
	cs = CreateMockContainerService("testcluster", "1.10.13", 3, 2, false)
	cs.Properties.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{
		KubeletConfig: map[string]string{
			"--enforce-node-allocatable": "kube-reserved/system-reserved",
		},
	}
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--enforce-node-allocatable"] != "kube-reserved/system-reserved" {
		t.Fatalf("got unexpected '--enforce-node-allocatable' kubelet config value %s, the expected value is %s",
			k["--enforce-node-allocatable"], "kube-reserved/system-reserved")
	}
}

func TestProtectKernelDefaults(t *testing.T) {
	// Validate default
	cs := CreateMockContainerService("testcluster", "1.12.7", 3, 2, false)
	cs.SetPropertiesDefaults(false, false)
	km := cs.Properties.MasterProfile.KubernetesConfig.KubeletConfig
	if km["--protect-kernel-defaults"] != "true" {
		t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s, the expected value is %s",
			km["--protect-kernel-defaults"], "true")
	}
	ka := cs.Properties.AgentPoolProfiles[0].KubernetesConfig.KubeletConfig
	if ka["--protect-kernel-defaults"] != "true" {
		t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s, the expected value is %s",
			ka["--protect-kernel-defaults"], "true")
	}

	// Validate that --protect-kernel-defaults is "true" by default for relevant distros
	for _, distro := range DistroValues {
		switch distro {
		case AKSUbuntu1604, AKSUbuntu1804:
			cs = CreateMockContainerService("testcluster", "1.10.13", 3, 2, false)
			cs.Properties.MasterProfile.Distro = distro
			cs.Properties.AgentPoolProfiles[0].Distro = distro
			cs.SetPropertiesDefaults(false, false)
			km = cs.Properties.MasterProfile.KubernetesConfig.KubeletConfig
			if km["--protect-kernel-defaults"] != "true" {
				t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s, the expected value is %s",
					km["--protect-kernel-defaults"], "true")
			}
			ka = cs.Properties.AgentPoolProfiles[0].KubernetesConfig.KubeletConfig
			if ka["--protect-kernel-defaults"] != "true" {
				t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s, the expected value is %s",
					ka["--protect-kernel-defaults"], "true")
			}

		// Validate that --protect-kernel-defaults is not enabled for relevant distros
		case Ubuntu, Ubuntu1804, ACC1604, CoreOS:
			cs = CreateMockContainerService("testcluster", "1.10.13", 3, 2, false)
			cs.Properties.MasterProfile.Distro = distro
			cs.Properties.AgentPoolProfiles[0].Distro = distro
			cs.SetPropertiesDefaults(false, false)
			km = cs.Properties.MasterProfile.KubernetesConfig.KubeletConfig
			if _, ok := km["--protect-kernel-defaults"]; ok {
				t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s",
					km["--protect-kernel-defaults"])
			}
			ka = cs.Properties.AgentPoolProfiles[0].KubernetesConfig.KubeletConfig
			if _, ok := ka["--protect-kernel-defaults"]; ok {
				t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s",
					ka["--protect-kernel-defaults"])
			}
		}
	}

	// Validate that --protect-kernel-defaults is not enabled for Windows
	cs = CreateMockContainerService("testcluster", "1.10.13", 3, 2, false)
	cs.Properties.MasterProfile.Distro = AKSUbuntu1604
	cs.Properties.AgentPoolProfiles[0].OSType = Windows
	cs.SetPropertiesDefaults(false, false)
	km = cs.Properties.MasterProfile.KubernetesConfig.KubeletConfig
	if km["--protect-kernel-defaults"] != "true" {
		t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s, the expected value is %s",
			km["--protect-kernel-defaults"], "true")
	}
	ka = cs.Properties.AgentPoolProfiles[0].KubernetesConfig.KubeletConfig
	if _, ok := ka["--protect-kernel-defaults"]; ok {
		t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s",
			ka["--protect-kernel-defaults"])
	}

	// Validate that --protect-kernel-defaults is overridable
	for _, distro := range DistroValues {
		switch distro {
		case Ubuntu, Ubuntu1804, AKSUbuntu1604, AKSUbuntu1804:
			cs = CreateMockContainerService("testcluster", "1.10.13", 3, 2, false)
			cs.Properties.MasterProfile.Distro = "ubuntu"
			cs.Properties.AgentPoolProfiles[0].Distro = "ubuntu"
			cs.Properties.OrchestratorProfile.KubernetesConfig = &KubernetesConfig{
				KubeletConfig: map[string]string{
					"--protect-kernel-defaults": "false",
				},
			}
			cs.SetPropertiesDefaults(false, false)
			km = cs.Properties.MasterProfile.KubernetesConfig.KubeletConfig
			if km["--protect-kernel-defaults"] != "false" {
				t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s, the expected value is %s",
					km["--protect-kernel-defaults"], "false")
			}
			ka = cs.Properties.AgentPoolProfiles[0].KubernetesConfig.KubeletConfig
			if ka["--protect-kernel-defaults"] != "false" {
				t.Fatalf("got unexpected '--protect-kernel-defaults' kubelet config value %s, the expected value is %s",
					ka["--protect-kernel-defaults"], "false")
			}
		}
	}
}

func TestStaticWindowsConfig(t *testing.T) {
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 1, false)
	p := GetK8sDefaultProperties(true)
	p.OrchestratorProfile.OrchestratorVersion = defaultTestClusterVer
	cs.Properties = p
	cs.Properties.OrchestratorProfile.KubernetesConfig.EnableSecureKubelet = to.BoolPtr(true)

	// Start with copy of Linux config
	staticLinuxKubeletConfig := map[string]string{
		"--address":                     "0.0.0.0",
		"--allow-privileged":            "true",
		"--anonymous-auth":              "false",
		"--authorization-mode":          "Webhook",
		"--client-ca-file":              "/etc/kubernetes/certs/ca.crt",
		"--pod-manifest-path":           "/etc/kubernetes/manifests",
		"--cluster-dns":                 cs.Properties.OrchestratorProfile.KubernetesConfig.DNSServiceIP,
		"--cgroups-per-qos":             "true",
		"--kubeconfig":                  "/var/lib/kubelet/kubeconfig",
		"--keep-terminated-pod-volumes": "false",
	}
	expected := make(map[string]string)
	for key, val := range staticLinuxKubeletConfig {
		if key != "--pod-manifest-path" {
			expected[key] = val
		}
	}

	// Add Windows-specific overrides
	// Eventually paths should not be hardcoded here. They should be relative to $global:KubeDir in the PowerShell script
	expected["--azure-container-registry-config"] = "c:\\k\\azure.json"
	expected["--pod-infra-container-image"] = "kubletwin/pause"
	expected["--kubeconfig"] = "c:\\k\\config"
	expected["--cloud-config"] = "c:\\k\\azure.json"
	expected["--cgroups-per-qos"] = "false"
	expected["--enforce-node-allocatable"] = "\"\"\"\""
	expected["--system-reserved"] = "memory=2Gi"
	expected["--client-ca-file"] = "c:\\k\\ca.crt"
	expected["--hairpin-mode"] = "promiscuous-bridge"
	expected["--image-pull-progress-deadline"] = "20m"
	expected["--resolv-conf"] = "\"\"\"\""
	expected["--eviction-hard"] = "\"\"\"\""

	cs.setKubeletConfig()

	for _, profile := range cs.Properties.AgentPoolProfiles {
		if profile.OSType == Windows {
			for key, val := range expected {
				if val != profile.KubernetesConfig.KubeletConfig[key] {
					t.Fatalf("got unexpected '%s' kubelet config value, expected %s, got %s",
						key, val, profile.KubernetesConfig.KubeletConfig[key])
				}
			}
		}
	}
}

func TestKubeletRotateCertificates(t *testing.T) {
	cs := CreateMockContainerService("testcluster", defaultTestClusterVer, 3, 2, false)
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--rotate-certificates"] != "" {
		t.Fatalf("got unexpected '--rotate-certificates' kubelet config value for k8s version %s: %s",
			defaultTestClusterVer, k["--rotate-certificates"])
	}

	// Test 1.11
	cs = CreateMockContainerService("testcluster", common.RationalizeReleaseAndVersion(Kubernetes, "1.11", "", false, false), 3, 2, false)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--rotate-certificates"] != "true" {
		t.Fatalf("got unexpected '--rotate-certificates' kubelet config value for k8s version %s: %s",
			defaultTestClusterVer, k["--rotate-certificates"])
	}

	// Test 1.14
	cs = CreateMockContainerService("testcluster", common.RationalizeReleaseAndVersion(Kubernetes, "1.14", "", false, false), 3, 2, false)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--rotate-certificates"] != "true" {
		t.Fatalf("got unexpected '--rotate-certificates' kubelet config value for k8s version %s: %s",
			defaultTestClusterVer, k["--rotate-certificates"])
	}

	// Test user-override
	cs = CreateMockContainerService("testcluster", common.RationalizeReleaseAndVersion(Kubernetes, "1.14", "", false, false), 3, 2, false)
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	k["--rotate-certificates"] = "false"
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--rotate-certificates"] != "false" {
		t.Fatalf("got unexpected '--rotate-certificates' kubelet config value despite override value %s: %s",
			"false", k["--rotate-certificates"])
	}
}
func TestKubeletConfigDefaultFeatureGates(t *testing.T) {
	// test 1.7
	cs := CreateMockContainerService("testcluster", "1.7.12", 3, 2, false)
	cs.setKubeletConfig()
	k := cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--feature-gates"] != "" {
		t.Fatalf("got unexpected '--feature-gates' kubelet config value for \"--feature-gates\": \"\": %s",
			k["--feature-gates"])
	}

	// test 1.8
	cs = CreateMockContainerService("testcluster", "1.8.15", 3, 2, false)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--feature-gates"] != "PodPriority=true" {
		t.Fatalf("got unexpected '--feature-gates' kubelet config value for \"--feature-gates\": \"\": %s",
			k["--feature-gates"])
	}

	// test 1.11
	cs = CreateMockContainerService("testcluster", common.RationalizeReleaseAndVersion(Kubernetes, "1.11", "", false, false), 3, 2, false)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--feature-gates"] != "PodPriority=true,RotateKubeletServerCertificate=true" {
		t.Fatalf("got unexpected '--feature-gates' kubelet config value for \"--feature-gates\": \"\": %s",
			k["--feature-gates"])
	}

	// test 1.14
	cs = CreateMockContainerService("testcluster", common.RationalizeReleaseAndVersion(Kubernetes, "1.14", "", false, false), 3, 2, false)
	cs.setKubeletConfig()
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	if k["--feature-gates"] != "PodPriority=true,RotateKubeletServerCertificate=true" {
		t.Fatalf("got unexpected '--feature-gates' kubelet config value for \"--feature-gates\": \"\": %s",
			k["--feature-gates"])
	}

	// test user-overrides
	cs = CreateMockContainerService("testcluster", "1.14.1", 3, 2, false)
	k = cs.Properties.OrchestratorProfile.KubernetesConfig.KubeletConfig
	k["--feature-gates"] = "DynamicKubeletConfig=true"
	cs.setKubeletConfig()
	if k["--feature-gates"] != "DynamicKubeletConfig=true,PodPriority=true,RotateKubeletServerCertificate=true" {
		t.Fatalf("got unexpected '--feature-gates' kubelet config value for \"--feature-gates\": \"\": %s",
			k["--feature-gates"])
	}
}
