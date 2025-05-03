# Scaled MCP Documentation Site

This directory contains the Hugo-based documentation site for the Scaled MCP project. The site is automatically built and deployed to GitHub Pages at [scaledmcp.com](https://scaledmcp.com) using GitHub Actions.

## Local Development

To run the documentation site locally:

1. Install Hugo (extended version 0.147.1 or later):
   ```bash
   # macOS
   brew install hugo
   
   # Linux
   # See https://gohugo.io/installation/linux/
   ```

2. Start the local development server:
   ```bash
   cd scaledmcp-site
   hugo server
   ```

3. View the site at http://localhost:1313

## Adding or Updating Content

The documentation content is organized as follows:

- `content/_index.md` - Homepage
- `content/docs/` - Main documentation sections
  - `content/docs/getting-started/` - Getting started guides
  - `content/docs/concepts/` - Core concepts documentation
  - `content/docs/examples/` - Example implementations
  - `content/docs/reference/` - API reference
- `content/about/` - About the project
- `content/menu/index.md` - Navigation menu structure

To add a new page:

1. Create a new Markdown file in the appropriate directory
2. Add front matter at the top of the file:
   ```yaml
   ---
   title: "Your Page Title"
   weight: 10  # Controls ordering in the section
   ---
   ```
3. Add your content using Markdown

## Deployment

The site is automatically deployed to GitHub Pages when changes are pushed to the main branch. The GitHub Actions workflow in `.github/workflows/hugo-deploy.yml` handles the build and deployment process.

## Custom Domain

The site is configured to use the custom domain [scaledmcp.com](https://scaledmcp.com) via the `static/CNAME` file. DNS for this domain should be configured to point to GitHub Pages.

## Theme

This site uses the [Hugo Book theme](https://github.com/alex-shpak/hugo-book) for documentation. See the theme documentation for advanced customization options.
