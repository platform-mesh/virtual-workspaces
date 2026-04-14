# Contributing to virtual-workspaces

## General Remarks

Contributions to this project are welcome.

Before contributing:

1. Follow the project [Code of Conduct](https://github.com/platform-mesh/.github/blob/main/CODE_OF_CONDUCT.md).
2. Be prepared to accept the Developer Certificate of Origin (DCO) during the pull request process.
3. If you plan a larger feature or architectural change, open an issue first to confirm it fits the project direction.
4. If you use generative AI while contributing, follow the org-wide [guideline for AI-generated code contributions](https://github.com/platform-mesh/.github/blob/main/CONTRIBUTING_USING_GENAI.md).

## How to Contribute

1. Fork the repository and create a branch from `main`.
2. Make your changes.
3. Add or update tests when behavior changes.
4. Verify your changes locally.
5. Open a pull request and address review feedback.

## Development

Prerequisites: Go, Docker, kubectl, and [Task](https://taskfile.dev). For local certificate and startup details, see [README.md](README.md).

Key commands:

- `go test ./...` — run tests
- `go fmt ./...` — format code
- `task manifests` — regenerate CRDs
- `task generate` — regenerate API objects and resource output after API changes
- `task docker-build` — build the container image
- `task docker:kind` — load the image into kind and restart the deployment
- `go run ./main.go start` — run the service locally

## Pull Requests

- Keep pull requests focused and easy to review.
- Update documentation when APIs, behavior, or local startup requirements change.
- Make sure local verification passes before opening or updating the PR.
- If you change API types or CRD shape, run `task generate` and include the generated output in the same PR.

## License

By contributing, you agree that your contributions will be licensed under the [Apache-2.0 License](LICENSE).
