package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bufbuild/protoplugin"

	"github.com/k0kubun/pp"
	"google.golang.org/protobuf/types/descriptorpb"
)

//go:generate go run ./generator

//go:embed  templates
var root embed.FS

func AllGoFiles() (embed.FS, error) {
	return root, nil
}

func main() {
	protoplugin.Main(protoplugin.HandlerFunc(handle))
}

func handle(
	ctx context.Context,
	e protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) error {
	// Set the flag indicating that we support proto3 optionals. We don't even use them in this
	// plugin, but protoc will error if it encounters a proto3 file with an optional but the
	// plugin has not indicated it will support it.
	responseWriter.SetFeatureProto3Optional()
	responseWriter.SetFeatureSupportsEditions(descriptorpb.Edition_EDITION_2023, descriptorpb.Edition_EDITION_2024)

	fileDescriptors, err := request.FileDescriptorsToGenerate()
	if err != nil {
		return err
	}

	params := parseParams(request.Parameter())

	defaultBufValidateFile := "buf/validate/validate.proto"

	if params["buf_validate_file"] != "" {
		defaultBufValidateFile = params["buf_validate_file"]
	}

	pp.Fprintf(os.Stderr, "defaultBufValidateFile: %+v\n", defaultBufValidateFile)

	pp.Fprintf(os.Stderr, "params: %+v\n", params)

	pp.Fprintf(os.Stderr, "paramString: %+v\n", request.Parameter())

	for _, fileDescriptor := range fileDescriptors {

		if !strings.HasSuffix(fileDescriptor.Path(), defaultBufValidateFile) {
			continue
		}

		desc, ok := fileDescriptor.Options().(*descriptorpb.FileOptions)
		if !ok {
			return fmt.Errorf("file descriptor is not a FileDescriptorProto")
		}

		gpkg := desc.GetGoPackage()

		pp.Fprintf(os.Stderr, "fileDescriptor.Path(): %+v\n", fileDescriptor.Path())
		pp.Fprintf(os.Stderr, "fileDescriptor.Name(): %+v\n", fileDescriptor.Name())
		pp.Fprintf(os.Stderr, "fileDescriptor.Package(): %+v\n", fileDescriptor.Package())
		pp.Fprintf(os.Stderr, "fileDescriptor.FullName(): %+v\n", fileDescriptor.FullName())

		rootPath := strings.TrimSuffix(fileDescriptor.Path(), "validate.proto")

		pp.Fprintf(os.Stderr, "gpkg: %+v\n", gpkg)

		tmpl := template.New("root")

		tmpl.Delims("[[[[[[[[", "]]]]]]]]")

		tmpl, err = tmpl.ParseFS(root, "templates/*.tmpl")
		if err != nil {
			return err
		}

		for _, file := range tmpl.Templates() {
			if file.Name() == "root" {
				continue
			}

			fileName := strings.TrimSuffix(file.Name(), ".tmpl")

			// fileName = strings.TrimPrefix(fileName, "gen/protovalidate/")

			fileName = strings.ReplaceAll(fileName, "___", "/")

			fmt.Fprintf(os.Stderr, "file: %+v\n", fileName)

			var buf bytes.Buffer
			err = file.Execute(&buf, map[string]any{"GoPackageOption": gpkg})
			if err != nil {
				return err
			}

			responseWriter.AddFile(
				filepath.Join(rootPath, "protovalidate", fileName),
				buf.String(),
			)
		}

		// err = addAllGoFiles(root, ".", gpkg, rootPath, responseWriter)
		// if err != nil {
		// 	return err
		// }

	}

	return nil
}

func parseParams(params string) map[string]string {
	paramMap := map[string]string{}

	for _, param := range strings.Split(params, ",") {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			continue
		}

		paramMap[parts[0]] = parts[1]
	}

	return paramMap
}

// func addAllGoFiles(fs embed.FS, path string, goPackage string, rootPath string, responseWriter protoplugin.ResponseWriter) error {
// 	entries, err := fs.ReadDir(path)
// 	if err != nil {
// 		return err
// 	}

// 	if path == "." {
// 		path = ""
// 	}

// 	for _, entry := range entries {
// 		fmt.Fprintf(os.Stderr, "%s %v %s\n", entry.Name(), entry.IsDir(), path)

// 		if entry.IsDir() {
// 			err = addAllGoFiles(fs, filepath.Join(path, entry.Name()), goPackage, rootPath, responseWriter)
// 			if err != nil {
// 				return err
// 			}
// 			continue
// 		}

// 		if strings.HasSuffix(entry.Name(), "_test.go") {
// 			continue
// 		}

// 		file, err := fs.Open(filepath.Join(path, entry.Name()))
// 		if err != nil {
// 			return err
// 		}

// 		bytes, err := io.ReadAll(file)
// 		if err != nil {
// 			return err
// 		}

// 		out := string(bytes)

// 		responseWriter.AddFile(
// 			filepath.Join(rootPath, "protovalidate", path, entry.Name()),
// 			out,
// 		)

// 	}

// 	return nil
// }
