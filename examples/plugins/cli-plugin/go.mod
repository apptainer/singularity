module github.com/hpcng/singularity/cli-example-plugin

go 1.16

require (
	github.com/hpcng/singularity v0.0.0
	github.com/spf13/cobra v1.2.1
)

replace github.com/hpcng/singularity => ./singularity_source
