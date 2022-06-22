package main

import (
	"cloud.google.com/go/storage"
	"context"
	"google.golang.org/api/option"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
)

func BaseName(filename string) string {
	filename = path.Base(filename)
	var extension = ".tar.gz"
	var name = filename[0 : len(filename)-len(extension)]
	return name
}

func main() {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile("br-blockchains-dev.json"))
	if err != nil {
		panic(err)
	}
	teamDropBucket := client.Bucket("team-drop")

	it := teamDropBucket.Objects(ctx, &storage.Query{Prefix: "master-datadir/datadir-", IncludeTrailingDelimiter: false})
	for {

		snapshot, err := it.Next()
		if err != nil {
			break
		}
		snapshotDir := filepath.Join("datadir", BaseName(snapshot.Name))
		err = os.MkdirAll(snapshotDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
		snapshotObj := teamDropBucket.Object(snapshot.Name)
		println(snapshotObj.ObjectName())
		// Download snapshot, TODO : use aria2 to download snapshots
		snapshotFilePath := filepath.Join(snapshotDir, "snapshot.tar.gz")
		snapshotFile, err := os.Create(snapshotFilePath)
		if err != nil {
			panic(err)
		}
		reader, err := snapshotObj.NewReader(ctx)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(snapshotFile, reader)
		if err != nil {
			panic(err)
		}
		dockerFilePath := filepath.Join(snapshotDir, "Dockerfile")
		f, err := os.Open(filepath.Join(workingDir, "Dockerfile"))
		if err != nil {
			panic(err)
		}
		dockerFile, err := os.Create(dockerFilePath)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(dockerFile, f)
		if err != nil {
			panic(err)
		}
	}
}
