# About

This project is a Golang library for dealing with TOML files.

The library supports v1.1.0 of the TOML specification, which can be found at https://toml.io/en/v1.1.0 or in the [toml-1.1.0-spec](../toml-1.1.0-spec/) directory.

The main value-add this library provides compared to others is the ability to read TOML files, update them in-memory, then write them back to disk, preserving as much whitespace and comments as possible from the original document.

## Current status

At this stage literally nothing exists and we are building it from scratch, meaning much of the above and below is effectively our TODO list.

## Design and implementation

Therefore the API is very much up for discussion, however it should provide idiomatic functions to ergonomically query the in-memory document, as well as iterate/enumerate its various keys and values, including the ability to get the keys of top-level values as well as nested or deeply nested values.

Likewise it should be possible to easily set and delete content from the document, including whitespace and comments.

When new content is added we will need to use heuristics to guess whitespace conventions when the document is written to disk.

- should there be a blank-line before a new table section
- should there be whitespace between the various tokens in an inline table

Additionally it should be possible for the user to set the whitespace rule for new and/or all content when writing the document.

It should be possible to format to bytes and/or work with Go's VFS abstraction.

## Testing

To ensure spec compliance we should use the existing https://github.com/toml-lang/toml-test tool, which provides a large suite of test cases for validating TOML parsers and serializers. We should also add our own test cases to cover our whitespace and comments preservation features work as intended, as well as any other specific APIs we have.

We will also eventually use Go's fuzz testing capabilities to generate random TOML documents and ensure our library can handle them without panicking or otherwise crashing.

## Performance

Performance is not a primary concern, but we should still aim for reasonable efficiency in our implementation, especially when it comes to parsing and serializing large TOML documents, but considering the typical uses of TOML files (configuration) we probably always want to load the entire document into memory, so we don't need to worry about streaming parsing or serialization.

## Dependencies

Ideally it will be zero-dependency, but we may need to use some third-party libraries for things like testing or VFS support. We should be judicious in our choice of dependencies to avoid unnecessary bloat and ensure the library remains easy to use and maintain.

## Documentation

The library source code should be well-documented, with clear examples of how to use the various APIs to read, manipulate, and write TOML documents. This includes the use of runnable+verifiable examples in docs.

The README should provide an overview of the library, its features, and how to get started with it, including installation instructions and basic usage examples.

We could also consider adding a more detailed documentation site hosted on GitHub pages if the library grows in complexity.

## CI/CD

We want to keep the library clean and tidy, linted, tested, vetted, etc on every commit, so we're going to use all the available OSS tools that make sense.
