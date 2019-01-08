# JSON Response

<a href="https://circleci.com/gh/sylabs/workflows/json-resp"><img src="https://circleci.com/gh/sylabs/json-resp.svg?style=shield&circle-token=48bc85b347052f2de57405ab063b0d8b96c7059d"></a>
<a href="https://app.zenhub.com/workspace/o/sylabs/json-resp/boards"><img src="https://raw.githubusercontent.com/ZenHubIO/support/master/zenhub-badge.png"></a>

The `json-resp` package contains a small set of functions that are used to marshall and unmarshall response data and errors in JSON format.

## Quick Start

Install [dep](https://golang.github.io/dep/docs/installation.html) and the [CircleCI Local CLI](https://circleci.com/docs/2.0/local-cli/). See the [Dependency Management](#dependency-management) and [Continuous Integration](#continuous-integration) sections below for more detail.

Use dep to populate the `vendor/` directory:

```sh
dep ensure
```

To build and test:

```sh
circleci build
```

## Dependency Management

This package uses [dep](https://golang.github.io/dep/) for dependency management. Install it by following the directions for your platform [here](https://golang.github.io/dep/docs/installation.html).

## Continuous Integration

This package uses [CircleCI](https://circleci.com) for Continuous Integration (CI). It runs automatically on commits and pull requests involving a protected branch. All CI checks must pass before a merge to a proected branch can be performed.

The CI checks are typically run in the cloud without user intervention. If desired, the CI checks can also be run locally using the [CircleCI Local CLI](https://circleci.com/docs/2.0/local-cli/).
