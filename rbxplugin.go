package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/anaminus/rbxweb"
	"github.com/anaminus/rbxweb/asset"
	flags "github.com/jessevdk/go-flags"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type Opts struct {
	Help        bool   `short:"h" long:"help"        optional:"true"                                              description:"Shows this help message."`
	Build       bool   `short:"b" long:"build"       optional:"true"                                              description:"Builds the plugin to the output file."`
	Deploy      bool   `short:"d" long:"deploy"      optional:"true"                                              description:"Deploys the results of the build to a plugin asset on the ROBLOX website."`
	Input       string `short:"i" long:"input"       optional:"true" value-name:"[dir]"    default:"."            description:"The directory to process into a plugin."`
	Output      string `short:"o" long:"output"      optional:"true" value-name:"[file]"   default:"./build.rbxm" description:"The rbxm file to output to."`
	Username    string `short:"u" long:"username"    optional:"true" value-name:"[string]"                        description:"The username to use for logging in when creating or deploying."`
	Password    string `short:"p" long:"password"    optional:"true" value-name:"[string]"                        description:"The password to use for logging in when creating or deploying."`
	Asset       int64  `short:"a" long:"asset"       optional:"true" value-name:"[int]"                           description:"The id of the asset to deploy to."`
	Name        string `short:"n" long:"name"        optional:"true" value-name:"[string]"                        description:"The plugin name used when creating or deploying."`
	Description string `short:"m" long:"description" optional:"true" value-name:"[string]"                        description:"The plugin description used when creating or deploying."`
}

var buildResults []byte

func Build(opts Opts) (results io.Reader, err error) {
	if len(buildResults) == 0 {
		results, err := WriteRBXM(opts.Input)
		if err != nil {
			return nil, err
		}
		buildResults = results
	}
	return bytes.NewReader(buildResults), nil
}

func Update(opts Opts, client *http.Client) (err error) {
	results, err := Build(opts)
	if err != nil {
		return err
	}

	info := url.Values{
		"type":    {"Plugin"},
		"assetid": {strconv.FormatInt(opts.Asset, 10)},
	}

	_, err = asset.Upload(client, results, info)

	if opts.Name != "" || opts.Description != "" {
		params := url.Values{
			"__RequestVerificationToken": {},
			"Name":                {},
			"Description":         {},
			"ThumbnailYoutubeUrl": {},
		}
		if opts.Name != "" {
			params.Set("Name", opts.Name)
		}
		if opts.Description != "" {
			params.Set("Description", opts.Description)
		}

		err = rbxweb.DoRawPost(client, `http://www.roblox.com/plugins/`+strconv.FormatInt(opts.Asset, 10)+`/update`, params)
		// rbxweb's error handling needs to be improved
		if err.Error() == "301: Moved Permanently" {
			return nil
		}
		return err
	}

	return err
}

func Create(opts Opts, client *http.Client) (err error) {
	results, err := Build(opts)
	if err != nil {
		return err
	}

	info := url.Values{
		"type":          {"Plugin"},
		"assetid":       {"0"},
		"name":          {opts.Name},
		"description":   {opts.Description},
		"isPublic":      {"True"},
		"genreTypeId":   {"1"},
		"allowComments": {"True"},
	}

	_, err = asset.Upload(client, results, info)
	return err
}

func Deploy(opts Opts) (err error) {
	if opts.Username == "" {
		return errors.New("Deployment requires a username to be specified")
	}

	if opts.Password == "" {
		return errors.New("Deployment requires a password to be specified")
	}

	client := http.DefaultClient
	err = rbxweb.Login(client, opts.Username, opts.Password)
	if err != nil && err != rbxweb.ErrLoggedIn {
		return err
	}

	if opts.Asset == 0 {
		return Create(opts, client)
	} else {
		return Update(opts, client)
	}
}

func assert(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

func main() {
	var opts Opts

	parser := flags.NewParser(&opts, flags.PrintErrors)
	parser.Parse()

	if opts.Help {
		parser.WriteHelp(os.Stderr)
		return
	}

	stat, err := os.Stat(opts.Input)
	if err != nil || !stat.IsDir() {
		assert(errors.New("Building requires an input directory to be specified"))
		return
	}

	modeActive := false
	if opts.Build {
		modeActive = true
		if opts.Output == "" {
			assert(errors.New("Building requires an output file to be specified"))
			return
		}

		results, err := Build(opts)
		assert(err)

		f, err := os.Create(opts.Output)
		assert(err)
		io.Copy(f, results)
		f.Close()
	}

	if opts.Deploy {
		modeActive = true
		err := Deploy(opts)
		assert(err)
	}

	if !modeActive {
		parser.WriteHelp(os.Stderr)
		return
	}
}
