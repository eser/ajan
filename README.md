# 🧢 ajan framework

[![codecov](https://codecov.io/gh/eser/ajan/branch/main/graph/badge.svg?token=w6s3ODtULz)](https://codecov.io/gh/eser/ajan)
[![Build Pipeline](https://github.com/eser/ajan/actions/workflows/build.yml/badge.svg)](https://github.com/eser/ajan/actions/workflows/build.yml)

`ajan` project is designed to unlock Golang's greatest strength—its standard library—by enabling you to harness it with maximum robustness and flexibility. Rather than reinventing the wheel, this project builds upon Golang's core, providing you with a continuously updated, battle-tested foundation. At the same time, it offers flexible structures that let you configure and extend the standard library to meet your unique needs.

## 📂 Components

|         Component         | Description |
| ------------------------- | ----------- |
| [cachefx](./cachefx/)     | Flexible caching solution with support for Redis and other backends |
| [configfx](./configfx/)   | Configuration management with support for multiple sources including environment variables and files |
| [connfx](./connfx/)       | Connection management and registry for databases, caches, and external services |
| [datafx](./datafx/)       | Database access layer supporting Postgres, MySQL, and SQLite |
| [di](./di/)               | Lightweight yet powerful dependency injection container |
| [eventsfx](./eventsfx/)   | Event handling and pub/sub system |
| [grpcfx](./grpcfx/)       | gRPC service integration and utilities |
| [httpclient](./httpclient/) | HTTP client utilities and helpers for external API communication |
| [httpfx](./httpfx/)       | HTTP service framework with routing and middleware support |
| [lib](./lib/)             | Common utilities and shared functionality including network, crypto, and string helpers |
| [logfx](./logfx/)         | Structured logging with pretty-printing and OpenTelemetry support |
| [metricsfx](./metricsfx/) | Metrics collection and monitoring utilities |
| [processfx](./processfx/) | Process and goroutine lifecycle management with graceful shutdown handling |
| [queuefx](./queuefx/)     | Message queue integration with RabbitMQ support |
| [results](./results/)     | Structured error handling and result types |
| [sampleapp](./sampleapp/) | Complete example application demonstrating ajan framework usage and best practices |
| [types](./types/)         | Custom data types including metric types with unit suffix support |

## 🙋🏻 FAQ

### Want to report a bug or request a feature?

If you're going to report a bug or request a new feature, please ensure first
that you comply with the conditions found under
[@eser/directives](https://github.com/eser/ajan/blob/dev/pkg/directives/README.md).
After that, you can report an issue or request using
[GitHub Issues](https://github.com/eser/ajan/issues). Thanks in advance.

### Want to contribute?

It is publicly open for any contribution from the community. Bug fixes, new
features and additional components are welcome.

If you're interested in becoming a contributor and enhancing the ecosystem,
please start by reading through our [CONTRIBUTING.md](./.github/CONTRIBUTING.md).

If you're not sure where to begin, take a look at the
[issues](https://github.com/eser/ajan/issues) labeled `good first issue` and
`help wanted`. Reviewing closed issues can also give you a sense of the types of
contributions we're looking for and you can tackle.

If you're already an experienced OSS contributor, let's take you to the shortest
path: To contribute to the codebase, just fork the repo, push your changes to
your fork, and then submit a pull request.

### Requirements

- Golang 1.24 or higher (https://go.dev/)

### Versioning

This project follows [Semantic Versioning](https://semver.org/). For the
versions available, see the
[tags on this repository](https://github.com/eser/ajan/tags).

### License

This project is licensed under the Apache 2.0 License. For further details,
please see the [LICENSE](LICENSE) file.

### To support the project...

[Visit my GitHub Sponsors profile at github.com/sponsors/eser](https://github.com/sponsors/eser)
