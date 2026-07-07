/*
 * Teleport
 * Copyright (C) 2023  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package kubeconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestLocalProxy(t *testing.T) {
	const (
		proxyHost           = "root-cluster.example.com"
		proxy               = proxyHost + ":443"
		rootKubeClusterAddr = "https://root-cluster.example.com"
		rootClusterName     = "root-cluster"
		leafClusterName     = "leaf-cluster"
	)

	kubeconfigPath, initialConfig := setup(t)
	creds, _, err := genUserKeyRing("localhost")
	require.NoError(t, err)
	exec := &ExecValues{
		TshBinaryPath: "/path/to/tsh",
	}

	// Simulate `tsh kube login`.
	require.NoError(t, Update(kubeconfigPath, Values{
		TeleportClusterName: rootClusterName,
		ClusterAddr:         rootKubeClusterAddr,
		KubeClusters:        []string{"kube1"},
		Credentials:         creds,
		Exec:                exec,
		ProxyAddr:           proxy,
	}, false))
	require.NoError(t, Update(kubeconfigPath, Values{
		TeleportClusterName: rootClusterName,
		ClusterAddr:         rootKubeClusterAddr,
		KubeClusters:        []string{"kube2"},
		Credentials:         creds,
		Exec:                exec,
		ProxyAddr:           proxy,
		SelectCluster:       "kube2",
	}, false))
	require.NoError(t, Update(kubeconfigPath, Values{
		TeleportClusterName: leafClusterName,
		ClusterAddr:         rootKubeClusterAddr,
		KubeClusters:        []string{"kube3"},
		Credentials:         creds,
		Namespace:           "namespace",
		Impersonate:         "as",
		ImpersonateGroups:   []string{"group1", "group2"},
		Exec:                exec,
		ProxyAddr:           proxy,
	}, false))

	configAfterLogins, err := Load(kubeconfigPath)
	require.NoError(t, err)

	t.Run("LocalProxyClustersFromDefaultConfig", func(t *testing.T) {
		clusters := LocalProxyClustersFromDefaultConfig(configAfterLogins, proxyHost, rootKubeClusterAddr)
		require.ElementsMatch(t, LocalProxyClusters{
			{
				TeleportCluster: rootClusterName,
				KubeCluster:     "kube1",
			},
			{
				TeleportCluster: rootClusterName,
				KubeCluster:     "kube2",
			},
			{
				TeleportCluster:   leafClusterName,
				KubeCluster:       "kube3",
				Namespace:         "namespace",
				Impersonate:       "as",
				ImpersonateGroups: []string{"group1", "group2"},
			},
		}, clusters)
	})

	t.Run("FindTeleportClusterForLocalProxy", func(t *testing.T) {
		inputConfig := configAfterLogins.DeepCopy()

		// Simulate a scenario that kube3 is already pointing to a local proxy
		// through ProxyURL.
		inputConfig.Clusters[leafClusterName+"-kube3"].ProxyURL = "https://localhost:8443"

		tests := []struct {
			name          string
			selectContext string
			checkResult   require.BoolAssertionFunc
			wantCluster   LocalProxyCluster
		}{
			{
				name:          "not Teleport cluster",
				selectContext: "dev",
				checkResult:   require.False,
			},
			{
				name:          "context not found",
				selectContext: "not-found",
				checkResult:   require.False,
			},
			{
				name:          "find Teleport cluster by context name",
				selectContext: rootClusterName + "-kube1",
				checkResult:   require.True,
				wantCluster: LocalProxyCluster{
					TeleportCluster: rootClusterName,
					KubeCluster:     "kube1",
				},
			},
			{
				name:          "find Teleport cluster by current context",
				selectContext: "",
				checkResult:   require.True,
				wantCluster: LocalProxyCluster{
					TeleportCluster: rootClusterName,
					KubeCluster:     "kube2",
				},
			},
			{
				name:          "skip local proxy config",
				selectContext: leafClusterName + "-kube3",
				checkResult:   require.False,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				cluster, found := FindTeleportClusterForLocalProxy(inputConfig, test.selectContext, proxyHost, rootKubeClusterAddr)
				test.checkResult(t, found)
				require.Equal(t, test.wantCluster, cluster)
			})
		}
	})

	t.Run("CreateLocalProxyConfig", func(t *testing.T) {
		caData := []byte("CAData")
		clientKeyData := []byte("clientKeyData")
		values := &LocalProxyValues{
			LocalProxyCAs:           map[string][]byte{rootClusterName: caData},
			TeleportProfileName:     proxyHost,
			TeleportKubeClusterAddr: rootKubeClusterAddr,
			LocalProxyURL:           "http://localhost:12345",
			ClientKeyData:           clientKeyData,
			Clusters: LocalProxyClusters{{
				TeleportCluster:   rootClusterName,
				KubeCluster:       "kube1",
				Namespace:         "namespace",
				Impersonate:       "as",
				ImpersonateGroups: []string{"group1", "group2"},
			}},
		}

		newConfig, err := CreateLocalProxyConfig(&initialConfig, values)
		require.NoError(t, err)
		err = Save(kubeconfigPath, *newConfig)
		require.NoError(t, err)

		generatedConfig, err := Load(kubeconfigPath)
		require.NoError(t, err)

		// Non-Teleport clusters should not change.
		wantConfig := initialConfig

		// Check for root-cluster-kube1.
		wantConfig.Clusters["root-cluster-kube1"] = &clientcmdapi.Cluster{
			ProxyURL:                 "http://localhost:12345",
			Server:                   rootKubeClusterAddr + "/v1/teleport/cm9vdC1jbHVzdGVy/a3ViZTE",
			CertificateAuthorityData: caData,
			TLSServerName:            rootClusterName,
			LocationOfOrigin:         kubeconfigPath,
			Extensions: map[string]runtime.Object{
				extProfileName: &runtime.Unknown{
					Raw:         []byte(`"` + proxyHost + `"`),
					ContentType: "application/json",
				},
				extTeleClusterName: &runtime.Unknown{
					Raw:         []byte(`"` + rootClusterName + `"`),
					ContentType: "application/json",
				},
				extKubeClusterName: &runtime.Unknown{
					Raw:         []byte(`"kube1"`),
					ContentType: "application/json",
				},
			},
		}
		wantConfig.Contexts["root-cluster-kube1"] = &clientcmdapi.Context{
			Namespace: "namespace",
			Cluster:   "root-cluster-kube1",
			AuthInfo:  "root-cluster-kube1",
			Extensions: map[string]runtime.Object{
				extProfileName: &runtime.Unknown{
					Raw:         []byte(`"` + proxyHost + `"`),
					ContentType: "application/json",
				},
				extTeleClusterName: &runtime.Unknown{
					Raw:         []byte(`"` + rootClusterName + `"`),
					ContentType: "application/json",
				},
				extKubeClusterName: &runtime.Unknown{
					Raw:         []byte(`"kube1"`),
					ContentType: "application/json",
				},
			},
		}
		wantConfig.AuthInfos["root-cluster-kube1"] = &clientcmdapi.AuthInfo{
			ClientCertificateData: caData,
			ClientKeyData:         clientKeyData,
			Impersonate:           "as",
			ImpersonateGroups:     []string{"group1", "group2"},
			Extensions: map[string]runtime.Object{
				extProfileName: &runtime.Unknown{
					Raw:         []byte(`"` + proxyHost + `"`),
					ContentType: "application/json",
				},
				extTeleClusterName: &runtime.Unknown{
					Raw:         []byte(`"` + rootClusterName + `"`),
					ContentType: "application/json",
				},
				extKubeClusterName: &runtime.Unknown{
					Raw:         []byte(`"kube1"`),
					ContentType: "application/json",
				},
			},
		}

		// Current context is updated.
		wantConfig.CurrentContext = "root-cluster-kube1"
		require.Empty(t, cmp.Diff(wantConfig, *generatedConfig, kubeconfigCmpOpts...))
	})
}

// TestCreateLocalProxyConfigStripsForeignExecEntries checks that
// a `tsh kube credentials` exec entry belonging to a *different* proxy is stripped
// from the generated local proxy kubeconfig (so it stays self-contained),
// while non-Teleport exec plugins are left untouched.
func TestCreateLocalProxyConfigStripsForeignExecEntries(t *testing.T) {
	const (
		curCluster   = "root-cluster"
		curProxyHost = "root-cluster.example.com"
		curKubeAddr  = "https://root-cluster.example.com"

		foreignCluster  = "other-cluster"
		foreignProxy    = "other-cluster.example.com:443"
		foreignKubeAddr = "https://other-cluster.example.com"
		foreignKubeCtx  = "other-cluster-kubeX"
	)

	kubeconfigPath, _ := setup(t)
	creds, _, err := genUserKeyRing("localhost")
	require.NoError(t, err)

	// Simulate `tsh kube login` against a *different* proxy.
	// This writes an exec-plugin entry that CreateLocalProxyConfig must strip.
	require.NoError(t, Update(kubeconfigPath, Values{
		TeleportClusterName: foreignCluster,
		ClusterAddr:         foreignKubeAddr,
		KubeClusters:        []string{"kubeX"},
		Credentials:         creds,
		Exec:                &ExecValues{TshBinaryPath: "/path/to/tsh"},
		ProxyAddr:           foreignProxy,
	}, false))

	configWithForeign, err := Load(kubeconfigPath)
	require.NoError(t, err)
	require.NotNil(t, configWithForeign.AuthInfos[foreignKubeCtx].Exec, "sanity: foreign exec entry should exist")

	newConfig, err := CreateLocalProxyConfig(configWithForeign, &LocalProxyValues{
		LocalProxyCAs:           map[string][]byte{curCluster: []byte("CAData")},
		TeleportProfileName:     curProxyHost,
		TeleportKubeClusterAddr: curKubeAddr,
		LocalProxyURL:           "http://localhost:12345",
		ClientKeyData:           []byte("clientKeyData"),
		Clusters: LocalProxyClusters{{
			TeleportCluster: curCluster,
			KubeCluster:     "kube1",
		}},
	})
	require.NoError(t, err)

	// The foreign Teleport exec entry must be gone.
	require.NotContains(t, newConfig.AuthInfos, foreignKubeCtx)
	require.NotContains(t, newConfig.Contexts, foreignKubeCtx)
	require.NotContains(t, newConfig.Clusters, foreignKubeCtx)

	// Non-Teleport entries from the user's kubeconfig are preserved, including a non-Teleport exec plugin.
	require.NotNil(t, newConfig.AuthInfos["support"].Exec, "non-Teleport exec plugin should be preserved")
	require.Contains(t, newConfig.AuthInfos, "developer")
	require.Contains(t, newConfig.AuthInfos, "admin")

	// The local proxy entry uses static creds, not an exec plugin.
	require.Contains(t, newConfig.AuthInfos, curCluster+"-kube1")
	require.Nil(t, newConfig.AuthInfos[curCluster+"-kube1"].Exec)
}
