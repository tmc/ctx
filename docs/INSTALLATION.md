# Installation

This document provides instructions for installing ctx, a flexible and extensible tool designed to gather contextual information primarily for use with Language Models (LLMs).

## Quick Start

For most users, the quickest way to install ctx is:

```bash
go install github.com/tmc/ctx/cmd/ctx@latest
```

This command will download and install the latest version of ctx.

## Detailed Installation Instructions

### Prerequisites

- Go 1.16 or later

### Installing from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/tmc/ctx.git
   ```

2. Change to the ctx directory:
   ```bash
   cd ctx
   ```

3. Build and install:
   ```bash
   go install ./cmd/ctx
   ```

### Verifying the Installation

After installation, you can verify that ctx is installed correctly by running:

```bash
ctx --version
```

This should display the version number of ctx.

<details>
<summary>Troubleshooting Installation Issues</summary>

If you encounter any issues during installation, try the following:

1. Ensure your Go installation is up to date:
   ```bash
   go version
   ```

2. Check that your GOPATH is set correctly:
   ```bash
   echo $GOPATH
   ```

3. Make sure the GOPATH/bin directory is in your PATH:
   ```bash
   echo $PATH
   ```

4. If you're still having issues, please open an issue on the GitHub repository with details about your system and the error you're encountering.

</details>

## Installing Plugins

ctx plugins are separate executables that should be installed in a directory in your system's PATH. Most plugins can be installed using the same method as ctx itself:

```bash
go install github.com/tmc/ctx-plugin-name@latest
```

Replace `ctx-plugin-name` with the name of the plugin you want to install.

<details>
<summary>Custom Plugin Installation</summary>

Some plugins may have specific installation instructions or dependencies. Always check the plugin's documentation for detailed installation steps.

If you're developing a custom plugin, you can install it by building it and moving the executable to a directory in your PATH:

```bash
go build -o ctx-myplugin main.go
sudo mv ctx-myplugin /usr/local/bin/
```

</details>

## Upcoming Installation Methods

We are planning to provide additional installation methods in the future:

- Binary releases for various platforms
- Homebrew formula for macOS users
- Package manager support for common Linux distributions

Stay tuned for updates on these installation methods.

## Upgrading ctx

To upgrade to the latest version of ctx, you can use the same command as for installation:

```bash
go install github.com/tmc/ctx/cmd/ctx@latest
```

<details>
<summary>Upgrading from a Pre-1.0 Version</summary>

If you're upgrading from a version prior to 1.0, please note that there may be breaking changes. It's recommended to review the changelog before upgrading and potentially test the new version in a separate environment before using it in production.

</details>

## Uninstalling ctx

To uninstall ctx, you can simply remove the executable from your GOPATH:

```bash
rm $GOPATH/bin/ctx
```

Note that this will not remove any installed plugins or configuration files.

<details>
<summary>Complete Uninstallation</summary>

For a complete uninstallation, including plugins and configuration:

1. Remove the ctx executable:
   ```bash
   rm $GOPATH/bin/ctx
   ```

2. Remove any installed plugins (replace `ctx-plugin-name` with actual plugin names):
   ```bash
   rm $GOPATH/bin/ctx-plugin-name
   ```

3. Remove configuration files:
   ```bash
   rm -rf ~/.config/ctx
   ```

</details>

For any additional installation help or to report issues, please open an issue on the GitHub repository.

## Troubleshooting

- Check system requirements
- Verify PATH configuration
- Ensure correct permissions
