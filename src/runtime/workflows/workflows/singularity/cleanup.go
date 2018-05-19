package runtime

/*
 * see https://github.com/opencontainers/runtime-spec/blob/master/runtime.md#lifecycle
 * we will run step 8/9 there
 */
func (c *RuntimeEngine) CleanupContainer() error {
	return nil
}
