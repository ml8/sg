# `sg` - another shitty static site generator

See [github.com/ml8/ml8.github.io](https://github.com/ml8/ml8.github.io) for an
example.

Input site structure:

* `sg.yaml` -- Site config. Fields defined in `Site`.

* `templates/` -- Directory of html templates. Use Go's
  [`html/template`](https://pkg.go.dev/html/template). Fields available to
  templates are those in `RenderContext` (and transitive references therefrom).

* `pages/` -- Directory of pages. Pages are either raw HTML (that may include
  template code), markdown files that are rendered, or plain files.
    
    * Markdown files must include some frontmatter (fields from `Page`)
      metadata.

To generate the site, supply the input directory containing the site structure
and an output directory to render to. Optionally supply `-w` to watch the input
directory or `-s` to serve a preview locally after rendering (`-w` implies
`-s`):

```
sg -i input_dir -o output_dir
```

`Renderer` can be used directly. `OfflineRenderer` renders a site in one-shot,
while `OnlineRenderer` watches an input directory and renders as files arrive.
