package resolver

import (
	"context"
	"fmt"
	"testing"

	"github.com/blang/semver"
	"github.com/operator-framework/operator-registry/pkg/client"
	opregistry "github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry/resolver/fakes"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"
)

func TestNewNamespaceSourceQuerier(t *testing.T) {
	emptySources := map[CatalogKey]registry.ClientInterface{}
	nonEmptySources := map[CatalogKey]registry.ClientInterface{
		CatalogKey{"test", "ns"}: &registry.Client{
			Client: &client.Client{
				Registry: &fakes.FakeRegistryClient{},
			},
		},
	}

	type args struct {
		sources map[CatalogKey]registry.ClientInterface
	}
	tests := []struct {
		name string
		args args
		want *NamespaceSourceQuerier
	}{
		{
			name: "nil",
			args: args{
				sources: nil,
			},
			want: &NamespaceSourceQuerier{sources: nil},
		},
		{
			name: "empty",
			args: args{
				sources: emptySources,
			},
			want: &NamespaceSourceQuerier{sources: emptySources},
		},
		{
			name: "nonEmpty",
			args: args{
				sources: nonEmptySources,
			},
			want: &NamespaceSourceQuerier{sources: nonEmptySources},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, NewNamespaceSourceQuerier(tt.args.sources), tt.want)
		})
	}
}

func TestNamespaceSourceQuerier_Queryable(t *testing.T) {
	type fields struct {
		sources map[CatalogKey]registry.ClientInterface
	}
	tests := []struct {
		name   string
		fields fields
		error  error
	}{
		{
			name: "nil",
			fields: fields{
				sources: nil,
			},
			error: fmt.Errorf("no catalog sources available"),
		},
		{
			name: "empty",
			fields: fields{
				sources: map[CatalogKey]registry.ClientInterface{},
			},
			error: fmt.Errorf("no catalog sources available"),
		},
		{
			name: "nonEmpty",
			fields: fields{
				sources: map[CatalogKey]registry.ClientInterface{
					CatalogKey{"test", "ns"}: &registry.Client{
						Client: &client.Client{
							Registry: &fakes.FakeRegistryClient{},
						},
					},
				},
			},
			error: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &NamespaceSourceQuerier{
				sources: tt.fields.sources,
			}
			require.Equal(t, q.Queryable(), tt.error)
		})
	}
}

func TestNamespaceSourceQuerier_FindProvider(t *testing.T) {
	fakeSource := fakes.FakeClientInterface{}
	fakeSource2 := fakes.FakeClientInterface{}
	sources := map[CatalogKey]registry.ClientInterface{
		CatalogKey{"test", "ns"}:  &fakeSource,
		CatalogKey{"test2", "ns"}: &fakeSource2,
	}
	bundle := opregistry.NewBundle("test", "testPkg", "testChannel")
	bundle2 := opregistry.NewBundle("test2", "testPkg2", "testChannel2")
	fakeSource.GetBundleThatProvidesStub = func(ctx context.Context, group, version, kind string) (*opregistry.Bundle, error) {
		if group != "group" || version != "version" || kind != "kind" {
			return nil, fmt.Errorf("Not Found")
		}
		return bundle, nil
	}
	fakeSource2.GetBundleThatProvidesStub = func(ctx context.Context, group, version, kind string) (*opregistry.Bundle, error) {
		if group != "group2" || version != "version2" || kind != "kind2" {
			return nil, fmt.Errorf("Not Found")
		}
		return bundle2, nil
	}
	fakeSource.FindBundleThatProvidesStub = func(ctx context.Context, group, version, kind, pkgName string) (*opregistry.Bundle, error) {
		if group != "group" || version != "version" || kind != "kind" {
			return nil, fmt.Errorf("Not Found")
		}
		return bundle, nil
	}
	fakeSource2.FindBundleThatProvidesStub = func(ctx context.Context, group, version, kind, pkgName string) (*opregistry.Bundle, error) {
		if group != "group2" || version != "version2" || kind != "kind2" {
			return nil, fmt.Errorf("Not Found")
		}
		return bundle2, nil
	}

	type fields struct {
		sources map[CatalogKey]registry.ClientInterface
	}
	type args struct {
		api        opregistry.APIKey
		catalogKey CatalogKey
	}
	type out struct {
		bundle *opregistry.Bundle
		key    *CatalogKey
		err    error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		out    out
	}{
		{
			fields: fields{
				sources: sources,
			},
			args: args{
				api:        opregistry.APIKey{"group", "version", "kind", "plural"},
				catalogKey: CatalogKey{},
			},
			out: out{
				bundle: bundle,
				key:    &CatalogKey{Name: "test", Namespace: "ns"},
				err:    nil,
			},
		},
		{
			fields: fields{
				sources: nil,
			},
			args: args{
				api:        opregistry.APIKey{"group", "version", "kind", "plural"},
				catalogKey: CatalogKey{},
			},
			out: out{
				bundle: nil,
				key:    nil,
				err:    fmt.Errorf("group/version/kind (plural) not provided by a package in any CatalogSource"),
			},
		},
		{
			fields: fields{
				sources: sources,
			},
			args: args{
				api:        opregistry.APIKey{"group2", "version2", "kind2", "plural2"},
				catalogKey: CatalogKey{Name: "test2", Namespace: "ns"},
			},
			out: out{
				bundle: bundle2,
				key:    &CatalogKey{Name: "test2", Namespace: "ns"},
				err:    nil,
			},
		},
		{
			fields: fields{
				sources: sources,
			},
			args: args{
				api:        opregistry.APIKey{"group2", "version2", "kind2", "plural2"},
				catalogKey: CatalogKey{Name: "test3", Namespace: "ns"},
			},
			out: out{
				bundle: bundle2,
				key:    &CatalogKey{Name: "test2", Namespace: "ns"},
				err:    nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &NamespaceSourceQuerier{
				sources: tt.fields.sources,
			}
			bundle, key, err := q.FindProvider(tt.args.api, tt.args.catalogKey, "")
			require.Equal(t, tt.out.err, err)
			require.Equal(t, tt.out.bundle, bundle)
			require.Equal(t, tt.out.key, key)
		})
	}
}

func TestNamespaceSourceQuerier_FindPackage(t *testing.T) {
	initialSource := fakes.FakeClientInterface{}
	otherSource := fakes.FakeClientInterface{}
	initalBundle := opregistry.NewBundle("test", "testPkg", "testChannel")
	startingBundle := opregistry.NewBundle("starting-test", "testPkg", "testChannel")
	otherBundle := opregistry.NewBundle("other", "otherPkg", "otherChannel")
	initialSource.GetBundleStub = func(ctx context.Context, pkgName, channelName, csvName string) (*opregistry.Bundle, error) {
		if csvName != startingBundle.Name {
			return nil, fmt.Errorf("not found")
		}
		return startingBundle, nil
	}
	initialSource.GetBundleInPackageChannelStub = func(ctx context.Context, pkgName, channelName string) (*opregistry.Bundle, error) {
		if pkgName != initalBundle.Name {
			return nil, fmt.Errorf("not found")
		}
		return initalBundle, nil
	}
	otherSource.GetBundleInPackageChannelStub = func(ctx context.Context, pkgName, channelName string) (*opregistry.Bundle, error) {
		if pkgName != otherBundle.Name {
			return nil, fmt.Errorf("not found")
		}
		return otherBundle, nil
	}
	initialKey := CatalogKey{"initial", "ns"}
	otherKey := CatalogKey{"other", "other"}
	sources := map[CatalogKey]registry.ClientInterface{
		initialKey: &initialSource,
		otherKey:   &otherSource,
	}

	type fields struct {
		sources map[CatalogKey]registry.ClientInterface
	}
	type args struct {
		pkgName       string
		channelName   string
		startingCSV   string
		initialSource CatalogKey
	}
	type out struct {
		bundle *opregistry.Bundle
		key    *CatalogKey
		err    error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		out    out
	}{
		{
			name:   "Initial/Found",
			fields: fields{sources: sources},
			args:   args{"test", "testChannel", "", CatalogKey{"initial", "ns"}},
			out:    out{bundle: initalBundle, key: &initialKey, err: nil},
		},
		{
			name:   "Initial/CatalogNotFound",
			fields: fields{sources: sources},
			args:   args{"test", "testChannel", "", CatalogKey{"absent", "found"}},
			out:    out{bundle: nil, key: nil, err: fmt.Errorf("CatalogSource {absent found} not found")},
		},
		{
			name:   "Initial/StartingCSVFound",
			fields: fields{sources: sources},
			args:   args{"test", "testChannel", "starting-test", CatalogKey{"initial", "ns"}},
			out:    out{bundle: startingBundle, key: &initialKey, err: nil},
		},
		{
			name:   "Initial/StartingCSVNotFound",
			fields: fields{sources: sources},
			args:   args{"test", "testChannel", "non-existent", CatalogKey{"initial", "ns"}},
			out:    out{bundle: nil, key: nil, err: fmt.Errorf("not found")},
		},
		{
			name:   "Other/Found",
			fields: fields{sources: sources},
			args:   args{"other", "testChannel", "", CatalogKey{"", ""}},
			out:    out{bundle: otherBundle, key: &otherKey, err: nil},
		},
		{
			name:   "NotFound",
			fields: fields{sources: sources},
			args:   args{"nope", "not", "", CatalogKey{"", ""}},
			out:    out{bundle: nil, err: fmt.Errorf("nope/not not found in any available CatalogSource")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &NamespaceSourceQuerier{
				sources: tt.fields.sources,
			}
			var got *opregistry.Bundle
			var key *CatalogKey
			var err error
			if tt.args.startingCSV != "" {
				got, key, err = q.FindBundle(tt.args.pkgName, tt.args.channelName, tt.args.startingCSV, tt.args.initialSource)
			} else {
				got, key, err = q.FindLatestBundle(tt.args.pkgName, tt.args.channelName, tt.args.initialSource)
			}
			require.Equal(t, tt.out.err, err)
			require.Equal(t, tt.out.bundle, got)
			require.Equal(t, tt.out.key, key)
		})
	}
}

func TestNamespaceSourceQuerier_FindReplacement(t *testing.T) {
	// TODO: clean up this test setup
	initialSource := fakes.FakeClientInterface{}
	otherSource := fakes.FakeClientInterface{}
	replacementSource := fakes.FakeClientInterface{}
	replacementAndLatestSource := fakes.FakeClientInterface{}
	replacementAndNoAnnotationLatestSource := fakes.FakeClientInterface{}

	latestVersion := semver.MustParse("1.0.0-1556661308")
	csv := v1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.ClusterServiceVersionKind,
			APIVersion: v1alpha1.GroupVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "latest",
			Namespace: "placeholder",
			Annotations: map[string]string{
				"olm.skipRange": ">= 1.0.0-0 < 1.0.0-1556661308",
			},
		},
		Spec: v1alpha1.ClusterServiceVersionSpec{
			CustomResourceDefinitions: v1alpha1.CustomResourceDefinitions{
				Owned:    []v1alpha1.CRDDescription{},
				Required: []v1alpha1.CRDDescription{},
			},
			APIServiceDefinitions: v1alpha1.APIServiceDefinitions{
				Owned:    []v1alpha1.APIServiceDescription{},
				Required: []v1alpha1.APIServiceDescription{},
			},
			Version: version.OperatorVersion{latestVersion},
		},
	}
	csvUnst, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&csv)
	require.NoError(t, err)

	nextBundle := opregistry.NewBundle("test.v1", "testPkg", "testChannel")
	latestBundle := opregistry.NewBundle("latest", "testPkg", "testChannel", &unstructured.Unstructured{Object: csvUnst})

	csv.SetAnnotations(map[string]string{})
	csvUnstNoAnnotation, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&csv)
	require.NoError(t, err)
	latestBundleNoAnnotation := opregistry.NewBundle("latest", "testPkg", "testChannel", &unstructured.Unstructured{Object: csvUnstNoAnnotation})

	initialSource.GetReplacementBundleInPackageChannelStub = func(ctx context.Context, bundleName, pkgName, channelName string) (*opregistry.Bundle, error) {
		return nil, fmt.Errorf("not found")
	}
	replacementSource.GetReplacementBundleInPackageChannelStub = func(ctx context.Context, bundleName, pkgName, channelName string) (*opregistry.Bundle, error) {
		return nextBundle, nil
	}
	initialSource.GetBundleInPackageChannelStub = func(ctx context.Context, pkgName, channelName string) (*opregistry.Bundle, error) {
		if pkgName != latestBundle.Package {
			return nil, fmt.Errorf("not found")
		}
		return latestBundle, nil
	}
	otherSource.GetBundleInPackageChannelStub = func(ctx context.Context, pkgName, channelName string) (*opregistry.Bundle, error) {
		if pkgName != latestBundle.Package {
			return nil, fmt.Errorf("not found")
		}
		return latestBundle, nil
	}
	replacementAndLatestSource.GetReplacementBundleInPackageChannelStub = func(ctx context.Context, bundleName, pkgName, channelName string) (*opregistry.Bundle, error) {
		return nextBundle, nil
	}
	replacementAndLatestSource.GetBundleInPackageChannelStub = func(ctx context.Context, pkgName, channelName string) (*opregistry.Bundle, error) {
		return latestBundle, nil
	}
	replacementAndNoAnnotationLatestSource.GetReplacementBundleInPackageChannelStub = func(ctx context.Context, bundleName, pkgName, channelName string) (*opregistry.Bundle, error) {
		return nextBundle, nil
	}
	replacementAndNoAnnotationLatestSource.GetBundleInPackageChannelStub = func(ctx context.Context, pkgName, channelName string) (*opregistry.Bundle, error) {
		return latestBundleNoAnnotation, nil
	}

	initialKey := CatalogKey{"initial", "ns"}
	otherKey := CatalogKey{"other", "other"}
	replacementKey := CatalogKey{"replacement", "ns"}
	replacementAndLatestKey := CatalogKey{"replat", "ns"}
	replacementAndNoAnnotationLatestKey := CatalogKey{"replatbad", "ns"}

	sources := map[CatalogKey]registry.ClientInterface{
		initialKey:                          &initialSource,
		otherKey:                            &otherSource,
		replacementKey:                      &replacementSource,
		replacementAndLatestKey:             &replacementAndLatestSource,
		replacementAndNoAnnotationLatestKey: &replacementAndNoAnnotationLatestSource,
	}

	startVersion := semver.MustParse("1.0.0-0")
	notInRange := semver.MustParse("1.0.0-1556661347")

	type fields struct {
		sources map[CatalogKey]registry.ClientInterface
	}
	type args struct {
		currentVersion *semver.Version
		pkgName        string
		channelName    string
		bundleName     string
		initialSource  CatalogKey
	}
	type out struct {
		bundle *opregistry.Bundle
		key    *CatalogKey
		err    error
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		out    out
	}{
		{
			name:   "FindsLatestInPrimaryCatalog",
			fields: fields{sources: sources},
			args:   args{&startVersion, "testPkg", "testChannel", "test.v1", initialKey},
			out:    out{bundle: latestBundle, key: &initialKey, err: nil},
		},
		{
			name:   "FindsLatestInSecondaryCatalog",
			fields: fields{sources: sources},
			args:   args{&startVersion, "testPkg", "testChannel", "test.v1", otherKey},
			out:    out{bundle: latestBundle, key: &otherKey, err: nil},
		},
		{
			name:   "PrefersLatestToReplaced/SameCatalog",
			fields: fields{sources: sources},
			args:   args{&startVersion, "testPkg", "testChannel", "test.v1", replacementAndLatestKey},
			out:    out{bundle: latestBundle, key: &replacementAndLatestKey, err: nil},
		},
		{
			name:   "PrefersLatestToReplaced/OtherCatalog",
			fields: fields{sources: sources},
			args:   args{&startVersion, "testPkg", "testChannel", "test.v1", initialKey},
			out:    out{bundle: latestBundle, key: &initialKey, err: nil},
		},
		{
			name:   "IgnoresLatestWithoutAnnotation",
			fields: fields{sources: sources},
			args:   args{&startVersion, "testPkg", "testChannel", "test.v1", replacementAndNoAnnotationLatestKey},
			out:    out{bundle: nextBundle, key: &replacementAndNoAnnotationLatestKey, err: nil},
		},
		{
			name:   "IgnoresLatestNotInRange",
			fields: fields{sources: sources},
			args:   args{&notInRange, "testPkg", "testChannel", "test.v1", replacementAndLatestKey},
			out:    out{bundle: nextBundle, key: &replacementAndLatestKey, err: nil},
		},
		{
			name:   "IgnoresLatestAtLatest",
			fields: fields{sources: sources},
			args:   args{&latestVersion, "testPkg", "testChannel", "test.v1", otherKey},
			out:    out{bundle: nil, key: nil, err: nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &NamespaceSourceQuerier{
				sources: tt.fields.sources,
			}
			var got *opregistry.Bundle
			var key *CatalogKey
			var err error
			got, key, err = q.FindReplacement(tt.args.currentVersion, tt.args.bundleName, tt.args.pkgName, tt.args.channelName, tt.args.initialSource)
			if err != nil {
				t.Log(err.Error())
			}
			require.Equal(t, tt.out.err, err)
			require.Equal(t, tt.out.bundle, got)
			require.Equal(t, tt.out.key, key)
		})
	}
}
