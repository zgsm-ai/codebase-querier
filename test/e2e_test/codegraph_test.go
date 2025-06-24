package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"testing"
)

func TestCodegraphSummary(t *testing.T) {
	assert.NotPanics(t, func() {
		ctx := context.Background()
		logx.DisableStat()
		codebasePath := "G:\\codebase-store\\7ec27814b60376c6fba936bf1fcaf430f8a84c37eb8f093f91e5664fd26c3160"
		graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
		summary, err := graphStore.GetIndexSummary(ctx, 1, codebasePath)
		assert.NoError(t, err)
		assert.True(t, summary.TotalFiles > 0)

	})

}

func TestQueryGraphByCodeSnippet(t *testing.T) {
	ctx := context.Background()
	logx.DisableStat()
	codebasePath := "G:\\codebase-store\\7ec27814b60376c6fba936bf1fcaf430f8a84c37eb8f093f91e5664fd26c3160"
	graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	assert.NoError(t, err)
	defer graphStore.Close()
	testCases := []struct {
		Name    string
		req     *types.DefinitionRequest
		wantErr error
	}{
		{
			Name: "go",
			req: &types.DefinitionRequest{
				ClientId:     "1",
				CodebasePath: codebasePath,
				FilePath:     "test.go", // TODO 下面的var config AdmissionConfig 没有解析成功； d := yaml.NewYAMLOrJSONDecoder 这种short_val 前面的variable没有解析成功； arguments没有去掉首尾括号；对象.字段未做解析
				CodeSnippet: `
package apiserver

import (
	"fmt"

	"k8s.io/apiserver/pkg/registry/generic"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
	svmrest "k8s.io/kubernetes/pkg/registry/storagemigration/rest"

	admissionregistrationrest "k8s.io/kubernetes/pkg/registry/admissionregistration/rest"
	apiserverinternalrest "k8s.io/kubernetes/pkg/registry/apiserverinternal/rest"
	authenticationrest "k8s.io/kubernetes/pkg/registry/authentication/rest"
	authorizationrest "k8s.io/kubernetes/pkg/registry/authorization/rest"
	certificatesrest "k8s.io/kubernetes/pkg/registry/certificates/rest"
	coordinationrest "k8s.io/kubernetes/pkg/registry/coordination/rest"
	corerest "k8s.io/kubernetes/pkg/registry/core/rest"
	eventsrest "k8s.io/kubernetes/pkg/registry/events/rest"
	flowcontrolrest "k8s.io/kubernetes/pkg/registry/flowcontrol/rest"
	rbacrest "k8s.io/kubernetes/pkg/registry/rbac/rest"
)

// RESTStorageProvider is a factory type for REST storage.
type RESTStorageProvider interface {
	GroupName() string
	NewRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (genericapiserver.APIGroupInfo, error)
}

// NewCoreGenericConfig returns a new core rest generic config.
func (c *CompletedConfig) NewCoreGenericConfig() *corerest.GenericConfig {
	return &corerest.GenericConfig{
		StorageFactory:              c.Extra.StorageFactory,
		EventTTL:                    c.Extra.EventTTL,
		LoopbackClientConfig:        c.Generic.LoopbackClientConfig,
		ServiceAccountIssuer:        c.Extra.ServiceAccountIssuer,
		ExtendExpiration:            c.Extra.ExtendExpiration,
		ServiceAccountMaxExpiration: c.Extra.ServiceAccountMaxExpiration,
		MaxExtendedExpiration:       c.Extra.ServiceAccountExtendedMaxExpiration,
		APIAudiences:                c.Generic.Authentication.APIAudiences,
		Informers:                   c.Extra.VersionedInformers,
	}
}

// GenericStorageProviders returns a set of APIs for a generic control plane.
// They ought to be a subset of those served by kube-apiserver.
func (c *CompletedConfig) GenericStorageProviders(discovery discovery.DiscoveryInterface) ([]RESTStorageProvider, error) {
	// The order here is preserved in discovery.
	// If resources with identical names exist in more than one of these groups (e.g. "deployments.apps"" and "deployments.extensions"),
	// the order of this list determines which group an unqualified resource language (e.g. "deployments") should prefer.
	// This priority order is used for local discovery, but it ends up aggregated in k8s.io/kubernetes/cmd/kube-apiserver/app/aggregator.go
	// with specific priorities.
	// TODO: describe the priority all the way down in the RESTStorageProviders and plumb it back through the various discovery
	// handlers that we have.
	return []RESTStorageProvider{
		c.NewCoreGenericConfig(),
		apiserverinternalrest.StorageProvider{},
		authenticationrest.RESTStorageProvider{Authenticator: c.Generic.Authentication.Authenticator, APIAudiences: c.Generic.Authentication.APIAudiences},
		authorizationrest.RESTStorageProvider{Authorizer: c.Generic.Authorization.Authorizer, RuleResolver: c.Generic.RuleResolver},
		certificatesrest.RESTStorageProvider{},
		coordinationrest.RESTStorageProvider{},
		rbacrest.RESTStorageProvider{Authorizer: c.Generic.Authorization.Authorizer},
		svmrest.RESTStorageProvider{},
		flowcontrolrest.RESTStorageProvider{InformerFactory: c.Generic.SharedInformerFactory},
		admissionregistrationrest.RESTStorageProvider{Authorizer: c.Generic.Authorization.Authorizer, DiscoveryClient: discovery},
		eventsrest.RESTStorageProvider{TTL: c.EventTTL},
	}, nil
}

// InstallAPIs will install the APIs for the restStorageProviders if they are enabled.
func (s *Server) InstallAPIs(restStorageProviders ...RESTStorageProvider) error {
	nonLegacy := []*genericapiserver.APIGroupInfo{}

	// used later in the loop to filter the served resource by those that have expired.
	resourceExpirationEvaluatorOpts := genericapiserver.ResourceExpirationEvaluatorOptions{
		CurrentVersion:                          s.GenericAPIServer.EffectiveVersion.EmulationVersion(),
		Prerelease:                              s.GenericAPIServer.EffectiveVersion.BinaryVersion().PreRelease(),
		EmulationForwardCompatible:              s.GenericAPIServer.EmulationForwardCompatible,
		RuntimeConfigEmulationForwardCompatible: s.GenericAPIServer.RuntimeConfigEmulationForwardCompatible,
	}
	resourceExpirationEvaluator, err := genericapiserver.NewResourceExpirationEvaluatorFromOptions(resourceExpirationEvaluatorOpts)
	if err != nil {
		return err
	}

	for _, restStorageBuilder := range restStorageProviders {
		groupName := restStorageBuilder.GroupName()
		apiGroupInfo, err := restStorageBuilder.NewRESTStorage(s.APIResourceConfigSource, s.RESTOptionsGetter)
		if err != nil {
			return fmt.Errorf("problem initializing API group %q: %w", groupName, err)
		}
		if len(apiGroupInfo.VersionedResourcesStorageMap) == 0 {
			// If we have no storage for any resource configured, this API group is effectively disabled.
			// This can happen when an entire API group, version, or development-stage (alpha, beta, GA) is disabled.
			klog.Infof("API group %q is not enabled, skipping.", groupName)
			continue
		}

		// Remove resources that serving kinds that are removed or not introduced yet at the current version.
		// We do this here so that we don't accidentally serve versions without resources or openapi information that for kinds we don't serve.
		// This is a spot above the construction of individual storage handlers so that no sig accidentally forgets to check.
		err = resourceExpirationEvaluator.RemoveUnavailableKinds(groupName, apiGroupInfo.Scheme, apiGroupInfo.VersionedResourcesStorageMap, s.APIResourceConfigSource)
		if err != nil {
			return err
		}
		if len(apiGroupInfo.VersionedResourcesStorageMap) == 0 {
			klog.V(1).Infof("Removing API group %v because it is time to stop serving it because it has no versions per APILifecycle.", groupName)
			continue
		}

		klog.V(1).Infof("Enabling API group %q.", groupName)

		if postHookProvider, ok := restStorageBuilder.(genericapiserver.PostStartHookProvider); ok {
			language, hook, err := postHookProvider.PostStartHook()
			if err != nil {
				return fmt.Errorf("error building PostStartHook: %w", err)
			}
			s.GenericAPIServer.AddPostStartHookOrDie(language, hook)
		}

		if len(groupName) == 0 {
			// the legacy group for core APIs is special that it is installed into /api via this special install method.
			if err := s.GenericAPIServer.InstallLegacyAPIGroup(genericapiserver.DefaultLegacyAPIPrefix, &apiGroupInfo); err != nil {
				return fmt.Errorf("error in registering legacy API: %w", err)
			}
		} else {
			// everything else goes to /apis
			nonLegacy = append(nonLegacy, &apiGroupInfo)
		}
	}

	if err := s.GenericAPIServer.InstallAPIGroups(nonLegacy...); err != nil {
		return fmt.Errorf("error in registering group versions: %w", err)
	}
	return nil
}
`,
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			definitions, err := graphStore.QueryDefinitions(ctx, tt.req)
			assert.NoError(t, err)
			assert.NotEmpty(t, definitions)
		})
	}

}
