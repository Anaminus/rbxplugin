# rbxplugin

`rbxplugin` is a command-line tool for building and deploying
ROBLOX plugins.

## Usage

rbxplugin has two available modes: **build** and **deploy**. Both modes may be
activated at the same time.

### Build mode

This mode can be activated with one of the following flags:

	--build
	-b

Build mode is used to generate a plugin file based on the contents of a
directory. This directory can be specified with one of the following options:

	--input [dir]
	--i [dir]

If unspecified, the input defaults to the working directory.

The file to output to may also be specified with one of the following options:

	--output [file]
	--o [file]

If unspecified, this defaults to `build.rbxm` in the working directory.

As mentioned, build mode converts a directory to a rbxm file. This involves
creating ROBLOX objects corresponding to each file or directory. For files,
the Name property of the object becomes the name of the file, without the
extension.

The following rules are considered:

- A directory is converted to a Backpack object, whose Name is the name of the
  directory.
- A file with a `.lua` extension is converted to a Script object.
- A lua file (`.lua`) with a `.module` suffix is converted to a ModuleScript
  object (e.g. `ModuleName.module.lua`).
- A ROBLOX model file (`.rbxm`) will have its contents included directly
  (assuming the contents are valid).
- Anything else is converted to a disabled Script, whose contents are enclosed
  in a block comment.

There is an exception for directories. If a directory shares its name with a
file, minus the extension, then a Backpack object will *not* be created.
Instead, the contents of the directory will be inserted as a child of the
object that corresponds to the matched file. For example, consider the
following directory:

	plugin             (directory)
	    foobar.lua     (file)
	    foobar         (directory)
	        foo.txt    (file)
	        bar.txt    (file)

This will generate a file with the following contents:

	plugin         (Backpack)
	    foobar     (Script)
	        foo    (Script)
	        bar    (Script)

This also works with model files, with an exception. As usual, the directory
must match the name of the file. However, since a model file may contain
multiple objects, the contents of the directory will be inserted as a child of
the first object in the model file.

### Deploy mode

This mode can be activated with one of the following flags:

	--deploy
	-d

Deploy mode is used to create or update a plugin on the ROBLOX website.
Deployment will build the input directory, then upload the results.

Uploading a plugin requires a ROBLOX user account. The following options are
necessary for authentication:

- `--username` (`-u`): The account's username.
- `--password` (`-p`): The account's password.

To update a plugin, an asset id must be specified:

	--asset [int]
	-- a [int]

To create a new plugin, don't specify an asset id.

The `--name` (`-n`) and `--description` (`-m`) options will specify the name
and description of the plugin, respectively. These work both for creating or
updating a plugin.

## Installing

[Install Go](http://golang.org/doc/install)

Install rbxplugin:

	go get github.com/anaminus/rbxplugin
	go install github.com/anaminus/rbxplugin
