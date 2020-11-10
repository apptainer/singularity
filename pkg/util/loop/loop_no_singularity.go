// +build !singularity_engine

package loop

// GetMaxLoopDevices Return the maximum number of loop devices allowed
func GetMaxLoopDevices() int {
	// externally imported package, use the default value
	return 256
}
