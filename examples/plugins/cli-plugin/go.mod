module github.com/sylabs/singularity/cli-example-plugin

go 1.13

require (
	github.com/spf13/cobra v1.1.3
	github.com/sylabs/singularity v0.0.0
)

replace github.com/sylabs/singularity => ./singularity_source
