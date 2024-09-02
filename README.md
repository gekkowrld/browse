# browse

A simple web app to view local files in a beautiful way.

To run the app simply navigate to the local cloned directory and run: `go run main.go`
Ensure that you have a `config.ini` file in `$XDG_CONFIG_HOME/browse` or `$HOME/.config/browse` for configuration.

Example configuration:

```ini
[directories]
dirs = ~/code/*

[settings]
preferred_name = Browse!
```

The `*` is matched against all the directories in the specified parent and displays them as standalone dirs, else they are sub dirs.

## view

The files are displayed as they appear in the user file system.
If it is a file, it is displayed (except binaries).
If a directory, then the children are displayed and some details about them.
Syntax highlighting is applied appropriately.

## search

The files are searched recursively through the file system.
The search can be narrowed down to one directory (top most parent) by using `dir:{directory}`.
The results displayed are searched naively line by line void of any context.
If the files structure is deep or big, it may take significantly more to do so.
