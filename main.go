package main

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/option"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
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
	var wg sync.WaitGroup
	for {
		snapshot, err := it.Next()
		if err != nil {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			generateDockerfile(snapshot, teamDropBucket, ctx, workingDir)
		}()

	}
	wg.Wait()
}

func generateDockerfile(snapshot *storage.ObjectAttrs, teamDropBucket *storage.BucketHandle, ctx context.Context, workingDir string) {
	snapshotDir := filepath.Join("datadir", BaseName(snapshot.Name))
	err := os.MkdirAll(snapshotDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	snapshotObj := teamDropBucket.Object(snapshot.Name)
	println(snapshotObj.ObjectName())
	// Download snapshot, TODO : use aria2 to download snapshots
	snapshotFilePath := filepath.Join(snapshotDir, "snapshot.tar.gz")
	snapshotFile, err := os.Create(snapshotFilePath)
	defer snapshotFile.Close()
	if err != nil {
		panic(err)
	}
	reader, err := snapshotObj.NewReader(ctx)
	defer reader.Close()
	if err != nil {
		panic(err)
	}
	fmt.Println("Downloading Snapshot ", BaseName(snapshot.Name))
	_, err = io.Copy(snapshotFile, reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("Downloaded Snapshot ", BaseName(snapshot.Name))
	dockerFilePath := filepath.Join(snapshotDir, "Dockerfile")
	f, err := os.Open(filepath.Join(workingDir, "Dockerfile"))
	defer f.Close()
	if err != nil {
		panic(err)
	}
	dockerFile, err := os.Create(dockerFilePath)
	defer dockerFile.Close()
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(dockerFile, f)
	fmt.Println("Created Dockerfile for Snapshot ", BaseName(snapshot.Name))
	if err != nil {
		panic(err)
	}

	fmt.Println("Finished ", BaseName(snapshot.Name))
}
