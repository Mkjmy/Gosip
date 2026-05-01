# buddystore

![Screenshot](screenshot)


buddystore is a small CLI-based app store / package manager written in Go.

it exists mostly because me and my friends like building our own tools instead of relying on existing ones. this is just our way of managing configs, scripts, and small projects in one place.

## overview

the system is simple:

* a GitHub-based registry (JSON)
* a CLI tool to browse and install apps from it

nothing fancy, just something that works for us.

## features

* **interactive interface**
  basic terminal UI with keyboard navigation

* **install from git repos**
  apps are pulled directly from GitHub

* **audit mode**
  see what an app does before running it

* **community apps**
  apps can be submitted via Issues and added to the registry

## install

```bash
go build -o buddy-store
./buddy-store init
```

this may modify your `.bashrc` / `.zshrc` to update PATH.

## usage

```bash
buddy-store
buddy-store update
buddy-store uninstall <name>
```

## notes

* this project is built and tested on Linux only
* **it will most likely not work on Windows**
* no guarantees, no compatibility layer (for now)

## philosophy

we just like building and using our own stuff.
this isn’t meant to be perfect or widely used — just useful for a small circle.

## disclaimer

use at your own risk, especially with community apps.

# Gosip
