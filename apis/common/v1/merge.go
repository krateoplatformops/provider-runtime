package v1

import (
	"github.com/imdario/mergo"
)

// MergeOptions Specifies merge options on a field path
type MergeOptions struct {
	// Specifies that already existing values in a merged map should be preserved
	// +optional
	KeepMapValues *bool `json:"keepMapValues,omitempty"`
	// Specifies that already existing elements in a merged slice should be preserved
	// +optional
	AppendSlice *bool `json:"appendSlice,omitempty"`
}

// MergoConfiguration the default behavior is to replace maps and slices
func (mo *MergeOptions) MergoConfiguration() []func(*mergo.Config) {
	config := []func(*mergo.Config){mergo.WithOverride}
	if mo == nil {
		return config
	}

	if mo.KeepMapValues != nil && *mo.KeepMapValues {
		config = config[:0]
	}
	if mo.AppendSlice != nil && *mo.AppendSlice {
		config = append(config, mergo.WithAppendSlice)
	}
	return config
}

// IsAppendSlice returns true if mo.AppendSlice is set to true
func (mo *MergeOptions) IsAppendSlice() bool {
	return mo != nil && mo.AppendSlice != nil && *mo.AppendSlice
}
