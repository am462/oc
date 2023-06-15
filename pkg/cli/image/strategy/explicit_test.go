package strategy

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	apicfgv1 "github.com/openshift/api/config/v1"

	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/openshift/library-go/pkg/image/reference"
)

func TestExplicitStrategyICSP(t *testing.T) {
	tests := []struct {
		name                 string
		icspList             []operatorv1alpha1.ImageContentSourcePolicy
		image                string
		imageSourcesExpected []string
	}{
		{
			name: "multiple ICSPs",
			icspList: []operatorv1alpha1.ImageContentSourcePolicy{
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "quay.io/multiple/icsps",
								Mirrors: []string{
									"someregistry/somerepo/release",
								},
							},
							{
								Source: "quay.io/ocp-test/another-release",
								Mirrors: []string{
									"someregistry/repo/does-not-exist",
								},
							},
						},
					},
				},
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "quay.io/multiple/icsps",
								Mirrors: []string{
									"anotherregistry/anotherrepo/release",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/multiple/icsps:4.5",
			imageSourcesExpected: []string{"someregistry/somerepo/release", "anotherregistry/anotherrepo/release", "quay.io/multiple/icsps"},
		},
		{
			name: "multiple mirrors, single source match",
			icspList: []operatorv1alpha1.ImageContentSourcePolicy{
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "docker.io/ocp-test/does-not-exist",
								Mirrors: []string{
									"does.not.exist/match/image",
								},
							},
							{
								Source: "quay.io/ocp-test/does-not-exist",
								Mirrors: []string{
									"exists/match/image",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/ocp-test/does-not-exist:4.7",
			imageSourcesExpected: []string{"exists/match/image", "quay.io/ocp-test/does-not-exist"},
		},
		{
			name: "single mirror and match",
			icspList: []operatorv1alpha1.ImageContentSourcePolicy{
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "quay.io/ocp-test/release",
								Mirrors: []string{
									"someregistry/mirrors/match",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"someregistry/mirrors/match", "quay.io/ocp-test/release"},
		},
		{
			name: "no source match",
			icspList: []operatorv1alpha1.ImageContentSourcePolicy{
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "docker.io/ocp-test/does-not-exist",
								Mirrors: []string{
									"does.not.exist/match/image",
								},
							},
							{
								Source: "quay.io/ocp-test/does-not-exist",
								Mirrors: []string{
									"exists/match/image",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/passed/image:4.5",
			imageSourcesExpected: []string{"quay.io/passed/image"},
		},
		{
			name: "multiple mirrors for single source match",
			icspList: []operatorv1alpha1.ImageContentSourcePolicy{
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "quay.io/ocp-test/release",
								Mirrors: []string{
									"someregistry/mirrors/match",
									"quay.io/another/release",
									"quay.io/andanother/release",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"someregistry/mirrors/match", "quay.io/another/release", "quay.io/andanother/release", "quay.io/ocp-test/release"},
		},
		{
			name: "docker.io vs registry-1.docker.io",
			icspList: []operatorv1alpha1.ImageContentSourcePolicy{
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "docker.io/ocp-test/release",
								Mirrors: []string{
									"quay.io/ocp-test/release",
								},
							},
						},
					},
				},
			},
			image:                "registry-1.docker.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"quay.io/ocp-test/release", "registry-1.docker.io/ocp-test/release"},
		},
		{
			name: "docker.io and registry-1.docker.io as source",
			icspList: []operatorv1alpha1.ImageContentSourcePolicy{
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "docker.io/ocp-test/release",
								Mirrors: []string{
									"quay.io/ocp-test/release",
								},
							},
						},
					},
				},
				{
					Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
						RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
							{
								Source: "registry-1.docker.io/ocp-test/release",
								Mirrors: []string{
									"quay.io/ocp-test/release",
								},
							},
						},
					},
				},
			},
			image:                "registry-1.docker.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"quay.io/ocp-test/release", "registry-1.docker.io/ocp-test/release"},
		},
		{
			name:                 "no ICSP",
			image:                "quay.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"quay.io/ocp-test/release"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := []reference.DockerImageReference{}
			for _, e := range tt.imageSourcesExpected {
				ref, _ := reference.Parse(e)
				expected = append(expected, ref)
			}

			alternates := NewICSPExplicitStrategy("name")
			readCount := 0
			onErr := alternates.(*explicitICSPStrategy)
			onErr.readICSPsFromFileFunc = func(string) ([]operatorv1alpha1.ImageContentSourcePolicy, error) {
				readCount++
				return tt.icspList, nil
			}
			imageRef, _ := reference.Parse(tt.image)

			actual, err := alternates.FirstRequest(context.Background(), imageRef)
			if err != nil {
				t.Errorf("Unexpected error %v", err)
				return
			}
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("Unexpected alternates got = %v, want %v", actual, expected)
			}

			actual2, err := alternates.OnFailure(context.Background(), imageRef)
			if err != nil {
				t.Errorf("Unexpected error %v", err)
				return
			}
			if !reflect.DeepEqual(actual2, actual) {
				t.Errorf("Unexpected alternates got = %v, want %v", actual, expected)
			}
			if readCount > 1 {
				t.Errorf("Unexpected number of ICSP reads, should be 1, got %d", readCount)
			}
		})
	}
}

func TestExplicitStrategyIDMS(t *testing.T) {
	tests := []struct {
		name                 string
		idmsList             []apicfgv1.ImageDigestMirrorSet
		image                string
		imageSourcesExpected []string
	}{
		{
			name: "multiple IDMSs",
			idmsList: []apicfgv1.ImageDigestMirrorSet{
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "quay.io/multiple/idmss",
								Mirrors: []apicfgv1.ImageMirror{
									"someregistry/somerepo/release",
								},
							},
							{
								Source: "quay.io/ocp-test/another-release",
								Mirrors: []apicfgv1.ImageMirror{
									"someregistry/repo/does-not-exist",
								},
							},
						},
					},
				},
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "quay.io/multiple/idmss",
								Mirrors: []apicfgv1.ImageMirror{
									"anotherregistry/anotherrepo/release",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/multiple/idmss:4.5",
			imageSourcesExpected: []string{"someregistry/somerepo/release", "anotherregistry/anotherrepo/release", "quay.io/multiple/idmss"},
		},
		{
			name: "multiple mirrors, single source match",
			idmsList: []apicfgv1.ImageDigestMirrorSet{
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "docker.io/ocp-test/does-not-exist",
								Mirrors: []apicfgv1.ImageMirror{
									"does.not.exist/match/image",
								},
							},
							{
								Source: "quay.io/ocp-test/does-not-exist",
								Mirrors: []apicfgv1.ImageMirror{
									"exists/match/image",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/ocp-test/does-not-exist:4.7",
			imageSourcesExpected: []string{"exists/match/image", "quay.io/ocp-test/does-not-exist"},
		},
		{
			name: "single mirror and match",
			idmsList: []apicfgv1.ImageDigestMirrorSet{
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "quay.io/ocp-test/release",
								Mirrors: []apicfgv1.ImageMirror{
									"someregistry/mirrors/match",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"someregistry/mirrors/match", "quay.io/ocp-test/release"},
		},
		{
			name: "no source match",
			idmsList: []apicfgv1.ImageDigestMirrorSet{
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "docker.io/ocp-test/does-not-exist",
								Mirrors: []apicfgv1.ImageMirror{
									"does.not.exist/match/image",
								},
							},
							{
								Source: "quay.io/ocp-test/does-not-exist",
								Mirrors: []apicfgv1.ImageMirror{
									"exists/match/image",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/passed/image:4.5",
			imageSourcesExpected: []string{"quay.io/passed/image"},
		},
		{
			name: "multiple mirrors for single source match",
			idmsList: []apicfgv1.ImageDigestMirrorSet{
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "quay.io/ocp-test/release",
								Mirrors: []apicfgv1.ImageMirror{
									"someregistry/mirrors/match",
									"quay.io/another/release",
									"quay.io/andanother/release",
								},
							},
						},
					},
				},
			},
			image:                "quay.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"someregistry/mirrors/match", "quay.io/another/release", "quay.io/andanother/release", "quay.io/ocp-test/release"},
		},
		{
			name: "docker.io vs registry-1.docker.io",
			idmsList: []apicfgv1.ImageDigestMirrorSet{
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "docker.io/ocp-test/release",
								Mirrors: []apicfgv1.ImageMirror{
									"quay.io/ocp-test/release",
								},
							},
						},
					},
				},
			},
			image:                "registry-1.docker.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"quay.io/ocp-test/release", "registry-1.docker.io/ocp-test/release"},
		},
		{
			name: "docker.io and registry-1.docker.io as source",
			idmsList: []apicfgv1.ImageDigestMirrorSet{
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "docker.io/ocp-test/release",
								Mirrors: []apicfgv1.ImageMirror{
									"quay.io/ocp-test/release",
								},
							},
						},
					},
				},
				{
					Spec: apicfgv1.ImageDigestMirrorSetSpec{
						ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
							{
								Source: "registry-1.docker.io/ocp-test/release",
								Mirrors: []apicfgv1.ImageMirror{
									"quay.io/ocp-test/release",
								},
							},
						},
					},
				},
			},
			image:                "registry-1.docker.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"quay.io/ocp-test/release", "registry-1.docker.io/ocp-test/release"},
		},
		{
			name:                 "no IDMS",
			image:                "quay.io/ocp-test/release:4.5",
			imageSourcesExpected: []string{"quay.io/ocp-test/release"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := []reference.DockerImageReference{}
			for _, e := range tt.imageSourcesExpected {
				ref, _ := reference.Parse(e)
				expected = append(expected, ref)
			}

			alternates := NewIDMSExplicitStrategy("name")
			readCount := 0
			onErr := alternates.(*explicitIDMSStrategy)
			onErr.readIDMSsFromFileFunc = func(string) ([]apicfgv1.ImageDigestMirrorSet, error) {
				readCount++
				return tt.idmsList, nil
			}
			imageRef, _ := reference.Parse(tt.image)

			actual, err := alternates.FirstRequest(context.Background(), imageRef)
			if err != nil {
				t.Errorf("Unexpected error %v", err)
				return
			}
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("Unexpected alternates got = %v, want %v", actual, expected)
			}

			actual2, err := alternates.OnFailure(context.Background(), imageRef)
			if err != nil {
				t.Errorf("Unexpected error %v", err)
				return
			}
			if !reflect.DeepEqual(actual2, actual) {
				t.Errorf("Unexpected alternates got = %v, want %v", actual, expected)
			}
			if readCount > 1 {
				t.Errorf("Unexpected number of IDMS reads, should be 1, got %d", readCount)
			}
		})
	}
}

func TestExplicitStrategyErrorsICSP(t *testing.T) {
	tests := []struct {
		name         string
		readICSPFunc readICSPsFromFileFunc
		image        string
		expectedErr  string
	}{
		{
			name:  "non-existent ICSP file",
			image: "quay.io/ocp-test/release:4.5",
			readICSPFunc: func(string) ([]operatorv1alpha1.ImageContentSourcePolicy, error) {
				return nil, errors.New("no ImageContentSourceFile")
			},
			expectedErr: "no ImageContentSourceFile",
		},
		{
			name:  "invalid source locator",
			image: "quay.io/ocp-test/release:4.5",
			readICSPFunc: func(string) ([]operatorv1alpha1.ImageContentSourcePolicy, error) {
				return []operatorv1alpha1.ImageContentSourcePolicy{
					{
						Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
							RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
								{
									Source: ".invalid-source-spec",
								},
							},
						},
					},
				}, nil
			},
			expectedErr: "invalid source",
		},
		{
			name:  "invalid mirror locator",
			image: "quay.io/ocp-test/release:4.5",
			readICSPFunc: func(string) ([]operatorv1alpha1.ImageContentSourcePolicy, error) {
				return []operatorv1alpha1.ImageContentSourcePolicy{
					{
						Spec: operatorv1alpha1.ImageContentSourcePolicySpec{
							RepositoryDigestMirrors: []operatorv1alpha1.RepositoryDigestMirrors{
								{
									Source: "quay.io/ocp-test/release",
									Mirrors: []string{
										".invalid-mirror-spec",
									},
								},
							},
						},
					},
				}, nil
			},
			expectedErr: "invalid mirror",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageRef, _ := reference.Parse(tt.image)
			alternates := NewICSPExplicitStrategy("name")
			onErr := alternates.(*explicitICSPStrategy)
			onErr.readICSPsFromFileFunc = tt.readICSPFunc
			_, err := alternates.FirstRequest(context.Background(), imageRef)
			if err == nil || !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Unexpected error, got %v, want %v", err, tt.expectedErr)
			}
		})
	}
}

func TestExplicitStrategyErrorsIDMS(t *testing.T) {
	tests := []struct {
		name         string
		readIDMSFunc readIDMSsFromFileFunc
		image        string
		expectedErr  string
	}{
		{
			name:  "non-existent IDMS file",
			image: "quay.io/ocp-test/release:4.5",
			readIDMSFunc: func(string) ([]apicfgv1.ImageDigestMirrorSet, error) {
				return nil, errors.New("no ImageDigestMirrorSet")
			},
			expectedErr: "no ImageDigestMirrorSet",
		},
		{
			name:  "invalid source locator",
			image: "quay.io/ocp-test/release:4.5",
			readIDMSFunc: func(string) ([]apicfgv1.ImageDigestMirrorSet, error) {
				return []apicfgv1.ImageDigestMirrorSet{
					{
						Spec: apicfgv1.ImageDigestMirrorSetSpec{
							ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
								{
									Source: ".invalid-source-spec",
								},
							},
						},
					},
				}, nil
			},
			expectedErr: "invalid source",
		},
		{
			name:  "invalid mirror locator",
			image: "quay.io/ocp-test/release:4.5",
			readIDMSFunc: func(string) ([]apicfgv1.ImageDigestMirrorSet, error) {
				return []apicfgv1.ImageDigestMirrorSet{
					{
						Spec: apicfgv1.ImageDigestMirrorSetSpec{
							ImageDigestMirrors: []apicfgv1.ImageDigestMirrors{
								{
									Source: "quay.io/ocp-test/release",
									Mirrors: []apicfgv1.ImageMirror{
										".invalid-mirror-spec",
									},
								},
							},
						},
					},
				}, nil
			},
			expectedErr: "invalid mirror",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageRef, _ := reference.Parse(tt.image)
			alternates := NewIDMSExplicitStrategy("name")
			onErr := alternates.(*explicitIDMSStrategy)
			onErr.readIDMSsFromFileFunc = tt.readIDMSFunc
			_, err := alternates.FirstRequest(context.Background(), imageRef)
			if err == nil || !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Unexpected error, got %v, want %v", err, tt.expectedErr)
			}
		})
	}
}
