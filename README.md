[![Release](https://img.shields.io/github/v/release/karlssonerik/go-service-doc)](https://github.com/karlssonerik/go-service-doc/releases/latest)
[![Build Status](https://img.shields.io/endpoint.svg?url=https%3A%2F%2Factions-badge.atrox.dev%2Fkarlssonerik%2Fgo-service-doc%2Fbadge%3Fref%3Dmain&style=flat)](https://actions-badge.atrox.dev/karlssonerik/go-service-doc/goto?ref=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/karlssonerik/go-service-doc)](https://goreportcard.com/report/github.com/karlssonerik/go-service-doc)
[![Coverage Status](https://coveralls.io/repos/github/karlssonerik/go-service-doc/badge.svg?branch=main)](https://coveralls.io/github/karlssonerik/go-service-doc?branch=main)

# go-service-doc

This is a tool to generate a static web site based on Markdown files.

It will:
- convert Markdown files to HTML pages ([more info](#html-page-generator))
- generate a menu based on `#` and `##` headers ([more info](#side-menu-generator))
- add styling similar to what is used by github to display Markdown files

go-service-doc also supports embedding static files, [more info](#embedding-images).

You can find a list of all features [here](#features)

go-service-doc will generate both HTML files to be deployed standalone and a `go` handler, which could be used in your service.
Here you can find a [deployed example](https://karlssonerik.github.io/go-service-doc) of the generated HTML files.

## Usage

### Install

> go install -u github.com/karlssonerik/go-service-doc

### Run

> go-service-doc

#### Flags

- **-s**

  > The Index Markdown filename to use for the base path, defaults to `service.md`.

- **-d**

  > The Source Directory where the markdown files are located, defaults to `docs`.

- **-o**

  > The Output Directory where to write the generated files, defaults to `docs`.

- **-p**

  > Base path to add for the generated documentation, defaults to `/docs`.

### Example

You can find this example with the markdown source files and the generated output in [cmd/example](cmd/example).

To generate the output, the following is executed from [cmd/example](cmd/example).

> go-service-doc -s bars.md -d docs/src -o docs/generated -p /go-service-doc

Example code:

```go
package main

import (
	"log"
	"net/http"

	service_docs "github.com/karlssonerik/go-service-doc/cmd/example/docs/generated"
)

const port = "8080"

func main() {
	server := &http.Server{Addr: ":" + port, Handler: service_docs.Handler()}

	log.Printf("Will start to listen and serve on port %s", port)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("HTTP server ListenAndServe")
	}
}
```

## Features

### HTML Page Generator

It will convert the Markdown files to HTML pages and add CSS similar to the CSS used by github to display Markdown files. The URL for the generated HTML page will be the kebab-case version of the filename excluding the extension, i.e. `monkey_bar.md` will be `/<base_path>/monkey-bar`.

### Side Menu Generator

The Side Menu is generated based on the Markdown Header Elements: `#` and `##`. It will only generate entries for the headers that have a defined Header ID, like: `{#header_id}`.

### Search Engine

The Side Menu features a Search field that can be used to search in all generated pages. The search engine will index content based on Markdown Headers.

### Embedding Images

Files found in the `static` folder will be embedded in the generated go-handler and can be referenced through `<base_path>/static/<file_name>`.

```
<src_directory>
└─ static
   ├─ bars.svg
   ├─ favicon-16x16.png
   └─ favicon.ico
```

#### Supported file extensions:

- .svg
- .png
- .ico

#### How to add an image in Markdown

From [cmd/example](cmd/example/docs/src/bars.md), `![The bars](/go-service-doc/static/bars.svg)`.

### Favicon

If a file called `favicon.ico` is found in the `static` folder, it will be used as the sites favicon.

```
<src_directory>
└─ static
   └─ favicon.ico
```
