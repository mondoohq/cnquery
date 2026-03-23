// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/recording"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRuntimeClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		Provider: &ConnectedProvider{
			Instance: &RunningProvider{
				Name: "test",
			},
		},
	}

	// Make sure the runtime was removed from the coordinator
	mockC.EXPECT().RemoveRuntime(r).Times(1)

	// Close the runtime
	r.Close()

	// Make sure the runtime is closed and the schema is empty
	assert.True(t, r.isClosed)
}

func TestRuntime_LookupResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		Provider: &ConnectedProvider{
			Instance: &RunningProvider{
				ID:   "test",
				Name: "test",
			},
		},
	}

	resName := "testResource"
	mockC.EXPECT().Schema().Times(1).Return(mockSchema)
	mockSchema.EXPECT().Lookup(resName).Times(1).Return(&resources.ResourceInfo{
		Name:     resName,
		Provider: BuiltinCoreID,
	})

	// Lookup the resource
	info, err := r.lookupResource(resName)
	require.NoError(t, err)
	assert.Equal(t, resName, info.Name)
	assert.Equal(t, BuiltinCoreID, info.Provider)
}

func TestRuntime_LookupResource_CoreOverridesAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		Provider: &ConnectedProvider{
			Instance: &RunningProvider{
				ID:   "test",
				Name: "test",
			},
		},
	}

	resName := "testResource"
	mockC.EXPECT().Schema().Times(1).Return(mockSchema)
	mockSchema.EXPECT().Lookup(resName).Times(1).Return(&resources.ResourceInfo{
		Name: resName,
		Others: []*resources.ResourceInfo{
			{Name: resName, Provider: "other"},
			{Name: resName, Provider: "test"}, // This matches the provider for the runtime
			{Name: resName, Provider: BuiltinCoreID},
		},
		Provider: "another",
	})

	// Lookup the resource
	info, err := r.lookupResource(resName)
	require.NoError(t, err)
	assert.Equal(t, resName, info.Name)
	assert.Equal(t, BuiltinCoreID, info.Provider) // we should get back the core resource
}

func TestRuntime_LookupResource_ProviderOverridesOthers(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		Provider: &ConnectedProvider{
			Instance: &RunningProvider{
				ID:   "test",
				Name: "test",
			},
		},
	}

	resName := "testResource"
	mockC.EXPECT().Schema().Times(1).Return(mockSchema)
	mockSchema.EXPECT().Lookup(resName).Times(1).Return(&resources.ResourceInfo{
		Name: resName,
		Others: []*resources.ResourceInfo{
			{Name: resName, Provider: "other"},
			{Name: resName, Provider: "test"}, // This matches the provider for the runtime
		},
		Provider: "another",
	})

	// Lookup the resource
	info, err := r.lookupResource(resName)
	require.NoError(t, err)
	assert.Equal(t, resName, info.Name)
	assert.Equal(t, "test", info.Provider) // we should get back the core resource
}

func TestRuntime_LookupFieldProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)
	p := &ConnectedProvider{
		Instance: &RunningProvider{
			ID:   BuiltinCoreID,
			Name: "test",
		},
	}
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		providers: map[string]*ConnectedProvider{
			BuiltinCoreID: p,
		},
		Provider: p,
	}

	resName := "testResource"
	fieldName := "testField"
	mockC.EXPECT().Schema().Times(1).Return(mockSchema)
	mockSchema.EXPECT().Lookup(resName).Times(1).Return(&resources.ResourceInfo{
		Name:     resName,
		Provider: BuiltinCoreID,
		Fields: map[string]*resources.Field{
			fieldName: {Name: fieldName, Provider: BuiltinCoreID},
		},
	})

	// Lookup the field
	_, res, field, err := r.lookupFieldProvider(resName, fieldName)
	require.NoError(t, err)
	assert.Equal(t, resName, res.Name)
	assert.Equal(t, BuiltinCoreID, res.Provider)
	assert.Equal(t, fieldName, field.Name)
	assert.Equal(t, BuiltinCoreID, field.Provider)
}

func TestRuntime_LookupFieldProvider_CoreOverridesAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)
	p := &ConnectedProvider{
		Instance: &RunningProvider{
			ID:   "test",
			Name: "test",
		},
	}
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		providers: map[string]*ConnectedProvider{
			BuiltinCoreID: p,
		},
		Provider: p,
	}

	resName := "testResource"
	fieldName := "testField"
	mockC.EXPECT().Schema().Times(1).Return(mockSchema)
	mockSchema.EXPECT().Lookup(resName).Times(1).Return(&resources.ResourceInfo{
		Name:     resName,
		Provider: BuiltinCoreID,
		Fields: map[string]*resources.Field{
			fieldName: {
				Name:     fieldName,
				Provider: "test",
				Others: []*resources.Field{
					{Name: fieldName, Provider: "other"},
					{Name: fieldName, Provider: BuiltinCoreID},
					{Name: fieldName, Provider: "test"}, // This matches the provider for the runtime
				},
			},
		},
	})

	// Lookup the field
	_, res, field, err := r.lookupFieldProvider(resName, fieldName)
	require.NoError(t, err)
	assert.Equal(t, resName, res.Name)
	assert.Equal(t, BuiltinCoreID, res.Provider) // we should get back the core resource

	assert.Equal(t, fieldName, field.Name)
	assert.Equal(t, BuiltinCoreID, field.Provider) // we should get back the core field
}

func TestRuntime_LookupFieldProvider_CoreOverridesAll_ResourceInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)
	p := &ConnectedProvider{
		Instance: &RunningProvider{
			ID:   BuiltinCoreID,
			Name: "test",
		},
	}
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		providers: map[string]*ConnectedProvider{
			BuiltinCoreID: p,
		},
		Provider: p,
	}

	// Here the core provider definition for the field is in another resource info
	resName := "testResource"
	fieldName := "testField"
	mockC.EXPECT().Schema().Times(1).Return(mockSchema)
	mockSchema.EXPECT().Lookup(resName).Times(1).Return(&resources.ResourceInfo{
		Name: resName,
		Others: []*resources.ResourceInfo{
			{Name: resName, Provider: "other"},
			{Name: resName, Provider: "test"}, // This matches the provider for the runtime
			{
				Name:     resName,
				Provider: BuiltinCoreID,
				Fields: map[string]*resources.Field{
					fieldName: {Name: fieldName, Provider: BuiltinCoreID},
				},
			},
		},
		Provider: "another",
		Fields: map[string]*resources.Field{
			fieldName: {
				Name:     fieldName,
				Provider: "test",
				Others: []*resources.Field{
					{Name: fieldName, Provider: "other"},
					{Name: fieldName, Provider: "another"},
				},
			},
		},
	})

	// Lookup the field
	_, res, field, err := r.lookupFieldProvider(resName, fieldName)
	require.NoError(t, err)
	assert.Equal(t, resName, res.Name)
	assert.Equal(t, BuiltinCoreID, res.Provider) // we should get back the core resource

	assert.Equal(t, fieldName, field.Name)
	assert.Equal(t, BuiltinCoreID, field.Provider) // we should get back the core field
}

func TestRuntime_LookupFieldProvider_ProviderOverridesOthers(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)
	p := &ConnectedProvider{
		Instance: &RunningProvider{
			ID:   "test",
			Name: "test",
		},
	}
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		providers: map[string]*ConnectedProvider{
			"test": p,
		},
		Provider: p,
	}

	resName := "testResource"
	fieldName := "testField"
	mockC.EXPECT().Schema().Times(1).Return(mockSchema)
	mockSchema.EXPECT().Lookup(resName).Times(1).Return(&resources.ResourceInfo{
		Name:     resName,
		Provider: "test",
		Fields: map[string]*resources.Field{
			fieldName: {
				Name:     fieldName,
				Provider: "another",
				Others: []*resources.Field{
					{Name: fieldName, Provider: "other"},
					{Name: fieldName, Provider: "test"}, // This matches the provider for the runtime
				},
			},
		},
	})

	// Lookup the field
	_, res, field, err := r.lookupFieldProvider(resName, fieldName)
	require.NoError(t, err)
	assert.Equal(t, resName, res.Name)
	assert.Equal(t, "test", res.Provider)
	assert.Equal(t, fieldName, field.Name)
	assert.Equal(t, "test", field.Provider)
}

func TestRuntime_LookupFieldProvider_ProviderOverridesOthers_ResourceInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockC := NewMockProvidersCoordinator(ctrl)
	mockSchema := NewMockResourcesSchema(ctrl)
	p := &ConnectedProvider{
		Instance: &RunningProvider{
			ID:   "test",
			Name: "test",
		},
	}
	r := &Runtime{
		coordinator: mockC,
		recording:   recording.Null{},
		providers: map[string]*ConnectedProvider{
			"test": p,
		},
		Provider: p,
	}

	// Here the core provider definition for the field is in another resource info
	resName := "testResource"
	fieldName := "testField"
	mockC.EXPECT().Schema().Times(1).Return(mockSchema)
	mockSchema.EXPECT().Lookup(resName).Times(1).Return(&resources.ResourceInfo{
		Name:     resName,
		Provider: "test",
		Others: []*resources.ResourceInfo{
			{Name: resName, Provider: "another"},
			{Name: resName, Provider: "test"}, // This matches the provider for the runtime
			{
				Name:     resName,
				Provider: "test",
				Fields: map[string]*resources.Field{
					fieldName: {Name: fieldName, Provider: "test"},
				},
			},
		},
		Fields: map[string]*resources.Field{
			fieldName: {
				Name:     fieldName,
				Provider: "another",
				Others: []*resources.Field{
					{Name: fieldName, Provider: "other"},
				},
			},
		},
	})

	// Lookup the field
	_, res, field, err := r.lookupFieldProvider(resName, fieldName)
	require.NoError(t, err)
	assert.Equal(t, resName, res.Name)
	assert.Equal(t, "test", res.Provider)
	assert.Equal(t, fieldName, field.Name)
	assert.Equal(t, "test", field.Provider)
}

func TestRuntime_CriticalErrors_Empty(t *testing.T) {
	r := &Runtime{}
	assert.Empty(t, r.CriticalErrors())
}

func TestRuntime_HandlePluginError_PanicRecordsCriticalError(t *testing.T) {
	r := &Runtime{}
	provider := &ConnectedProvider{
		Instance: &RunningProvider{Name: "aws"},
	}

	panicErr := status.Error(codes.Internal, "panic in provider aws: runtime error: nil pointer")
	handled, err := r.handlePluginError(panicErr, provider)

	assert.True(t, handled)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider panicked")

	critErrs := r.CriticalErrors()
	require.Len(t, critErrs, 1)
	assert.Contains(t, critErrs[0].Error(), "provider panicked")
}

func TestRuntime_HandlePluginError_CrashDoesNotRecordCriticalError(t *testing.T) {
	r := &Runtime{}
	provider := &ConnectedProvider{
		Instance: &RunningProvider{Name: "aws"},
	}

	crashErr := status.Error(codes.Unavailable, "connection lost")
	handled, err := r.handlePluginError(crashErr, provider)

	assert.False(t, handled)
	require.Error(t, err)
	assert.Empty(t, r.CriticalErrors())
}

func TestRuntime_HandlePluginError_NonPanicInternalDoesNotRecordCriticalError(t *testing.T) {
	r := &Runtime{}
	provider := &ConnectedProvider{
		Instance: &RunningProvider{Name: "aws"},
	}

	internalErr := status.Error(codes.Internal, "some other internal error")
	handled, err := r.handlePluginError(internalErr, provider)

	assert.False(t, handled)
	require.Error(t, err)
	assert.Empty(t, r.CriticalErrors())
}

func TestRuntime_CriticalErrors_MultiplePanics(t *testing.T) {
	r := &Runtime{}
	provider := &ConnectedProvider{
		Instance: &RunningProvider{Name: "aws"},
	}

	for i := 0; i < 3; i++ {
		panicErr := status.Error(codes.Internal, "panic in provider aws: error")
		r.handlePluginError(panicErr, provider) // nolint:errcheck
	}

	assert.Len(t, r.CriticalErrors(), 3)
}
