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
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
)

type Opts struct {
	Help        bool   `short:"h" long:"help"        optional:"true"                                              description:"Shows this help message."`
	Build       bool   `short:"b" long:"build"       optional:"true"                                              description:"Builds the plugin to the output file."`
	Deploy      bool   `short:"d" long:"deploy"      optional:"true"                                              description:"Deploys the results of the build to a plugin asset on the ROBLOX website."`
	Create      bool   `short:"c" long:"create"      optional:"true"                                              description:"Creates a new plugin asset on the ROBLOX website."`
	Input       string `short:"i" long:"input"       optional:"true" value-name:"[dir]"    default:"."            description:"The directory to process into a plugin."`
	Output      string `short:"o" long:"output"      optional:"true" value-name:"[file]"   default:"./build.rbxm" description:"The rbxm file to output to."`
	Username    string `short:"u" long:"username"    optional:"true" value-name:"[string]"                        description:"The username to use for logging in when creating or deploying."`
	Password    string `short:"p" long:"password"    optional:"true" value-name:"[string]"                        description:"The password to use for logging in when creating or deploying."`
	Asset       int64  `short:"a" long:"asset"       optional:"true" value-name:"[int]"                           description:"The id of the asset to deploy to."`
	Name        string `short:"n" long:"name"        optional:"true" value-name:"[string]" default:"Plugin"       description:"The plugin name used when creating or deploying."`
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

func Deploy(opts Opts) (err error) {
	if opts.Username == "" {
		return errors.New("Deployment requires a username to be specified")
	}

	if opts.Password == "" {
		return errors.New("Deployment requires a password to be specified")
	}

	if opts.Asset == 0 {
		return errors.New("Deployment requires an asset id to be specified")
	}

	results, err := Build(opts)
	if err != nil {
		return err
	}

	client := http.DefaultClient
	if client.Jar == nil {
		client.Jar, _ = cookiejar.New(&cookiejar.Options{})
	}
	err = rbxweb.Login(client, opts.Username, opts.Password)
	if err != nil && err != rbxweb.ErrLoggedIn {
		return err
	}

	info := url.Values{
		"type":    {"Plugin"},
		"assetid": {strconv.FormatInt(opts.Asset, 10)},
		//"name":          {opts.Name},
		//"description":   {opts.Description},
		//"isPublic":      {"True"},
		//"genreTypeId":   {"1"},
		//"allowComments": {"True"},
	}

	_, err = asset.Upload(client, results, info)
	return
}

func Create(opts Opts) (err error) {
	if opts.Username == "" {
		return errors.New("Creating requires a username to be specified")
	}

	if opts.Password == "" {
		return errors.New("Creating requires a password to be specified")
	}

	results, err := Build(opts)
	if err != nil {
		return err
	}

	client := http.DefaultClient
	err = rbxweb.Login(client, opts.Username, opts.Password)
	if err != nil && err != rbxweb.ErrLoggedIn {
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
	return
}

func assert(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

func isInCharacterRange(r rune) (inrange bool) {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xDF77 ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
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

	if opts.Build {
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

	if opts.Create {
		err := Create(opts)
		assert(err)
	}

	if opts.Deploy {
		err := Deploy(opts)
		assert(err)
	}
}
