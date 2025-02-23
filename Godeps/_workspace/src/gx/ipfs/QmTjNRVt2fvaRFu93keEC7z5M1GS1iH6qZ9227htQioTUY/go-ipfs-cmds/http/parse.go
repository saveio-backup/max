package http

import (
	"context"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"strconv"
	"strings"

	cmds "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTjNRVt2fvaRFu93keEC7z5M1GS1iH6qZ9227htQioTUY/go-ipfs-cmds"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit/files"
)

// parseRequest parses the data in a http.Request and returns a command Request object
func parseRequest(ctx context.Context, r *http.Request, root *cmds.Command) (*cmds.Request, error) {
	if r.URL.Path[0] == '/' {
		r.URL.Path = r.URL.Path[1:]
	}

	var (
		stringArgs []string
		pth        = strings.Split(r.URL.Path, "/")
		getPath    = pth[:len(pth)-1]
	)

	cmd, err := root.Get(getPath)
	if err != nil {
		// 404 if there is no command at that path
		return nil, ErrNotFound
	}

	sub := cmd.Subcommands[pth[len(pth)-1]]

	if sub == nil {
		if cmd.Run == nil {
			return nil, ErrNotFound
		}

		// if the last string in the path isn't a subcommand, use it as an argument
		// e.g. /objects/Qabc12345 (we are passing "Qabc12345" to the "objects" command)
		stringArgs = append(stringArgs, pth[len(pth)-1])
		pth = pth[:len(pth)-1]
	} else {
		cmd = sub
	}

	opts, stringArgs2 := parseOptions(r)
	optDefs, err := root.GetOptions(pth)
	if err != nil {
		return nil, err
	}
	for k, v := range opts {
		if optDef, ok := optDefs[k]; ok {
			name := optDef.Names()[0]
			if k != name {
				opts[name] = v
				delete(opts, k)
			}
		}
	}
	// default to setting encoding to JSON
	if _, ok := opts[cmds.EncLong]; !ok {
		opts[cmds.EncLong] = cmds.JSON
	}

	stringArgs = append(stringArgs, stringArgs2...)

	// count required argument definitions
	numRequired := 0
	for _, argDef := range cmd.Arguments {
		if argDef.Required {
			numRequired++
		}
	}

	// count the number of provided argument values
	valCount := len(stringArgs)

	args := make([]string, valCount)

	valIndex := 0
	requiredFile := ""
	for _, argDef := range cmd.Arguments {
		// skip optional argument definitions if there aren't sufficient remaining values
		if valCount-valIndex <= numRequired && !argDef.Required {
			continue
		} else if argDef.Required {
			numRequired--
		}

		if argDef.Type == cmdkit.ArgString {
			if argDef.Variadic {
				for _, s := range stringArgs {
					args[valIndex] = s
					valIndex++
				}
				valCount -= len(stringArgs)

			} else if len(stringArgs) > 0 {
				args[valIndex] = stringArgs[0]
				stringArgs = stringArgs[1:]
				valIndex++

			} else {
				break
			}
		} else if argDef.Type == cmdkit.ArgFile && argDef.Required && len(requiredFile) == 0 {
			requiredFile = argDef.Name
		}
	}

	// create cmds.File from multipart/form-data contents
	contentType := r.Header.Get(contentTypeHeader)
	mediatype, _, _ := mime.ParseMediaType(contentType)

	var f files.File
	if mediatype == "multipart/form-data" {
		reader, err := r.MultipartReader()
		if err != nil {
			return nil, err
		}

		f = &files.MultipartFile{
			Mediatype: mediatype,
			Reader:    reader,
		}
	}

	// if there is a required filearg, error if no files were provided
	if len(requiredFile) > 0 && f == nil {
		return nil, fmt.Errorf("File argument '%s' is required", requiredFile)
	}

	req, err := cmds.NewRequest(ctx, pth, opts, args, f, root)
	if err != nil {
		return nil, err
	}

	err = cmd.CheckArguments(req)
	if err != nil {
		return nil, err
	}

	err = req.FillDefaults()
	return req, err
}

func parseOptions(r *http.Request) (map[string]interface{}, []string) {
	opts := make(map[string]interface{})
	var args []string

	query := r.URL.Query()
	for k, v := range query {
		if k == "arg" {
			args = v
		} else {

			opts[k] = v[0]
		}
	}

	return opts, args
}

// parseResponse decodes a http.Response to create a cmds.Response
func parseResponse(httpRes *http.Response, req *cmds.Request) (cmds.Response, error) {
	res := &Response{
		res: httpRes,
		req: req,
		rr:  &responseReader{httpRes},
	}

	lengthHeader := httpRes.Header.Get(extraContentLengthHeader)
	if len(lengthHeader) > 0 {
		length, err := strconv.ParseUint(lengthHeader, 10, 64)
		if err != nil {
			return nil, err
		}
		res.length = length
	}

	contentType := httpRes.Header.Get(contentTypeHeader)
	contentType = strings.Split(contentType, ";")[0]

	encType, found := MIMEEncodings[contentType]
	if found {
		makeDec, ok := cmds.Decoders[encType]
		if ok {
			res.dec = makeDec(res.rr)
		}
	}

	// If we ran into an error
	if httpRes.StatusCode >= http.StatusBadRequest {
		e := &cmdkit.Error{}

		switch {
		case httpRes.StatusCode == http.StatusNotFound:
			// handle 404s
			e.Message = "Command not found."
			e.Code = cmdkit.ErrClient
		case contentType == plainText:
			// handle non-marshalled errors
			mes, err := ioutil.ReadAll(res.rr)
			if err != nil {
				return nil, err
			}
			e.Message = string(mes)
			e.Code = cmdkit.ErrNormal
		default:
			// handle marshalled errors
			err := res.dec.Decode(&e)
			if err != nil {
				return nil, err
			}
		}

		res.initErr = e
	}

	return res, nil
}
