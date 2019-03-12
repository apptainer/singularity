# JSON Response

[![GoDoc](https://godoc.org/github.com/sylabs/json-resp?status.svg)](https://godoc.org/github.com/sylabs/json-resp)
[![Build Status](https://circleci.com/gh/sylabs/json-resp.svg?style=shield)](https://circleci.com/gh/sylabs/workflows/json-resp)
[![Go Report Card](https://goreportcard.com/badge/github.com/sylabs/json-resp)](https://goreportcard.com/report/github.com/sylabs/json-resp)

The `json-resp` package contains a small set of functions that are used to marshall and unmarshall response data and errors in JSON format.

## Quick Start

Install the [CircleCI Local CLI](https://circleci.com/docs/2.0/local-cli/). See the [Continuous Integration](#continuous-integration) section below for more detail.

To build and test:

```sh
circleci build
```

## Continuous Integration

This package uses [CircleCI](https://circleci.com) for Continuous Integration (CI). It runs automatically on commits and pull requests involving a protected branch. All CI checks must pass before a merge to a proected branch can be performed.

The CI checks are typically run in the cloud without user intervention. If desired, the CI checks can also be run locally using the [CircleCI Local CLI](https://circleci.com/docs/2.0/local-cli/).
