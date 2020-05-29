module github.com/sylabs/singularity/e2e-cli-plugin

go 1.13

require (
	github.com/spf13/cobra v1.0.0
	github.com/sylabs/singularity v0.0.0
)

replace github.com/sylabs/singularity => ./singularity_source
