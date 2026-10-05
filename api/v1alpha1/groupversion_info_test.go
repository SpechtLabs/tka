package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAddToScheme(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme))

	tests := []struct {
		name string
		obj  runtime.Object
		kind string
	}{
		{name: "signin", obj: &TkaSignin{}, kind: "TkaSignin"},
		{name: "signin list", obj: &TkaSigninList{}, kind: "TkaSigninList"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvks, _, err := scheme.ObjectKinds(tt.obj)
			require.NoError(t, err)
			require.Len(t, gvks, 1)
			assert.Equal(t, GroupVersion.WithKind(tt.kind), gvks[0])
		})
	}

	// metav1.AddToGroupVersion registers the meta types the API machinery
	// needs for the group, such as the options for list and watch calls.
	assert.True(t, scheme.Recognizes(GroupVersion.WithKind("ListOptions")))
}
