// Code generated by "stringer -type=Feature"; DO NOT EDIT.

package cnquery

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[MassQueries-1]
	_ = x[PiperCode-2]
	_ = x[BoolAssertions-3]
	_ = x[K8sNodeDiscovery-4]
	_ = x[MQLAssetContext-5]
	_ = x[ErrorsAsFailures-6]
	_ = x[StoreResourcesData-7]
	_ = x[FineGrainedAssets-8]
	_ = x[SerialNumberAsID-9]
	_ = x[ForceShellCompletion-10]
	_ = x[ResourceContext-11]
	_ = x[FailIfNoEntryPoints-12]
}

const _Feature_name = "MassQueriesPiperCodeBoolAssertionsK8sNodeDiscoveryMQLAssetContextErrorsAsFailuresStoreResourcesDataFineGrainedAssetsSerialNumberAsIDForceShellCompletionResourceContextFailIfNoEntryPoints"

var _Feature_index = [...]uint8{0, 11, 20, 34, 50, 65, 81, 99, 116, 132, 152, 167, 186}

func (i Feature) String() string {
	i -= 1
	if i >= Feature(len(_Feature_index)-1) {
		return "Feature(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _Feature_name[_Feature_index[i]:_Feature_index[i+1]]
}
