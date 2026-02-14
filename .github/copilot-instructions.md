# About

This project is a Golang library for dealing with TOML files.

The library supports v1.1.0 of the TOML specification, which can be found at https://toml.io/en/v1.1.0 or in the [toml-1.1.0-spec](../toml-1.1.0-spec/) directory.

The main value-add this library provides compared to others is the ability to read TOML files, update them in-memory, then write them back to disk, preserving as much whitespace and comments as possible from the original document.

## Current status

The library is in the early stages of development, with the basic structure and API still being designed and implemented. The core parsing and serialization logic is being developed, along with the in-memory representation of TOML documents.

## Design and implementation

Thus the API is very much up for discussion and change, however it should provide idiomatic functions to ergonomically query the in-memory document, as well as iterate/enumerate its various keys and values, including the ability to get the keys of top-level values as well as nested or deeply nested values.

Likewise it should be possible to easily set, delete, insert and append content in the document, including whitespace and comments.

When new content is added we may need to use heuristics to guess whitespace conventions when the document is written to disk.

- should there be a blank-line before a new table section
- should there be whitespace between the various tokens in an inline table

Additionally it should be possible for the user to set the whitespace rules for new and/or all content when writing the document, so the document is consistently formatted.

It should be possible to format to bytes and/or work with Go's VFS abstraction.

## Usage

The library should be easy to use, with a simple API for reading, manipulating, and writing TOML documents.

The README should provide clear examples of how to use the library for common tasks, such as reading a TOML file into memory, updating values, and writing the updated document back to disk.

When reporting invalid TOML errors, the library should provide as much context as possible about the error, including the line and column number where the error occurred, and a helpful message describing the nature of the error.

## Testing

To ensure spec compliance we are working through test compliance with the https://github.com/toml-lang/toml-test tool, which provides a large suite of test cases for validating TOML parsers and serializers.

We will also need our own test cases to cover our whitespace and comments preservation features work as intended, as well as any other specific APIs we have.

We will also eventually use Go's fuzz testing capabilities to generate random TOML documents and ensure our library can handle them without panicking or otherwise crashing.

## Performance

Performance is not a primary concern, but we should still aim for reasonable efficiency in our implementation, especially when it comes to parsing and serializing large TOML documents, but considering the typical uses of TOML files (configuration) we probably always want to load the entire document into memory, so we don't need to worry about streaming parsing or serialization.

## Dependencies

Ideally it will be zero-dependency, but we may need to use some third-party libraries for things like testing or VFS support. We should be judicious in our choice of dependencies to avoid unnecessary bloat and ensure the library remains easy to use and maintain.

## Documentation

The library source code should be well-documented, with clear examples of how to use the various APIs to read, manipulate, and write TOML documents. This includes the use of runnable+verifiable examples in docs.

The README should provide an overview of the library, its features, and how to get started with it, including installation instructions and basic usage examples.

We could also consider adding a more detailed documentation site hosted on GitHub pages if the library grows in complexity.

## Go binary tools

Many Go binary tools suggest installing via source `go install <tool>@latest`, but we prefer to add them as a Go tool (ie, `go get -tool xxx` then `go tool xxx`) since this is simpler for other developers (with fewer initial setup scripts), keeps versioning consistent, avoids polluting the global Go binary space.

## Typical tasks

We're using `Task` for all of the project tasks like build, test, lint, etc. This is a simple task runner that allows us to define tasks in a `Taskfile.yml` and run them with `task <task-name>`. This helps keep our development workflow consistent and easy to use.

Use `task tasks` to see the available tasks and their descriptions or read the `Taskfile.yml`; it's quite short and easy to grok.

Try to avoid using `go run`, `go test` or `go fmt` directly, and instead use the appropriate `task` commands, which will ensure that all the necessary setup and configuration is done for you.

## Agentic development

All changes made by agents should pass the various tasks in `Taskfile.yml` to ensure that the code is linted, tested, formatted, etc before being merged into the main branch. This helps maintain code quality and consistency across the project, and avoids a large number of lint or other issues after a prolonged period of agentic development.

## CI/CD

We want to keep the library clean and tidy, linted, tested, vetted, etc on every commit, so we're going to use all the available OSS tools that make sense.
