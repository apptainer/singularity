module github.com/hpcng/singularity/cli-example-plugin

go 1.13

require (
	github.com/spf13/cobra v1.0.0
	github.com/hpcng/singularity v0.0.0
)

replace github.com/hpcng/singularity => ./singularity_source
